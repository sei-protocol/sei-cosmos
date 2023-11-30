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
	lockTimerID string
	mx          sync.Mutex
	condMx      sync.Mutex
	heapMx      sync.Mutex
	cond        *sync.Cond
	once        sync.Once
	executing   map[int]struct{}
	finished    sync.Map
	tasks       []*TxTask
	queue       *taskHeap
	timer       *Timer
	closed      bool
}

func NewTaskQueue(tasks []*TxTask) Queue {
	sq := &taskQueue{
		tasks:     tasks,
		queue:     &taskHeap{},
		timer:     NewTimer("Queue"),
		executing: make(map[int]struct{}),
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
		sq.executing[idx] = struct{}{}
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

func (sq *taskQueue) isExecuting(idx int) bool {
	_, ok := sq.executing[idx]
	return ok
}

// FinishExecute marks a task as finished executing and transitions directly validation
func (sq *taskQueue) FinishExecute(idx int) {
	id := sq.timer.Start("FinishExecute")
	defer sq.timer.End("FinishExecute", id)

	id2 := sq.timer.Start("FinishExecute-LOCK")
	sq.lock()
	defer func() {
		sq.unlock()
		sq.timer.End("FinishExecute-LOCK", id2)
	}()

	TaskLog(sq.tasks[idx], "-> finish task execute")

	if !sq.isExecuting(idx) {
		TaskLog(sq.tasks[idx], "not executing, but trying to finish execute")
		panic("not executing, but trying to finish execute")
	}

	delete(sq.executing, idx)
	sq.validate(idx)
}

// FinishTask marks a task as finished if nothing else queued it
// this drives whether the queue thinks everything is done processing
func (sq *taskQueue) FinishTask(idx int) {
	id := sq.timer.Start("FinishTask")
	defer sq.timer.End("FinishTask", id)
	id2 := sq.timer.Start("FinishTask-LOCK")
	sq.lock()
	defer func() {
		sq.unlock()
		sq.timer.End("FinishTask-LOCK", id2)
	}()

	TaskLog(sq.tasks[idx], "FinishTask -> task is FINISHED (for now)")

	sq.finished.Store(idx, struct{}{})
}

// ReValidate re-validates a task (back to queue from validation)
func (sq *taskQueue) ReValidate(idx int) {
	id := sq.timer.Start("ReValidate")
	defer sq.timer.End("ReValidate", id)
	sq.lock()
	defer sq.unlock()

	if sq.isExecuting(idx) {
		TaskLog(sq.tasks[idx], "task is executing (unexpected)")
		panic("cannot re-validate an executing task")
	}

	sq.validate(idx)
}

func (sq *taskQueue) Execute(idx int) {
	id := sq.timer.Start("Execute-full")
	defer sq.timer.End("Execute-full", id)
	id3 := sq.timer.Start("Execute-LOCK")
	sq.lock()
	defer func() {
		sq.unlock()
		sq.timer.End("Execute-LOCK", id3)
	}()

	id2 := sq.timer.Start("Execute-logic")
	defer sq.timer.End("Execute-logic", id2)

	TaskLog(sq.tasks[idx], fmt.Sprintf("-> Execute (%d)", sq.tasks[idx].Incarnation))

	sq.tasks[idx].Increment()
	sq.execute(idx)
}

// ValidateLaterTasks marks all tasks after the given index as pending validation.
// any executing tasks are skipped
func (sq *taskQueue) ValidateLaterTasks(afterIdx int) {
	id := sq.timer.Start("ValidateLaterTasks")
	defer sq.timer.End("ValidateLaterTasks", id)

	for idx := afterIdx + 1; idx < len(sq.tasks); idx++ {
		sq.lock()
		sq.validate(idx)
		sq.unlock()
	}
}

func (sq *taskQueue) isFinished(idx int) bool {
	id := sq.timer.Start("isFinished")
	defer sq.timer.End("isFinished", id)

	_, ok := sq.finished.Load(idx)
	return ok && sq.tasks[idx].IsStatus(statusValidated)
}

func (sq *taskQueue) DependenciesFinished(idx int) bool {
	id := sq.timer.Start("DependenciesFinished")
	defer sq.timer.End("DependenciesFinished", id)
	for _, dep := range sq.tasks[idx].Dependencies {
		if !sq.isFinished(dep) {
			return false
		}
	}
	return true
}

// IsCompleted returns true if all tasks are "finished"
func (sq *taskQueue) IsCompleted() bool {
	id := sq.timer.Start("IsCompleted")
	defer sq.timer.End("IsCompleted", id)
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
	id := sq.timer.Start("pushTask")
	defer sq.timer.End("pushTask", id)
	sq.condMx.Lock()
	defer sq.condMx.Unlock()
	TaskLog(sq.tasks[idx], fmt.Sprintf("-> PUSH task (%s/%d)", taskType, sq.tasks[idx].Incarnation))
	heap.Push(sq.queue, idx)
	sq.cond.Broadcast()
}

// ExecuteAll executes all tasks in the queue (called to start processing)
func (sq *taskQueue) ExecuteAll() {
	id := sq.timer.Start("ExecuteAll")
	defer sq.timer.End("ExecuteAll", id)

	for idx := range sq.tasks {
		sq.lock()
		sq.execute(idx)
		sq.unlock()
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
		sq.timer.PrintReport()
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
