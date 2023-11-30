package tasks

import (
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
	"github.com/tendermint/tendermint/abci/types"
	"sync"
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

type TxTask struct {
	Ctx           sdk.Context
	AbortCh       chan occ.Abort
	rwMx          sync.RWMutex
	mx            sync.Mutex
	taskType      TaskType
	status        status
	ExecutionID   string
	Dependencies  []int
	Abort         *occ.Abort
	Index         int
	Incarnation   int
	Request       types.RequestDeliverTx
	Response      *types.ResponseDeliverTx
	VersionStores map[sdk.StoreKey]*multiversion.VersionIndexedStore
}

func (dt *TxTask) LockTask() {
	dt.mx.Lock()
}

func (dt *TxTask) UnlockTask() {
	dt.mx.Unlock()
}

func (dt *TxTask) IsStatus(s status) bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == s
}

func (dt *TxTask) SetTaskType(tt TaskType) bool {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	switch tt {
	case TypeValidation:
		if dt.taskType == TypeNone {
			TaskLog(dt, "SCHEDULE task VALIDATION")
			dt.taskType = tt
			return true
		}
	case TypeExecution:
		if dt.taskType != TypeExecution {
			TaskLog(dt, "SCHEDULE task EXECUTION")
			dt.taskType = tt
			return true
		}
	}
	return false
}

func (dt *TxTask) PopTaskType() (TaskType, bool) {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	tt := dt.taskType
	dt.taskType = TypeNone
	return tt, tt != TypeNone
}

func (dt *TxTask) SetStatus(s status) {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.status = s
}

func (dt *TxTask) Status() status {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status
}

func (dt *TxTask) IsInvalid() bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == statusInvalid || dt.status == statusAborted
}

func (dt *TxTask) IsValid() bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == statusValidated
}

func (dt *TxTask) IsWaiting() bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.status == statusWaiting
}

func (dt *TxTask) Reset() {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.status = statusPending
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.Dependencies = nil
	dt.VersionStores = nil
}

func (dt *TxTask) ResetForExecution() {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.status = statusPending
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.Dependencies = nil
	dt.VersionStores = nil
}

func (dt *TxTask) Increment() {
	dt.Incarnation++
}
