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

// TODO: remove after we have a good sense this is working
func TaskLog(task *deliverTxTask, msg string) {
	// helpful for debugging state transitions
	//fmt.Println(fmt.Sprintf("Task(%d\t%s):\t%s", task.Index, task.Status, msg))
}

func waitWithMsg(msg string) context.CancelFunc {
	goctx, cancel := context.WithCancel(context.Background())
	tick := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-goctx.Done():
				return
			case <-tick.C:
				fmt.Println(msg)
			}
		}
	}()
	return cancel
}

func (s *scheduler) ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error) {
	// initialize mutli-version stores if they haven't been initialized yet
	s.tryInitMultiVersionStore(ctx)
	// prefill estimates
	s.PrefillEstimates(ctx, reqs)
	tasks := toTasks(reqs)
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

	ch := make(chan int, len(tasks))
	active := atomic.Int32{}
	wg := sync.WaitGroup{}
	wg.Add(workers)
	var final bool

	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()

			for {
				nt := waitWithMsg(fmt.Sprintf("worker=%d: next task...(%d)", worker, active.Load()))
				t, ok := queue.NextTask()
				nt()
				if !ok {
					return
				}
				active.Add(1)
				if t.Incarnation > 20 {
					panic("too many incarnations")
				}
				if s.processTask(t, ctx, queue, tasks) {
					ch <- t.Index
				} else {
					final = false
				}
				active.Add(-1)
			}

		}(i)
	}

	wg.Add(1)
	go func() {
		defer close(ch)
		defer wg.Done()
		defer queue.Close()
		for {
			select {
			case <-ctx.Context().Done():
				return
			case <-ch:
				// if all tasks are completed AND there are no more tasks in the queue
				if active.Load() == 0 && queue.IsCompleted() {
					if final {
						return
					}
					// try one more validation of everything
					final = true
					for i := 0; i < len(tasks); i++ {
						queue.AddValidationTask(i)
					}
				}
			}
		}
	}()

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
		switch t.Status {
		case statusValidated:
			TaskLog(t, "VALIDATED")
			queue.SetToIdle(t.Index)
			return true
		case statusWaiting:
			queue.ReValidate(t.Index)
		case statusInvalid:
			queue.ReExecute(t.Index)
			for i := t.Index + 1; i < len(tasks); i++ {
				queue.AddValidationTask(i)
			}
		case statusAborted:
			//if s.allTasks[t.Abort.DependentTxIdx].Status == statusValidated {
			//	queue.ReExecute(t.Abort.DependentTxIdx)
			//}
			queue.ReExecute(t.Index)
		case statusPending:
			queue.ReExecute(t.Index)
		default:
			TaskLog(t, "unexpected status")
			panic("unexpected status ")
		}

	case TypeExecution:
		TaskLog(t, "execute")

		s.executeTask(ctx, t)
		queue.ValidateExecutedTask(t.Index)
	default:
		TaskLog(t, "unexpected type")
		panic("unexpected type ")
	}
	return false
}
