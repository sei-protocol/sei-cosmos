package tasks

import (
	"container/heap"
	"fmt"
	"sort"
	"sync"
)

type TaskType string

const (
	TypeNone       TaskType = "NONE"
	TypeExecution  TaskType = "EXECUTE"
	TypeValidation TaskType = "VALIDATE"
)

type Queue interface {
	// NextTask returns the next task to be executed, or nil if the queue is closed.
	NextTask() (*TxTask, bool)
	// Close closes the queue, causing NextTask to return false.
	Close()
	// ExecuteAll executes all tasks in the queue.
	ExecuteAll()
	// Execute executes a task
	Execute(idx int)
	// ReExecute re-executes a task that just executed
	ReExecute(idx int)
	// ReValidate re-validates a task.
	ReValidate(idx int)
	// FinishExecute marks a task as finished executing.
	FinishExecute(idx int)
	// FinishTask marks a task as finished (only upon valid).
	FinishTask(idx int)
	// ValidateLaterTasks marks all tasks after the given index as pending validation.
	ValidateLaterTasks(afterIdx int)
	// IsCompleted returns true if all tasks have been executed and validated.
	IsCompleted() bool
	// DependenciesFinished returns whether all dependencies are finished
	DependenciesFinished(idx int) bool
}

type taskQueue struct {
	mx        sync.Mutex
	condMx    sync.Mutex
	heapMx    sync.Mutex
	cond      *sync.Cond
	once      sync.Once
	executing sync.Map
	queued    sync.Map
	finished  sync.Map
	tasks     []*TxTask
	queue     *taskHeap
	closed    bool
}

func NewTaskQueue(tasks []*TxTask) Queue {
	sq := &taskQueue{
		tasks: tasks,
		queue: &taskHeap{},
	}
	sq.cond = sync.NewCond(&sq.condMx)

	return sq
}

func (sq *taskQueue) lock() {
	sq.mx.Lock()
}

func (sq *taskQueue) unlock() {
	sq.mx.Unlock()
}

func (sq *taskQueue) execute(idx int) {
	if sq.tasks[idx].SetTaskType(TypeExecution) {
		TaskLog(sq.tasks[idx], "-> execute")
		sq.finished.Delete(idx)
		sq.executing.Store(idx, struct{}{})
		sq.pushTask(idx, TypeExecution)
	}
}

func (sq *taskQueue) validate(idx int) {
	if sq.isExecuting(idx) {
		TaskLog(sq.tasks[idx], "(skip validating, executing...)")
		return
	}
	if sq.tasks[idx].SetTaskType(TypeValidation) {
		TaskLog(sq.tasks[idx], "-> validate")
		sq.pushTask(idx, TypeValidation)
	}
}

func (sq *taskQueue) isQueued(idx int) bool {
	_, ok := sq.queued.Load(idx)
	return ok
}

func (sq *taskQueue) isExecuting(idx int) bool {
	_, ok := sq.executing.Load(idx)
	return ok
}

// FinishExecute marks a task as finished executing and transitions directly validation
func (sq *taskQueue) FinishExecute(idx int) {
	sq.lock()
	defer sq.unlock()

	TaskLog(sq.tasks[idx], "-> finish task execute")

	if !sq.isExecuting(idx) {
		TaskLog(sq.tasks[idx], "not executing, but trying to finish execute")
		panic("not executing, but trying to finish execute")
	}

	sq.executing.Delete(idx)
	sq.validate(idx)
}

// FinishTask marks a task as finished if nothing else queued it
// this drives whether the queue thinks everything is done processing
func (sq *taskQueue) FinishTask(idx int) {
	sq.lock()
	defer sq.unlock()

	TaskLog(sq.tasks[idx], "FinishTask -> task is FINISHED (for now)")

	sq.finished.Store(idx, struct{}{})
}

// ReValidate re-validates a task (back to queue from validation)
func (sq *taskQueue) ReValidate(idx int) {
	sq.lock()
	defer sq.unlock()

	if sq.isExecuting(idx) {
		TaskLog(sq.tasks[idx], "task is executing (unexpected)")
		panic("cannot re-validate an executing task")
	}

	sq.validate(idx)
}

