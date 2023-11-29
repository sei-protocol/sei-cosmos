package tasks

import (
	"sync"

	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
	"github.com/tendermint/tendermint/abci/types"
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
	// statusInvalid means the task has been invalidated
	statusInvalid status = "invalid"
	// statusWaiting tasks are waiting for another tx to complete
	statusWaiting status = "waiting"
)

type deliverTxTask struct {
	Ctx     sdk.Context
	AbortCh chan occ.Abort
	rwMx    sync.RWMutex
	mx      sync.Mutex

	taskType      taskType
	status        status
	Dependencies  []int
	Abort         *occ.Abort
	Index         int
	Incarnation   int
	Request       types.RequestDeliverTx
	Response      *types.ResponseDeliverTx
	VersionStores map[sdk.StoreKey]*multiversion.VersionIndexedStore
	ValidateCh    chan status
}

func (dt *deliverTxTask) LockTask() {
	dt.mx.Lock()
}

func (dt *deliverTxTask) UnlockTask() {
	dt.mx.Unlock()
}

func (dt *deliverTxTask) SetTaskType(t taskType) {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.taskType = t
}

func (dt *deliverTxTask) IsIdle() bool {
	return dt.IsTaskType(TypeIdle)
}

func (dt *deliverTxTask) IsTaskType(t taskType) bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.taskType == t
}

func (dt *deliverTxTask) IsStatus(s status) bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == s
}

func (dt *deliverTxTask) TaskType() taskType {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.taskType
}

func (dt *deliverTxTask) SetStatus(s status) {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.status = s
}

func (dt *deliverTxTask) Status() status {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status
}

func (dt *deliverTxTask) IsInvalid() bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == statusInvalid || dt.status == statusAborted
}

func (dt *deliverTxTask) IsValid() bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == statusValidated
}

func (dt *deliverTxTask) IsWaiting() bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == statusWaiting
}

func (dt *deliverTxTask) Reset() {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.status = statusPending
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.Dependencies = nil
	dt.VersionStores = nil
}

func (dt *deliverTxTask) ResetForExecution() {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.status = statusPending
	dt.taskType = TypeExecution
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.Dependencies = nil
	dt.VersionStores = nil
}

func (dt *deliverTxTask) Increment() {
	dt.Incarnation++
	dt.ValidateCh = make(chan status, 1)
}
