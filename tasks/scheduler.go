package tasks

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/utils/tracing"
	"github.com/tendermint/tendermint/abci/types"
	"sync"
	"sync/atomic"
)

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error)
}

type scheduler struct {
	deliverTx          func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)
	workers            int
	multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
	tracingInfo        *tracing.Info
	allTasks           []*TxTask
	executeCh          chan func()
	validateCh         chan func()
}

// NewScheduler creates a new scheduler
func NewScheduler(workers int, tracingInfo *tracing.Info, deliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)) Scheduler {
	return &scheduler{
		workers:     workers,
		deliverTx:   deliverTxFunc,
		tracingInfo: tracingInfo,
	}
}

func (s *scheduler) ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error) {
	// initialize mutli-version stores
	s.initMultiVersionStore(ctx)
	// prefill estimates
	s.PrefillEstimates(reqs)
	tasks := toTasks(ctx, reqs)
	s.allTasks = tasks

	workers := s.workers
	if s.workers < 1 {
		workers = len(tasks)
	}

	// initialize scheduler queue
	queue := NewTaskQueue(tasks)

	// send all tasks to queue
	queue.ExecuteAll()

	wg := sync.WaitGroup{}
	wg.Add(workers)
	count := atomic.Int32{}

	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()

			for {

				// check if all tasks are complete AND not running anything
				if queue.IsCompleted() {
					queue.Close()
				}

				task, anyTasks := queue.NextTask()
				if !anyTasks {
					return
				}

				task.LockTask()
				if tt, ok := task.PopTaskType(); ok {
					count.Add(1)
					s.processTask(ctx, tt, worker, task, queue)
				} else {
					TaskLog(task, "NONE FOUND...SKIPPING")
				}
				task.UnlockTask()
			}

		}(i)
	}

	wg.Wait()

	fmt.Println("count", count.Load())

	for _, mv := range s.multiVersionStores {
		mv.WriteLatestToStore()
	}
	return collectResponses(tasks), nil
}

func (s *scheduler) processTask(ctx sdk.Context, taskType TaskType, w int, t *TxTask, queue Queue) bool {
	switch taskType {
	case TypeValidation:
		TaskLog(t, fmt.Sprintf("TypeValidation (worker=%d)", w))

		s.validateTask(ctx, t)

		// check the outcome of validation and do things accordingly
		switch t.status {
		case statusValidated:
			// task is possibly finished (can be re-validated by others)
			TaskLog(t, "*** VALIDATED ***")
			// informs queue that it's complete (any subsequent submission for idx unsets this)
			queue.FinishTask(t.Index)
			return true
		case statusWaiting:
			// task should be re-validated (waiting on others)
			// how can we wait on dependencies?
			TaskLog(t, "waiting/executed...revalidating")
			if queue.DependenciesFinished(t.Index) {
				queue.Execute(t.Index)
			}
		case statusInvalid:
			TaskLog(t, "invalid (re-executing, re-validating > tx)")
			queue.Execute(t.Index)
		default:
			TaskLog(t, "unexpected status")
			panic("unexpected status ")
		}

	case TypeExecution:
		t.ResetForExecution()
		TaskLog(t, fmt.Sprintf("TypeExecution (worker=%d)", w))

		s.executeTask(t)

		if t.IsStatus(statusAborted) {
			queue.ReExecute(t.Index)
		} else {
			TaskLog(t, fmt.Sprintf("FINISHING task EXECUTION (worker=%d, incarnation=%d)", w, t.Incarnation))
			queue.FinishExecute(t.Index)
			//TODO: speed this up, too slow to do every time
			queue.ValidateLaterTasks(t.Index)
		}

	default:
		TaskLog(t, "unexpected type")
		panic("unexpected type")
	}
	return false
}
