package tasks

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
)

// TODO: remove after things work
func TaskLog(task *deliverTxTask, msg string) {
	// helpful for debugging state transitions
	//fmt.Println(fmt.Sprintf("Task(%d\t%s):\t%s", task.Index, task.Status, msg))
}

// TODO: remove after things work
// waitWithMsg prints a message every 1s, so we can tell what's hanging
func waitWithMsg(msg string, handlers ...func()) context.CancelFunc {
	goctx, cancel := context.WithCancel(context.Background())
	tick := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-goctx.Done():
				return
			case <-tick.C:
				fmt.Println(msg)
				for _, h := range handlers {
					h()
				}
			}
		}
	}()
	return cancel
}

func (s *scheduler) ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error) {
	// initialize mutli-version stores if they haven't been initialized yet
	s.tryInitMultiVersionStore(ctx)
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
	for _, t := range tasks {
		queue.AddExecutionTask(t.Index)
	}

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
						for i := 0; i < len(tasks); i++ {
							queue.AddValidationTask(i)
						}
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
	switch t.Type {
	case TypeValidation:
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
			queue.ReValidate(t.Index)
		case statusInvalid:
			// task should be re-executed along with all +1 tasks
			queue.ReExecute(t.Index)
			for i := t.Index + 1; i < len(tasks); i++ {
				queue.AddValidationTask(i)
			}
		case statusAborted:
			// task should be re-executed
			queue.ReExecute(t.Index)
		default:
			TaskLog(t, "unexpected status")
			panic("unexpected status ")
		}

	case TypeExecution:
		TaskLog(t, "execute")
		s.prepareTask(t)
		s.executeTask(t)
		queue.ValidateExecutedTask(t.Index)
	default:
		TaskLog(t, "unexpected type")
		panic("unexpected type ")
	}
	return false
}
