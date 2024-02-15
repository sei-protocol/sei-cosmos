package tasks

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/store/multiversion"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/occ"
	"github.com/cosmos/cosmos-sdk/utils/tracing"
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
	// statusWaiting tasks are waiting for another tx to complete
	statusWaiting status = "waiting"
)

var (
	TotalExecute  = atomic.Int64{}
	TotalReset    = atomic.Int64{}
	TotalRerun    = atomic.Int64{}
	TotalValidate = atomic.Int64{}
)

type deliverTxTask struct {
	Ctx     sdk.Context
	AbortCh chan occ.Abort

	Status        status
	Dependencies  []int
	Abort         *occ.Abort
	Index         int
	Incarnation   int
	Request       types.RequestDeliverTx
	Response      *types.ResponseDeliverTx
	VersionStores map[sdk.StoreKey]*multiversion.VersionIndexedStore
	ValidateCh    chan struct{}
}

func (dt *deliverTxTask) Reset() {
	TotalReset.Add(1)
	dt.Status = statusPending
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.Dependencies = nil
	dt.VersionStores = nil
}

func (dt *deliverTxTask) Increment() {
	dt.Incarnation++
	dt.ValidateCh = make(chan struct{}, 1)
}

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error)
}

type scheduler struct {
	deliverTx          func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)
	workers            int
	multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
	tracingInfo        *tracing.Info
	allTasks           []*deliverTxTask
	executeCh          chan func()
	validateCh         chan func()
}

// NewScheduler creates a new scheduler
func NewScheduler(workers int, tracingInfo *tracing.Info, deliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)) Scheduler {
	return &scheduler{
		workers:     workers,
		deliverTx:   deliverTxFunc,
		tracingInfo: tracingInfo,
	}
}

func (s *scheduler) invalidateTask(task *deliverTxTask) {
	for _, mv := range s.multiVersionStores {
		mv.InvalidateWriteset(task.Index, task.Incarnation)
		mv.ClearReadset(task.Index)
		mv.ClearIterateset(task.Index)
	}
}

func start(ctx context.Context, ch chan func(), workers int) {
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case work := <-ch:
					work()
				}
			}
		}()
	}
}

func (s *scheduler) DoValidate(work func()) {
	s.validateCh <- work
}

func (s *scheduler) DoExecute(work func()) {
	s.executeCh <- work
}

