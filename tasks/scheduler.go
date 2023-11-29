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

type scheduler struct {
	deliverTx          func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)
	workers            int
	multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
	tracingInfo        *tracing.Info
	allTasks           []*deliverTxTask
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
	queue := NewSchedulerQueue(tasks, workers)
	queue.AddAllTasksToExecutionQueue()

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
						queue.ValidateTasksAfterIndex(-1)
					}
				}
				mx.Unlock()

				//TODO: remove once we feel good about this not hanging
				nt := waitWithMsg(fmt.Sprintf("worker=%d: next task...", worker), func() {
					fmt.Println(fmt.Sprintf("worker=%d: active=%d", worker, active.Load()))
				})
				t, ok := queue.NextTask()
				nt()
				if !ok {
					return
				}
				active.Add(1)
				if !s.processTask(t, ctx, queue, tasks) {
					// if anything doesn't validate successfully, we will need a final re-sweep
					final.Store(false)
				}
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

func (s *scheduler) processTask(t *deliverTxTask, ctx sdk.Context, queue *SchedulerQueue, tasks []*deliverTxTask) bool {
	switch t.TaskType() {
	case TypeValidation, TypeIdle:
		TaskLog(t, "validate")
		s.validateTask(ctx, t)

		// check the outcome of validation and do things accordingly
		switch t.Status() {
		case statusValidated:
			// task is possibly finished (can be re-validated by others)
			TaskLog(t, "VALIDATED (possibly finished)")
			queue.SetToIdle(t.Index)
			return true
		case statusWaiting, statusExecuted:
			// task should be re-validated (waiting on others)
			// how can we wait on dependencies?
			queue.ReValidate(t.Index)
		case statusInvalid:
			// task should be re-executed along with all +1 tasks
			queue.ReExecute(t.Index)
			queue.ValidateTasksAfterIndex(t.Index)
		default:
			TaskLog(t, "unexpected status")
			panic("unexpected status ")
		}

	case TypeExecution:
		TaskLog(t, "execute")
		t.LockTask()
		s.prepareTask(t)
		s.executeTask(t)

		if t.Status() == statusAborted {
			queue.ReExecute(t.Index)
		} else {
			queue.ValidateExecutedTask(t.Index)
			if t.Incarnation > 0 {
				queue.ValidateTasksAfterIndex(t.Index)
			}
		}
		t.UnlockTask()

	default:
		TaskLog(t, "unexpected type")
		panic("unexpected type")
	}
	return false
}
