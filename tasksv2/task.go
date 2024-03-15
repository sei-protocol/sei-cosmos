package tasksv2

import (
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
	"github.com/tendermint/tendermint/abci/types"
	"sync"
	"sync/atomic"
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
	Parents       *intSetMap
	Dependents    *intSetMap
	Abort         *occ.Abort
	AbsoluteIndex int
	Executing     byte
	Validating    byte
	Incarnation   int
	Request       types.RequestDeliverTx
	Response      *types.ResponseDeliverTx
	VersionStores map[sdk.StoreKey]*multiversion.VersionIndexedStore

	SdkTx    sdk.Tx
	Checksum [32]byte
}

func (dt *TxTask) LockTask() {
	dt.mx.Lock()
}

func (dt *TxTask) TryLockTask() bool {
	return dt.mx.TryLock()
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
	if tt == TypeValidation && dt.taskType == TypeNone && dt.Response != nil {
		return dt.updateTaskType(tt)
	} else if tt == TypeExecution && dt.taskType != TypeExecution {
		return dt.updateTaskType(tt)
	}
	return false
}

// updateTaskType assumes that an update is likely needed and does the final check within the lock.
func (dt *TxTask) updateTaskType(tt TaskType) bool {
	if tt == TypeValidation && dt.taskType == TypeNone {
		dt.taskType = tt
		return true
	} else if tt == TypeExecution && dt.taskType != TypeExecution {
		dt.taskType = tt
		return true
	}
	return false
}

func (dt *TxTask) IsTaskType(tt TaskType) bool {
	dt.rwMx.RLock()
	defer dt.rwMx.RUnlock()
	return dt.taskType == tt
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
	dt.VersionStores = nil
}

func (dt *TxTask) ResetForExecution() {
	dt.rwMx.Lock()
	defer dt.rwMx.Unlock()
	dt.status = statusPending
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.VersionStores = nil
}

func (dt *TxTask) Increment() {
	dt.Incarnation++
}

// syncSet uses byte slices instead of a map (fastest benchmark)
type syncSet struct {
	locks  []sync.RWMutex
	state  []byte
	length int32
}

func newSyncSet(size int) *syncSet {
	return &syncSet{
		state: make([]byte, size),
		locks: make([]sync.RWMutex, size),
	}
}

func (ss *syncSet) Add(idx int) {
	ss.locks[idx].Lock()
	defer ss.locks[idx].Unlock()
	// Check again to make sure it hasn't changed since acquiring the lock.
	if ss.state[idx] == byte(0) {
		ss.state[idx] = byte(1)
		atomic.AddInt32(&ss.length, 1)
	}
}

func (ss *syncSet) Delete(idx int) {
	ss.locks[idx].Lock()
	defer ss.locks[idx].Unlock()

	// Check again to make sure it hasn't changed since acquiring the lock.
	if ss.state[idx] == byte(1) {
		ss.state[idx] = byte(0)
		atomic.AddInt32(&ss.length, -1)
	}

}

func (ss *syncSet) Length() int {
	return int(atomic.LoadInt32(&ss.length))
}

func (ss *syncSet) Exists(idx int) bool {
	// Atomic read of a single byte is safe
	return ss.state[idx] == byte(1)
}
