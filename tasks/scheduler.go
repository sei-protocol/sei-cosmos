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
	tasks              []*TxTask
	executeCh          chan func()
	validateCh         chan func()
	timer              *Timer
}

// NewScheduler creates a new scheduler
func NewScheduler(workers int, tracingInfo *tracing.Info, deliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)) Scheduler {
	return &scheduler{
		workers:     workers,
		deliverTx:   deliverTxFunc,
		tracingInfo: tracingInfo,
		timer:       NewTimer("Scheduler"),
	}
}

func (s *scheduler) initScheduler(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) (Queue, int) {
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
	var err error
	counter := atomic.Int32{}

	WithTimer(s.timer, "ProcessAll", func() {

		queue, workers := s.initScheduler(ctx, reqs)
		wg := sync.WaitGroup{}
		wg.Add(workers)
		mx := sync.Mutex{}
		var activeCount int32
		final := atomic.Bool{}

		for i := 0; i < workers; i++ {
			go func(worker int) {
				defer wg.Done()

				for {

					mx.Lock()
					if atomic.LoadInt32(&activeCount) == 0 {
						if queue.IsCompleted() {
							if final.Load() {
								queue.Close()
							} else {
								final.Store(true)
								queue.ValidateAll()
							}
						}
					}
					mx.Unlock()

					cancel := hangDebug(func() {
						fmt.Printf("worker=%d, completed=%v\n", worker, queue.IsCompleted())
					})
					task, anyTasks := queue.NextTask(worker)
					cancel()
					atomic.AddInt32(&activeCount, 1)

					if !anyTasks {
						return
					}

					// this safely gets the task type while someone could be editing it
					if tt, ok := task.PopTaskType(); ok {
						counter.Add(1)
						if !s.processTask(ctx, tt, worker, task, queue) {
							final.Store(false)
						}
					}
					atomic.AddInt32(&activeCount, -1)
				}

			}(i)
		}

		wg.Wait()

		for _, mv := range s.multiVersionStores {
			mv.WriteLatestToStore()
		}
		results = collectResponses(s.tasks)
		err = nil
	})
	//s.timer.PrintReport()
	//fmt.Printf("Total Tasks: %d\n", counter.Load())

	return results, err
}

func (s *scheduler) processTask(ctx sdk.Context, taskType TaskType, w int, t *TxTask, queue Queue) bool {
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
			queue.FinishTask(t.Index)
			return true
		case statusWaiting:
			// task should be re-validated (waiting on others)
			// how can we wait on dependencies?
			TaskLog(t, "waiting/executed...revalidating")
			queue.AddDependentToParents(t.Index)

		case statusInvalid:
			TaskLog(t, "invalid (re-executing, re-validating > tx)")
			queue.ValidateLaterTasks(t.Index)
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
			//TODO ideally this would wait until dependencies are finished
			t.Parents = []int{t.Abort.DependentTxIdx}
			queue.AddDependentToParents(t.Index)
		} else {
			TaskLog(t, fmt.Sprintf("FINISHING task EXECUTION (worker=%d, incarnation=%d)", w, t.Incarnation))
			queue.FinishExecute(t.Index)
		}

	default:
		TaskLog(t, "unexpected type")
		panic("unexpected type")
	}
	return false
}
