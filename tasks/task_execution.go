package tasks

import (
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
)

// prepareTask initializes the context and version stores for a task
func (s *scheduler) prepareTask(task *TxTask) {
	ctx := task.Ctx.WithTxIndex(task.Index)

	_, span := s.traceSpan(ctx, "SchedulerPrepare", task)
	defer span.End()

	// initialize the context
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
}

// executeTask executes a single task
func (s *scheduler) executeTask(task *TxTask) {
	dCtx, dSpan := s.traceSpan(task.Ctx, "SchedulerExecuteTask", task)
	defer dSpan.End()
	task.Ctx = dCtx

	s.prepareTask(task)

	resp := s.deliverTx(task.Ctx, task.Request)

	close(task.AbortCh)

	if abt, ok := <-task.AbortCh; ok {
		task.SetStatus(statusAborted)
		task.Abort = &abt
		return
	}

	// write from version store to multiversion stores
	for _, v := range task.VersionStores {
		v.WriteToMultiVersionStore()
	}

	task.SetStatus(statusExecuted)
	task.Response = &resp
}