func (s *scheduler) findConflicts(task *deliverTxTask) (bool, []int) {
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

func toTasks(reqs []*sdk.DeliverTxEntry) []*deliverTxTask {
	res := make([]*deliverTxTask, 0, len(reqs))
	for idx, r := range reqs {
		res = append(res, &deliverTxTask{
			Request:    r.Request,
			Index:      idx,
			Status:     statusPending,
			ValidateCh: make(chan struct{}, 1),
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

func (s *scheduler) tryInitMultiVersionStore(ctx sdk.Context) {
	if s.multiVersionStores != nil {
		return
	}
	mvs := make(map[sdk.StoreKey]multiversion.MultiVersionStore)
	keys := ctx.MultiStore().StoreKeys()
	for _, sk := range keys {
		mvs[sk] = multiversion.NewMultiVersionStore(ctx.MultiStore().GetKVStore(sk))
	}
	s.multiVersionStores = mvs
}

func indexesValidated(tasks []*deliverTxTask, idx []int) bool {
	for _, i := range idx {
		if tasks[i].Status != statusValidated {
			return false
		}
	}
	return true
}

func allValidated(tasks []*deliverTxTask) bool {
	for _, t := range tasks {
		if t.Status != statusValidated {
			return false
		}
	}
	return true
}

func (s *scheduler) PrefillEstimates(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) {
	// iterate over TXs, update estimated writesets where applicable
	for i, req := range reqs {
		mappedWritesets := req.EstimatedWritesets
		// order shouldnt matter for storeKeys because each storeKey partitioned MVS is independent
		for storeKey, writeset := range mappedWritesets {
			// we use `-1` to indicate a prefill incarnation
			s.multiVersionStores[storeKey].SetEstimatedWriteset(i, -1, writeset)
		}
	}
}

func (s *scheduler) ProcessAll(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) ([]types.ResponseDeliverTx, error) {
	startTime := time.Now()

	// initialize mutli-version stores if they haven't been initialized yet
	s.tryInitMultiVersionStore(ctx)
	s.PrefillEstimates(ctx, reqs)

	tasks := toTasks(reqs)
	s.allTasks = tasks
	s.executeCh = make(chan func(), len(tasks))
	s.validateCh = make(chan func(), len(tasks))
	TotalExecute.Store(0)
	TotalValidate.Store(0)
	TotalRerun.Store(0)
	TotalReset.Store(0)
	defer func() {
		fmt.Printf("[Debug] ProcessAll %d txs took %s, total execute %d, total validate %d, total reset %d, total rerun %d \n",
			len(reqs), time.Since(startTime), TotalExecute.Load(), TotalValidate.Load(), TotalReset.Load(), TotalRerun.Load())
	}()
	workers := s.workers
	if s.workers < 1 {
		workers = len(tasks)
	}

	workerCtx, cancel := context.WithCancel(ctx.Context())
	defer cancel()
	start(workerCtx, s.executeCh, workers)
	start(workerCtx, s.validateCh, workers)

	toExecute := tasks
	var totalExecuteAllLatency = int64(0)
	var totalValidateAllLatency = int64(0)
	for !allValidated(tasks) {
		var err error

		// execute sets statuses of tasks to either executed or aborted
		if len(toExecute) > 0 {
			executeAllStartTime := time.Now()
			err = s.executeAll(ctx, toExecute)
			totalExecuteAllLatency += time.Since(executeAllStartTime).Microseconds()
			if err != nil {
				return nil, err
			}
		}

		// validate returns any that should be re-executed
		// note this processes ALL tasks, not just those recently executed
		validateAllStartTime := time.Now()
		toExecute, err = s.validateAll(ctx, tasks)
		totalValidateAllLatency += time.Since(validateAllStartTime).Microseconds()
		if err != nil {
			return nil, err
		}
	}
	for _, mv := range s.multiVersionStores {
		mv.WriteLatestToStore()
	}
	return collectResponses(tasks), nil
}

func (s *scheduler) shouldRerun(task *deliverTxTask) bool {
	switch task.Status {

	case statusAborted, statusPending:
		return true

	// validated tasks can become unvalidated if an earlier re-run task now conflicts
	case statusExecuted, statusValidated:
		if valid, conflicts := s.findConflicts(task); !valid {
			s.invalidateTask(task)

			// if the conflicts are now validated, then rerun this task
			if indexesValidated(s.allTasks, conflicts) {
				return true
			} else {
				// otherwise, wait for completion
				task.Dependencies = conflicts
				task.Status = statusWaiting
				return false
			}
		} else if len(conflicts) == 0 {
			// mark as validated, which will avoid re-validating unless a lower-index re-validates
			task.Status = statusValidated
			return false
		}
		// conflicts and valid, so it'll validate next time
		return false

	case statusWaiting:
		// if conflicts are done, then this task is ready to run again
		return indexesValidated(s.allTasks, task.Dependencies)
	}
	panic("unexpected status: " + task.Status)
}

func (s *scheduler) validateTask(ctx sdk.Context, task *deliverTxTask) bool {
	TotalValidate.Add(1)
	if s.shouldRerun(task) {
		TotalRerun.Add(1)
		return false
	}
	return true
}

func (s *scheduler) findFirstNonValidated() (int, bool) {
	for i, t := range s.allTasks {
		if t.Status != statusValidated {
			return i, true
		}
	}
	return 0, false
}

func (s *scheduler) validateAll(ctx sdk.Context, tasks []*deliverTxTask) ([]*deliverTxTask, error) {
	//ctx, span := s.traceSpan(ctx, "SchedulerValidateAll", nil)
	//defer span.End()

	var mx sync.Mutex
	var res []*deliverTxTask

	wg := sync.WaitGroup{}
	for i := 0; i < len(tasks); i++ {
		t := tasks[i]
		wg.Add(1)
		s.DoValidate(func() {
			defer wg.Done()
			if !s.validateTask(ctx, t) {
				t.Reset()
				t.Increment()
				mx.Lock()
				res = append(res, t)
				mx.Unlock()
			}
		})
	}
	wg.Wait()

	return res, nil
}

// ExecuteAll executes all tasks concurrently
// Tasks are updated with their status
// TODO: error scenarios
func (s *scheduler) executeAll(ctx sdk.Context, tasks []*deliverTxTask) error {
	//ctx, span := s.traceSpan(ctx, "SchedulerExecuteAll", nil)
	//defer span.End()

	// validationWg waits for all validations to complete
	// validations happen in separate goroutines in order to wait on previous index
	validationWg := &sync.WaitGroup{}
	validationWg.Add(len(tasks))

	for _, task := range tasks {
		t := task
		s.DoExecute(func() {
			s.prepareAndRunTask(validationWg, ctx, t)
		})
	}

	validationWg.Wait()

	return nil
}

func (s *scheduler) prepareAndRunTask(wg *sync.WaitGroup, ctx sdk.Context, task *deliverTxTask) {
	//eCtx, eSpan := s.traceSpan(ctx, "SchedulerExecute", task)
	//defer eSpan.End()
	task.Ctx = ctx

	s.executeTask(task.Ctx, task)
	TotalExecute.Add(1)
	go func() {
		defer wg.Done()
		defer close(task.ValidateCh)
		// wait on previous task to finish validation
		if task.Index > 0 {
			<-s.allTasks[task.Index-1].ValidateCh
		}
		if !s.validateTask(task.Ctx, task) {
			task.Reset()
		}
		task.ValidateCh <- struct{}{}
	}()
}

// prepareTask initializes the context and version stores for a task
func (s *scheduler) prepareTask(ctx sdk.Context, task *deliverTxTask) {
	ctx = ctx.WithTxIndex(task.Index)

	//_, span := s.traceSpan(ctx, "SchedulerPrepare", task)
	//defer span.End()

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
func (s *scheduler) executeTask(ctx sdk.Context, task *deliverTxTask) {

	s.prepareTask(ctx, task)

	//dCtx, dSpan := s.traceSpan(task.Ctx, "SchedulerDeliverTx", task)
	//defer dSpan.End()
	//task.Ctx = dCtx

	resp := s.deliverTx(task.Ctx, task.Request)

	close(task.AbortCh)

	if abt, ok := <-task.AbortCh; ok {
		task.Status = statusAborted
		task.Abort = &abt
		return
	}

	// write from version store to multiversion stores
	for _, v := range task.VersionStores {
		v.WriteToMultiVersionStore()
	}

	task.Status = statusExecuted
	task.Response = &resp
}
