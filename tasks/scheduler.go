package tasks

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/utils/tracing"
	"github.com/tendermint/tendermint/abci/types"
)

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error)
}

var aborts = atomic.Int32{}

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

	active := atomic.Int32{}
	wg := sync.WaitGroup{}
	wg.Add(workers)
	final := atomic.Bool{}
	finisher := sync.Once{}
	mx := sync.Mutex{}

	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()

			for {

				// check if all tasks are complete AND not running anything
				mx.Lock()
				if active.Load() == 0 && queue.IsCompleted() {
					if final.Load() {
						finisher.Do(func() {
							queue.Close()
						})
					} else {
						// try one more validation of everything at end
						final.Store(true)
						queue.ValidateLaterTasks(-1)
					}
				}
				mx.Unlock()

				//TODO: remove once we feel good about this not hanging
				nt := waitWithMsg(fmt.Sprintf("worker=%d: next task...", worker), func() {
					fmt.Println(fmt.Sprintf("worker=%d: active=%d, complete=%v", worker, active.Load(), queue.IsCompleted()))
				})
				task, anyTasks := queue.NextTask()
				nt()
				if !anyTasks {
					return
				}
				active.Add(1)

				task.LockTask()
				if taskType, ok := task.PopTaskType(); ok {
					if !s.processTask(ctx, taskType, worker, task, queue) {
						final.Store(false)
					}
				} else {
					TaskLog(task, "NONE FOUND...SKIPPING")
				}
				task.UnlockTask()
				active.Add(-1)
			}

		}(i)
	}

	wg.Wait()

	for _, mv := range s.multiVersionStores {
		mv.WriteLatestToStore()
	}
	return collectResponses(tasks), nil
}

func (s *scheduler) processTask(ctx sdk.Context, taskType TaskType, w int, t *TxTask, queue Queue) bool {
	switch taskType {
	case TypeValidation:
		TaskLog(t, "validate")
		s.validateTask(ctx, t)

		// check the outcome of validation and do things accordingly
		switch t.Status() {
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
			} else {
				queue.ReValidate(t.Index)
			}
		case statusInvalid:
			// task should be re-executed along with all +1 tasks
			TaskLog(t, "invalid (re-executing, re-validating > tx)")
			queue.Execute(t.Index)
		default:
			TaskLog(t, "unexpected status")
			panic("unexpected status ")
		}

	case TypeExecution:
		t.ResetForExecution()
		TaskLog(t, fmt.Sprintf("execute (worker=%d)", w))

		s.executeTask(t)

		if t.IsStatus(statusAborted) {
			aborts.Add(1)
			if aborts.Load() > 50 {
				TaskLog(t, fmt.Sprintf("too many aborts, depending on: index=%d", t.Abort.DependentTxIdx))
				panic("too many aborts")
			}
			queue.ReExecute(t.Index)

		} else {
			aborts.Store(0)
			queue.ValidateLaterTasks(t.Index)
			TaskLog(t, fmt.Sprintf("FINISHING task EXECUTION (worker=%d, incarnation=%d)", w, t.Incarnation))
			queue.FinishExecute(t.Index)
		}

	default:
		TaskLog(t, "unexpected type")
		panic("unexpected type")
	}
	return false
}