func (sq *taskQueue) Execute(idx int) {
	sq.lock()
	defer sq.unlock()

	TaskLog(sq.tasks[idx], fmt.Sprintf("-> Execute (%d)", sq.tasks[idx].Incarnation))

	if sq.isExecuting(idx) {
		TaskLog(sq.tasks[idx], "task is executing (unexpected)")
		panic("cannot execute an executing task")
	}

	sq.tasks[idx].Increment()
	sq.execute(idx)
}

// ReExecute re-executes a task (back to queue from execution)
func (sq *taskQueue) ReExecute(idx int) {
	sq.lock()
	defer sq.unlock()

	TaskLog(sq.tasks[idx], fmt.Sprintf("-> RE-execute (%d)", sq.tasks[idx].Incarnation))

	if !sq.isExecuting(idx) {
		TaskLog(sq.tasks[idx], "task is not executing (unexpected)")
		panic("cannot re-execute a non-executing task")
	}

	sq.tasks[idx].Increment()
	sq.execute(idx)
}

// ValidateLaterTasks marks all tasks after the given index as pending validation.
// any executing tasks are skipped
func (sq *taskQueue) ValidateLaterTasks(afterIdx int) {
	sq.lock()
	defer sq.unlock()

	for idx := afterIdx + 1; idx < len(sq.tasks); idx++ {
		sq.validate(idx)
	}
}

func (sq *taskQueue) isFinished(idx int) bool {
	_, ok := sq.finished.Load(idx)
	return ok && sq.tasks[idx].IsStatus(statusValidated)
}

func (sq *taskQueue) DependenciesFinished(idx int) bool {
	for _, dep := range sq.tasks[idx].Dependencies {
		if !sq.isFinished(dep) {
			return false
		}
	}
	return true
}

// IsCompleted returns true if all tasks are "finished"
func (sq *taskQueue) IsCompleted() bool {
	sq.lock()
	defer sq.unlock()

	if len(*sq.queue) == 0 {
		for _, t := range sq.tasks {
			if !sq.isFinished(t.Index) {
				TaskLog(t, "not finished yet")
				return false
			}
		}
		return true
	}
	return false
}

func (sq *taskQueue) pushTask(idx int, taskType TaskType) {
	sq.condMx.Lock()
	defer sq.condMx.Unlock()
	sq.queued.Store(idx, struct{}{})
	TaskLog(sq.tasks[idx], fmt.Sprintf("-> PUSH task (%s/%d)", taskType, sq.tasks[idx].Incarnation))
	heap.Push(sq.queue, idx)
	sq.cond.Broadcast()
}

// ExecuteAll executes all tasks in the queue (called to start processing)
func (sq *taskQueue) ExecuteAll() {
	sq.lock()
	defer sq.unlock()

	for idx := range sq.tasks {
		sq.execute(idx)
	}
}

// NextTask returns the next task to be executed, or nil if the queue is closed.
// this hangs if no tasks are ready because it's possible a new task might arrive
// closing the queue causes NextTask to return false immediately
func (sq *taskQueue) NextTask() (*TxTask, bool) {
	sq.condMx.Lock()
	defer sq.condMx.Unlock()

	for len(*sq.queue) == 0 && !sq.closed {
		sq.cond.Wait()
	}

	if sq.closed {
		return nil, false
	}

	sq.heapMx.Lock()
	idx := heap.Pop(sq.queue).(int)
	sq.heapMx.Unlock()

	defer sq.queued.Delete(idx)

	res := sq.tasks[idx]

	TaskLog(res, fmt.Sprintf("<- POP task (%d)", res.Incarnation))

	return res, true
}

// Close closes the queue, causing NextTask to return false.
func (sq *taskQueue) Close() {
	sq.once.Do(func() {
		sq.condMx.Lock()
		defer sq.condMx.Unlock()
		sq.closed = true
		sq.cond.Broadcast()
	})
}

type taskHeap []int

func (h taskHeap) Len() int           { return len(h) }
func (h taskHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h taskHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *taskHeap) Push(x interface{}) {
	// Check if the integer already exists in the heap
	for _, item := range *h {
		if item == x.(int) {
			return
		}
	}
	// If it doesn't exist, append it
	*h = append(*h, x.(int))
	// Sort the heap
	sort.Ints(*h)
}

func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
