package tasks

import (
	"container/heap"
	"sync"
)

type taskType int

const (
	TypeIdle taskType = iota
	TypeExecution
	TypeValidation
)

type SchedulerQueue struct {
	mx   sync.Mutex
	cond *sync.Cond
	once sync.Once

	active  sync.Map
	tasks   []*deliverTxTask
	queue   *taskHeap
	workers int
	closed  bool
}

func NewSchedulerQueue(tasks []*deliverTxTask, workers int) *SchedulerQueue {
	sq := &SchedulerQueue{
		tasks:   tasks,
		queue:   &taskHeap{},
		workers: workers,
	}
	sq.cond = sync.NewCond(&sq.mx)

	return sq
}

func (sq *SchedulerQueue) Lock() {
	sq.mx.Lock()
}

func (sq *SchedulerQueue) Unlock() {
	sq.mx.Unlock()
}

func (sq *SchedulerQueue) SetToIdle(idx int) {
	sq.Lock()
	defer sq.Unlock()
	sq.tasks[idx].SetTaskType(TypeIdle)
	sq.active.Delete(idx)
}

func (sq *SchedulerQueue) ReExecute(idx int) {
	sq.Lock()
	defer sq.Unlock()

	TaskLog(sq.tasks[idx], "-> re-execute")

	sq.tasks[idx].ResetForExecution()
	sq.pushTask(idx)
}

// ReValidate is a helper method that revalidates a task
// without making it eligible for other workers to request it to validate
func (sq *SchedulerQueue) ReValidate(idx int) {
	sq.Lock()
	defer sq.Unlock()

	if sq.tasks[idx].TaskType() != TypeValidation {
		panic("trying to re-validate a task not in validation state")
	}

	TaskLog(sq.tasks[idx], "-> re-validate")
	sq.tasks[idx].Abort = nil
	sq.tasks[idx].SetStatus(statusExecuted)
	sq.pushTask(idx)
}

func (sq *SchedulerQueue) IsCompleted() bool {
	sq.Lock()
	defer sq.Unlock()

	if len(*sq.queue) == 0 {
		for _, t := range sq.tasks {
			if !t.IsValid() || !t.IsIdle() {
				TaskLog(t, "not valid or not idle")
				return false
			}
		}
		return true
	}
	return false
}

// ValidateExecutedTask adds a task to the validation queue IFF it just executed
// this allows us to transition to validation without making it eligible for something else
// to add it to validation
func (sq *SchedulerQueue) ValidateExecutedTask(idx int) {
	sq.Lock()
	defer sq.Unlock()

	if !sq.tasks[idx].IsTaskType(TypeExecution) {
		TaskLog(sq.tasks[idx], "not in execution")
		panic("trying to validate a task not in execution")
	}

	TaskLog(sq.tasks[idx], "-> validate")
	sq.tasks[idx].SetTaskType(TypeValidation)
	sq.pushTask(idx)
}

// AddValidationTask adds a task to the validation queue IF NOT ALREADY in a queue
func (sq *SchedulerQueue) AddValidationTask(idx int) {
	sq.Lock()
	defer sq.Unlock()

	// already active
	if _, ok := sq.active.Load(idx); ok {
		return
	}

	TaskLog(sq.tasks[idx], "-> validate")
	sq.tasks[idx].SetStatus(statusExecuted)
	sq.tasks[idx].SetTaskType(TypeValidation)
	sq.pushTask(idx)
}

func (sq *SchedulerQueue) pushTask(idx int) {
	sq.active.Store(idx, struct{}{})
	heap.Push(sq.queue, idx)
	sq.cond.Broadcast()
}

func (sq *SchedulerQueue) AddExecutionTask(idx int) {
	sq.Lock()
	defer sq.Unlock()

	// already active
	if _, ok := sq.active.Load(idx); ok {
		return
	}

	TaskLog(sq.tasks[idx], "-> execute")
	sq.tasks[idx].SetTaskType(TypeExecution)
	sq.pushTask(idx)
}

func (sq *SchedulerQueue) NextTask() (*deliverTxTask, bool) {
	sq.Lock()
	defer sq.Unlock()

	for len(*sq.queue) == 0 && !sq.closed {
		sq.cond.Wait()
	}

	if sq.closed {
		return nil, false
	}

	idx := heap.Pop(sq.queue).(int)
	return sq.tasks[idx], true
}

func (sq *SchedulerQueue) Close() {
	sq.once.Do(func() {
		sq.Lock()
		defer sq.Unlock()
		sq.closed = true
		sq.cond.Broadcast()
	})
}
