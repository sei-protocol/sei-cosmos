package tasks

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
)

type deliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)

type asyncScheduler struct {
	deliverTx          deliverTxFunc
	workers            int
	multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
	ctxMx              sync.Mutex
	mx                 sync.Mutex
	vx                 sync.Mutex
}

// NewAsyncScheduler creates a new scheduler
func NewAsyncScheduler(workers int, handler deliverTxFunc) Scheduler {
	return &asyncScheduler{
		workers:   workers,
		deliverTx: handler,
	}
}

func (s *asyncScheduler) invalidateTask(task *TxTask) {
	for _, mv := range s.multiVersionStores {
		mv.InvalidateWriteset(task.Index, task.Incarnation)
	}
}

func (s *asyncScheduler) refreshCtx(ctx sdk.Context, task *TxTask) {
	cancel := waitWithMsg(fmt.Sprintf("TASK(%d): refreshCtx wait for lock", task.Index))
	defer cancel()
	s.ctxMx.Lock()
	defer s.ctxMx.Unlock()
	task.RefreshCtx(ctx, s.multiVersionStores)
}

func (s *asyncScheduler) findConflicts(task *TxTask) (bool, []int) {
	var conflicts []int
	uniq := make(map[int]struct{})
	valid := true
	for _, mv := range s.multiVersionStores {
		ok, mvConflicts := mv.ValidateTransactionState(task.Index)
		for _, c := range mvConflicts {
			if _, ok := uniq[c]; !ok {
				conflicts = append(conflicts, c)
				uniq[c] = struct{}{}
			}
		}
		// any non-ok value makes valid false
		valid = ok && valid
	}
	sort.Ints(conflicts)
	return valid, conflicts
}

func (s *asyncScheduler) initMultiVersionStore(ctx sdk.Context) {
	mvs := make(map[sdk.StoreKey]multiversion.MultiVersionStore)
	keys := ctx.MultiStore().StoreKeys()
	for _, sk := range keys {
		mvs[sk] = multiversion.NewMultiVersionStore(ctx.MultiStore().GetKVStore(sk))
	}
	s.multiVersionStores = mvs
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

func (s *asyncScheduler) ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error) {
	s.initMultiVersionStore(ctx)
	tasks, _ := toTxTasks(reqs)

	workers := s.workers
	if len(tasks) < workers || s.workers < 1 {
		workers = len(tasks)
	}

	validSignal := make(chan int, len(tasks)*100)
	defer close(validSignal)

	queue := NewSchedulerQueue(tasks)

	for idx := range tasks {
		queue.AddToExecution(idx, false)
	}

	var final bool

	wg := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for {
				task, ok := queue.NextTask()
				if !ok {
					fmt.Printf("************************ WORKER DONE %d/%d\n", worker, workers)
					return
				}

				pl := waitWithMsg("PROCESS LOCK")
				task.ProcessLock()
				pl()

				switch task.TaskType {
				case TypeExecution:
					s.refreshCtx(ctx, task)
					status := task.Execute(task.Ctx, s.deliverTx)
					queue.Complete(task.Index)
					switch status {
					case statusExecuted:
						fmt.Println(fmt.Sprintf("W(%d) TASK(%d): EXECUTED", worker, task.Index))
						queue.AddToValidation(task.Index)
					case statusAborted:
						fmt.Println(fmt.Sprintf("W(%d) TASK(%d): ABORTED (type=%d)", worker, task.Index, task.TaskType))
						queue.AddToExecution(task.Index, false)
					default:
						panic("unexpected status during execution")
					}
				case TypeValidation:
					status := s.validate(task)
					queue.Complete(task.Index)
					switch status {
					case statusInvalid:
						fmt.Println(fmt.Sprintf("W(%d) TASK(%d): INVALID", worker, task.Index))
						final = false
						queue.AddToExecution(task.Index, true)
						queue.AddLaterTasksToValidation(task.Index)
					case statusValid:
						fmt.Println(fmt.Sprintf("W(%d) TASK(%d): VALID", worker, task.Index))
						if !final {
							queue.AddLaterTasksToValidation(task.Index)
						}
						nvw := waitWithMsg("NOTIFY VALID WAIT (valid)")
						validSignal <- task.Index
						nvw()
					case statusWaiting:
						fmt.Println(fmt.Sprintf("W(%d) TASK(%d): WAITING", worker, task.Index))
						if indexesTasksValidated(tasks, task.Dependencies) {
							queue.AddToValidation(task.Index)
						}
					default:
						panic(fmt.Sprintf("W(%d) unexpected status during validation: %v", worker, task.TaskType))
					}
				case TypeIdle:
					fmt.Println(fmt.Sprintf("W(%d) TASK(%d): FOUND IDLE (do nothing)", worker, task.Index))
					if task.status == statusValid {
						c := waitWithMsg("NOTIFY VALID WAIT (idle)")
						validSignal <- task.Index
						c()
					}
				default:
					panic("unexpected task type")
				}
				task.ProcessUnlock()
			}
		}(i)
	}

	go func() {
		wg.Add(1)
		defer func() {
			fmt.Println("TRYING TO CLOSE")
			queue.Close()
			wg.Done()
		}()
		for {
			select {
			case <-ctx.Context().Done():
				return
			case <-validSignal:
				if final && allTasksValidated(tasks) && queue.Len() == 0 {
					fmt.Println("ALL DONE")
					return
				} else if !final && allTasksValidated(tasks) && queue.Len() == 0 {
					fmt.Println("ALL VALID")
					final = true
					queue.AddLaterTasksToValidation(-1)
				} else if queue.Len() == 0 {
					fmt.Printf("EMPTY QUEUE!!!!!!!!!!!!!! (valid=%v)", allTasksValidated(tasks))
				}
			}
		}
	}()

	cancel := waitWithMsg("WAITGROUP WAIT")

	wg.Wait()

	cancel()
	fmt.Println("WAITGROUP DONE")

	for _, mv := range s.multiVersionStores {
		mv.WriteLatestToStore()
	}
	return collectTaskResponses(tasks), nil
}

func (s *asyncScheduler) validate(task *TxTask) status {
	switch task.Status() {
	// validated tasks can become unvalidated if an earlier re-run task now conflicts
	case statusValidating, statusValid:
		if valid, conflicts := s.findConflicts(task); !valid {
			s.invalidateTask(task)
			if len(conflicts) > 0 {
				task.Dependencies = conflicts
				task.SetStatus(statusWaiting)
				return statusWaiting
			}
			task.SetStatus(statusInvalid)
			return statusInvalid
		} else if len(conflicts) > 0 {
			task.Dependencies = conflicts
			task.SetStatus(statusWaiting)
			return statusWaiting
		} else {
			task.SetStatus(statusValid)
			if task.Response == nil {
				panic(fmt.Sprintf("Task(%d): response is nil", task.Index))
			}
			return statusValid
		}
	default:
		panic(fmt.Sprintf("Task(%d): invalid status: %s", task.Index, task.Status()))
	}
}
