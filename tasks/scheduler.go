package tasks

import (
	"sort"
	"sync"

	"github.com/tendermint/tendermint/abci/types"
	"golang.org/x/sync/errgroup"

	"github.com/cosmos/cosmos-sdk/store/multiversion"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
)

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs []types.RequestDeliverTx) ([]types.ResponseDeliverTx, error)
}

type scheduler struct {
	mx                 sync.Mutex
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

func (s *scheduler) invalidateTask(task *TxTask) {
	for _, mv := range s.multiVersionStores {
		mv.InvalidateWriteset(task.Index, task.Incarnation)
	}
}

func (s *scheduler) findConflicts(task *TxTask) (bool, []int) {
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
	tasks, _ := toTasks(reqs)
	toExecute := tasks
	for !allValidated(tasks) {
		var err error

		// execute sets statuses of tasks to either executed or aborted
		if len(toExecute) > 0 {
			err = s.executeAll(ctx, toExecute)
			if err != nil {
				return nil, err
			}
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
	for _, mv := range s.multiVersionStores {
		mv.WriteLatestToStore()
	}
	return collectResponses(tasks), nil
}

func (s *scheduler) validateAll(tasks []*TxTask) ([]*TxTask, error) {
	var res []*TxTask

	// find first non-validated entry
	var startIdx int
	for idx, t := range tasks {
		if t.Status() != statusValid {
			startIdx = idx
			break
		}
	}

	for i := startIdx; i < len(tasks); i++ {
		switch tasks[i].Status() {
		case statusAborted:
			// aborted means it can be re-run immediately
			res = append(res, tasks[i])

		// validated tasks can become unvalidated if an earlier re-run task now conflicts
		case statusExecuted, statusValid:
			if valid, conflicts := s.findConflicts(tasks[i]); !valid {
				s.invalidateTask(tasks[i])

				// if the conflicts are now validated, then rerun this task
				if indexesValidated(tasks, conflicts) {
					res = append(res, tasks[i])
				} else {
					// otherwise, wait for completion
					tasks[i].Dependencies = conflicts
					tasks[i].SetStatus(statusWaiting)
				}
			} else if len(conflicts) == 0 {
				tasks[i].SetStatus(statusValid)
			}

		case statusWaiting:
			// if conflicts are done, then this task is ready to run again
			if indexesValidated(tasks, tasks[i].Dependencies) {
				res = append(res, tasks[i])
			}
		}
	}
	return res, nil
}

// ExecuteAll executes all tasks concurrently
// Tasks are updated with their status
// TODO: error scenarios
func (s *scheduler) executeAll(ctx sdk.Context, tasks []*TxTask) error {
	ch := make(chan *TxTask, len(tasks))
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

					resp := s.deliverTx(task.Ctx, task.Request)

					close(task.AbortCh)

					if abt, ok := <-task.AbortCh; ok {
						task.SetStatus(statusAborted)
						task.Abort = &abt
						continue
					}

					// write from version store to multiversion stores
					for _, v := range task.VersionStores {
						v.WriteToMultiVersionStore()
					}

					task.SetStatus(statusExecuted)
					task.Response = &resp
				}
			}
		})
	}
	grp.Go(func() error {
		defer close(ch)
		for _, task := range tasks {
			// initialize the context
			ctx = ctx.WithTxIndex(task.Index)
			abortCh := make(chan occ.Abort, len(s.multiVersionStores))

			// if there are no stores, don't try to wrap, because there's nothing to wrap
			if len(s.multiVersionStores) > 0 {
				// non-blocking
				cms := ctx.MultiStore().CacheMultiStore()

				// init version stores by store key
				vs := make(map[store.StoreKey]*multiversion.VersionIndexedStore)
				for storeKey, mvs := range s.multiVersionStores {
					vs[storeKey] = mvs.VersionedIndexedStore(task.Index, task.Incarnation, abortCh)
				}

				// save off version store so we can ask it things later
				task.VersionStores = vs
				ms := cms.SetKVStores(func(k store.StoreKey, kvs sdk.KVStore) store.CacheWrap {
					return vs[k]
				})

				ctx = ctx.WithMultiStore(ms)
			}

			task.AbortCh = abortCh
			task.Ctx = ctx

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
