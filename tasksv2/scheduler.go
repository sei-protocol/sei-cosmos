package tasksv2

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/utils/tracing"
	"github.com/tendermint/tendermint/abci/types"
)

type SchedulerHandler func(ctx sdk.Context, req types.RequestDeliverTx, tx sdk.Tx, checksum [32]byte) (res types.ResponseDeliverTx)

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error)
}

type scheduler struct {
	deliverTx          SchedulerHandler
	workers            int
	multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
	tracingInfo        *tracing.Info
	tasks              []*TxTask
	executeCh          chan func()
	validateCh         chan func()
	timer              *Timer
}

// NewScheduler creates a new scheduler
func NewScheduler(workers int, tracingInfo *tracing.Info, deliverTxFunc SchedulerHandler) Scheduler {
	return &scheduler{
		workers:     workers,
		deliverTx:   deliverTxFunc,
		tracingInfo: tracingInfo,
		timer:       NewTimer("Scheduler"),
	}
}

func (s *scheduler) initSchedulerAndQueue(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) (Queue, int) {
	// initialize mutli-version stores
	s.initMultiVersionStore(ctx)
	// prefill estimates
	s.PrefillEstimates(reqs)
	tasks := toTasks(ctx, reqs)
	s.tasks = tasks

	workers := s.workers
	if s.workers < 1 {
		workers = len(tasks)
	}

	// initialize scheduler queue
	queue := NewTaskQueue(tasks, workers)

	// send all tasks to queue
	go queue.ExecuteAll()

	return queue, workers
}

func (s *scheduler) ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error) {
	if len(reqs) == 0 {
		return []types.ResponseDeliverTx{}, nil
	}

	var results []types.ResponseDeliverTx

	queue, workers := s.initSchedulerAndQueue(ctx, reqs)
	wg := sync.WaitGroup{}
	wg.Add(workers)
	var activeWorkers int32

	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()

			for {
				if atomic.LoadInt32(&activeWorkers) == 0 {
					if queue.IsCompleted() {
						queue.Close()
					}
				}

				task, anyTasks := queue.NextTask(worker)
				atomic.AddInt32(&activeWorkers, 1)

				if !anyTasks {
					return
				}

				// threadsafe acquisition of task type, avoids dual-execution
				if tt, ok := task.PopTaskType(); ok {
					s.processTask(ctx, tt, worker, task, queue)
				}
				atomic.AddInt32(&activeWorkers, -1)
			}

		}(i)
	}

	wg.Wait()

	// if there are any tasks left, process them sequentially
	// this can happen with high conflict rates and high incarnations
	s.validateAll(ctx)

	for _, mv := range s.multiVersionStores {
		mv.WriteLatestToStore()
	}
	results = collectResponses(s.tasks)

	return results, nil
}

func (s *scheduler) processTask(ctx sdk.Context, taskType TaskType, w int, t *TxTask, queue Queue) {
	switch taskType {
	case TypeValidation:
		TaskLog(t, fmt.Sprintf("TypeValidation (worker=%d)", w))

		s.validateTask(ctx, t)

		// check the outcome of validation and do things accordingly
		switch t.status {
		case statusValidated:
			// task is finished (but can be re-validated by others)
			TaskLog(t, "*** VALIDATED ***")
			// informs queue that it's complete (counts towards overall completion)
			queue.FinishTask(t.AbsoluteIndex)

		case statusWaiting:
			// task should be re-validated (waiting on others)
			TaskLog(t, "waiting, executing again")
			queue.Execute(t.AbsoluteIndex)

		case statusInvalid:
			TaskLog(t, "invalid (re-executing, re-validating > tx)")
			queue.ValidateLaterTasks(t.AbsoluteIndex)
			queue.Execute(t.AbsoluteIndex)

		default:
			TaskLog(t, "unexpected status")
			panic("unexpected status")
		}

	case TypeExecution:
		t.ResetForExecution()
		TaskLog(t, fmt.Sprintf("TypeExecution (worker=%d)", w))

		s.executeTask(t)

		switch t.status {
		case statusAborted:
			parent := s.tasks[t.Abort.DependentTxIdx]
			parent.LockTask()
			if parent.IsTaskType(TypeExecution) {
				t.Parents.Add(t.Abort.DependentTxIdx)
				queue.AddDependentToParents(t.AbsoluteIndex)
			} else {
				queue.Execute(t.AbsoluteIndex)
			}
			parent.UnlockTask()

		case statusExecuted:
			TaskLog(t, fmt.Sprintf("FINISHING task EXECUTION (worker=%d, incarnation=%d)", w, t.Incarnation))
			queue.FinishExecute(t.AbsoluteIndex)

		default:
			TaskLog(t, "unexpected status")
			panic("unexpected status")
		}

	default:
		TaskLog(t, "unexpected type")
		panic("unexpected type")
	}
}
