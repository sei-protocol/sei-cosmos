package tasks

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
	"github.com/tendermint/tendermint/abci/types"
	"sync"
	"time"
)

type status string

const (
	// statusPending tasks are ready for execution
	// all executing tasks are in pending state
	statusPending status = "pending"
	// statusExecuted tasks are ready for validation
	// these tasks did not abort during execution
	statusExecuted status = "executed"
	// statusExecuting means the task is currently executing
	statusExecuting status = "executing"
	// statusAborted means the task has been aborted
	// these tasks transition to pending upon next execution
	statusAborted status = "aborted"
	// statusValid means the task has been validated
	// tasks in this status can be reset if an earlier task fails validation
	statusValid status = "valid"
	// statusInvalid means the task is invalid
	statusInvalid status = "invalid"
	// statusValidating means the task is being validated
	statusValidating status = "validating"
	// statusWaiting tasks are waiting for another tx to complete
	statusWaiting status = "waiting"
)

type TxTask struct {
	Ctx     sdk.Context
	AbortCh chan occ.Abort

	TaskType      TaskType
	mx            sync.Mutex
	pmx           sync.Mutex
	status        status
	Dependencies  []int
	Abort         *occ.Abort
	Index         int
	Incarnation   int
	Processing    bool
	Request       types.RequestDeliverTx
	Response      *types.ResponseDeliverTx
	VersionStores map[sdk.StoreKey]*multiversion.VersionIndexedStore
}

func (dt *TxTask) RefreshCtx(ctx sdk.Context, mvsMap map[sdk.StoreKey]multiversion.MultiVersionStore) {
	ctx = ctx.WithTxIndex(dt.Index)
	abortCh := make(chan occ.Abort, len(mvsMap))

	// if there are no stores, don't try to wrap, because there's nothing to wrap
	if len(mvsMap) > 0 {
		// non-blocking
		cms := ctx.MultiStore().CacheMultiStore()

		// init version stores by store key
		vs := make(map[store.StoreKey]*multiversion.VersionIndexedStore)
		for storeKey, mvs := range mvsMap {
			vs[storeKey] = mvs.VersionedIndexedStore(dt.Index, dt.Incarnation, abortCh)
		}

		// save off version store so we can ask it things later
		dt.VersionStores = vs
		ms := cms.SetKVStores(func(k store.StoreKey, kvs sdk.KVStore) store.CacheWrap {
			return vs[k]
		})

		ctx = ctx.WithMultiStore(ms)
	}
	dt.AbortCh = abortCh
	dt.Ctx = ctx
}

func (dt *TxTask) ProcessLock() {
	dt.pmx.Lock()
}

func (dt *TxTask) ProcessUnlock() {
	dt.pmx.Unlock()
}

func (dt *TxTask) Lock() {
	dt.mx.Lock()
}

func (dt *TxTask) Unlock() {
	dt.mx.Unlock()
}

func (dt *TxTask) Execute(ctx sdk.Context, handler deliverTxFunc) status {
	cancel := waitWithMsg(fmt.Sprintf("TASK(%d): Executing...", dt.Index))
	defer cancel()

	dt.status = statusExecuting
	resp := handler(ctx, dt.Request)

	close(dt.AbortCh)

	if abt, ok := <-dt.AbortCh; ok {
		dt.Abort = &abt
		dt.status = statusAborted
		return dt.status
	}

	// write from version store to multiversion stores
	for _, v := range dt.VersionStores {
		v.WriteToMultiVersionStore()
	}

	dt.Response = &resp
	dt.status = statusExecuted

	return dt.status
}

func (dt *TxTask) Increment() {
	dt.Incarnation++
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.Dependencies = nil
	dt.VersionStores = nil
}

func (dt *TxTask) SetStatus(status status) {
	dt.status = status
}

func (dt *TxTask) IsExecuting() bool {
	return dt.status == statusExecuting || dt.status == statusPending || dt.status == statusAborted
}

func (dt *TxTask) IsValidating() bool {
	return dt.status == statusValidating
}

func (dt *TxTask) Status() status {
	return dt.status
}

func collectResponses(tasks []*TxTask) []types.ResponseDeliverTx {
	res := make([]types.ResponseDeliverTx, 0, len(tasks))
	for _, t := range tasks {
		if t.Response == nil {
			fmt.Printf("%d IS NIL\n", t.Index)
			time.Sleep(5 * time.Second)
		}
		res = append(res, *t.Response)
	}
	return res
}

func toTasks(reqs []types.RequestDeliverTx) ([]*TxTask, *sync.WaitGroup) {
	res := make([]*TxTask, 0, len(reqs))
	wg := sync.WaitGroup{}
	for idx, r := range reqs {
		wg.Add(1)
		res = append(res, &TxTask{
			Request: r,
			Index:   idx,
			status:  statusPending,
		})
	}
	return res, &wg
}

func indexesValidated(tasks []*TxTask, idx []int) bool {
	for _, i := range idx {
		if tasks[i].Status() != statusValid {
			return false
		}
	}
	return true
}

func allValidated(tasks []*TxTask) bool {
	for _, t := range tasks {
		if t.Status() != statusValid {
			//fmt.Println(fmt.Sprintf("TASK(%d): %s", t.Index, t.Status()))
			return false
		}
	}
	return true
}
