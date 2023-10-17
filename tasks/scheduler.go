package tasks

import (
	"github.com/tendermint/tendermint/abci/types"
	"golang.org/x/sync/errgroup"

	"github.com/cosmos/cosmos-sdk/store/multiversion"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
)

type status string

const (
	// statusPending tasks are ready for execution
	// all executing tasks are in pending state
	statusPending status = "pending"
	// statusExecuted tasks are ready for validation
	// these tasks did not abort during execution
	statusExecuted status = "executed"
	// statusAborted means the task has been aborted
	// these tasks transition to pending upon next execution
	statusAborted status = "aborted"
	// statusValidated means the task has been validated
	// tasks in this status can be reset if an earlier task fails validation
	statusValidated status = "validated"
)

type deliverTxTask struct {
	Status        status
	Abort         *occ.Abort
	Index         int
	Incarnation   int
	Request       types.RequestDeliverTx
	Response      *types.ResponseDeliverTx
	VersionStores map[sdk.StoreKey]*multiversion.VersionIndexedStore
}

func (dt *deliverTxTask) Increment() {
	dt.Incarnation++
	dt.Status = statusPending
	dt.Response = nil
	dt.Abort = nil
	dt.VersionStores = nil
}

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs []types.RequestDeliverTx) ([]types.ResponseDeliverTx, error)
}

type scheduler struct {
	deliverTx          func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)
	workers            int
	multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
}

// NewScheduler creates a new scheduler
func NewScheduler(workers int, deliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)) Scheduler {
	return &scheduler{
		workers:   workers,
		deliverTx: deliverTxFunc,
	}
}

func toTasks(reqs []types.RequestDeliverTx) []*deliverTxTask {
	res := make([]*deliverTxTask, 0, len(reqs))
	for idx, r := range reqs {
		res = append(res, &deliverTxTask{
			Request: r,
			Index:   idx,
			Status:  statusPending,
		})
	}
	return res
}

func collectResponses(tasks []*deliverTxTask) []types.ResponseDeliverTx {
	res := make([]types.ResponseDeliverTx, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, *t.Response)
	}
	return res
}

func (s *scheduler) initMultiVersionStore(ctx sdk.Context) {
	mvs := make(map[sdk.StoreKey]multiversion.MultiVersionStore)
	keys := ctx.MultiStore().StoreKeys()
	for _, sk := range keys {
		mvs[sk] = multiversion.NewMultiVersionStore(ctx.MultiStore().GetKVStore(sk))
	}
	s.multiVersionStores = mvs
}

func (s *scheduler) ProcessAll(ctx sdk.Context, reqs []types.RequestDeliverTx) ([]types.ResponseDeliverTx, error) {
	s.initMultiVersionStore(ctx)
	tasks := toTasks(reqs)
	toExecute := tasks
	for len(toExecute) > 0 {

		// execute sets statuses of tasks to either executed or aborted
		err := s.executeAll(ctx, toExecute)
		if err != nil {
			return nil, err
		}

		// validate returns any that should be re-executed
		// note this processes ALL tasks, not just those recently executed
		toExecute, err = s.validateAll(tasks)
		if err != nil {
			return nil, err
		}
		for _, t := range toExecute {
			t.Increment()
		}
	}
	return collectResponses(tasks), nil
}

// TODO: validate each tasks
// TODO: return list of tasks that are invalid
func (s *scheduler) validateAll(tasks []*deliverTxTask) ([]*deliverTxTask, error) {
	var res []*deliverTxTask

	// find first non-validated entry
	var startIdx int
	for idx, t := range tasks {
		if t.Status != statusValidated {
			startIdx = idx
			break
		}
	}

	for i := startIdx; i < len(tasks); i++ {
		// any aborted tx is known to be suspect here
		if tasks[i].Status == statusAborted {
			res = append(res, tasks[i])
		} else {
			for _, mv := range s.multiVersionStores {
				conflicts := mv.ValidateTransactionState(tasks[i].Index)
				if len(conflicts) == 0 {
					tasks[i].Status = statusValidated
				} else {
					//TODO: should one do anything with the conflicts?
					mv.InvalidateWriteset(tasks[i].Index, tasks[i].Incarnation)
					tasks[i].Increment()
				}
			}
		}
	}
	return res, nil
}

// ExecuteAll executes all tasks concurrently
// Tasks are updated with their status
// TODO: retries on aborted tasks
// TODO: error scenarios
func (s *scheduler) executeAll(ctx sdk.Context, tasks []*deliverTxTask) error {
	ch := make(chan *deliverTxTask, len(tasks))
	grp, gCtx := errgroup.WithContext(ctx.Context())

	// a workers value < 1 means no limit
	workers := s.workers
	if s.workers < 1 {
		workers = len(tasks)
	}

	for i := 0; i < workers; i++ {
		grp.Go(func() error {
			for {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				case task, ok := <-ch:
					if !ok {
						return nil
					}

					// non-blocking
					abortCh := make(chan occ.Abort, len(s.multiVersionStores))
					cms := ctx.MultiStore().CacheMultiStore()

					// init version stores by store key
					vs := make(map[store.StoreKey]*multiversion.VersionIndexedStore)
					for k, v := range s.multiVersionStores {
						vs[k] = v.VersionedIndexedStore(task.Incarnation, task.Index, abortCh)
					}
					task.VersionStores = vs
					ms := cms.SetKVStores(func(k store.StoreKey, kvs sdk.KVStore) store.CacheWrapper {
						return vs[k]
					})

					ctx = ctx.WithMultiStore(ms)

					resp := s.deliverTx(ctx, task.Request)
					close(abortCh)

					if abt, ok := <-abortCh; ok {
						task.Status = statusAborted
						task.Abort = &abt
						continue
					}

					// write to multiversion stores so they're available to other tasks
					for _, v := range vs {
						v.WriteToMultiVersionStore()
					}

					task.Status = statusExecuted
					task.Response = &resp
				}
			}
		})
	}
	grp.Go(func() error {
		defer close(ch)
		for _, task := range tasks {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case ch <- task:
			}
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		return err
	}

	return nil
}
