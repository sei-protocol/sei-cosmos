package tasks

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
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
	lockTimerID   string
	mx            sync.Mutex
	condMx        sync.Mutex
	heapMx        sync.Mutex
	cond          *sync.Cond
	once          sync.Once
	executing     sync.Map
	finished      sync.Map
	finishedCount atomic.Int32

	out   chan int
	tasks []*TxTask
	queue *taskHeap
	timer *Timer
}

func NewTaskQueue(tasks []*TxTask) Queue {
	sq := &taskQueue{
		tasks: tasks,
		queue: &taskHeap{},
		timer: NewTimer("Queue"),
		out:   make(chan int, len(tasks)),
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
	if sq.getTask(idx).SetTaskType(TypeExecution) {
		TaskLog(sq.getTask(idx), "-> execute")

		if sq.isFinished(idx) {
			sq.finished.Delete(idx)
			sq.finishedCount.Add(-1)
		}

		sq.executing.Store(idx, struct{}{})
		sq.pushTask(idx, TypeExecution)
	}
}

func (sq *taskQueue) getTask(idx int) *TxTask {
	return sq.tasks[idx]
}

func (sq *taskQueue) validate(idx int) {
	task := sq.getTask(idx)
	if sq.isExecuting(idx) {
		TaskLog(task, "(skip validating, executing...)")
		return
	}
	if sq.getTask(idx).SetTaskType(TypeValidation) {
		TaskLog(task, "-> validate")
		sq.pushTask(idx, TypeValidation)
	}
}

func (sq *taskQueue) isExecuting(idx int) bool {
	_, ok := sq.executing.Load(idx)
	return ok
}

// FinishExecute marks a task as finished executing and transitions directly validation
func (sq *taskQueue) FinishExecute(idx int) {
	id := sq.timer.Start("FinishExecute")
	defer sq.timer.End("FinishExecute", id)

	TaskLog(sq.getTask(idx), "-> finish task execute")

	if !sq.isExecuting(idx) {
		TaskLog(sq.getTask(idx), "not executing, but trying to finish execute")
		panic("not executing, but trying to finish execute")
	}

	sq.executing.Delete(idx)
	sq.validate(idx)
}

// FinishTask marks a task as finished if nothing else queued it
// this drives whether the queue thinks everything is done processing
func (sq *taskQueue) FinishTask(idx int) {
	if sq.isFinished(idx) {
		return
	}

	id := sq.timer.Start("FinishTask")
	defer sq.timer.End("FinishTask", id)

	TaskLog(sq.getTask(idx), "FinishTask -> task is FINISHED (for now)")
	sq.finishedCount.Add(1)
	sq.finished.Store(idx, struct{}{})
}

// ReValidate re-validates a task (back to queue from validation)
func (sq *taskQueue) ReValidate(idx int) {
	id := sq.timer.Start("ReValidate")
	defer sq.timer.End("ReValidate", id)

	if sq.isExecuting(idx) {
		TaskLog(sq.getTask(idx), "task is executing (unexpected)")
		panic("cannot re-validate an executing task")
	}

	sq.validate(idx)
}

func (sq *taskQueue) Execute(idx int) {
	id := sq.timer.Start("Execute")
	defer sq.timer.End("Execute", id)

	//TODO: might need lock here
	task := sq.tasks[idx]
	TaskLog(task, fmt.Sprintf("-> Execute (%d)", sq.getTask(idx).Incarnation))
	task.Increment()
	sq.execute(idx)
}

// ValidateLaterTasks marks all tasks after the given index as pending validation.
// any executing tasks are skipped
func (sq *taskQueue) ValidateLaterTasks(afterIdx int) {
	id := sq.timer.Start("ValidateLaterTasks")
	defer sq.timer.End("ValidateLaterTasks", id)

	for idx := afterIdx + 1; idx < len(sq.tasks); idx++ {
		sq.validate(idx)
	}
}

func (sq *taskQueue) isFinished(idx int) bool {
	id := sq.timer.Start("isFinished")
	defer sq.timer.End("isFinished", id)

	_, ok := sq.finished.Load(idx)
	return ok && sq.getTask(idx).IsStatus(statusValidated)
}

func (sq *taskQueue) DependenciesFinished(idx int) bool {
	id := sq.timer.Start("DependenciesFinished")
	defer sq.timer.End("DependenciesFinished", id)
	for _, dep := range sq.getTask(idx).Dependencies {
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
	fc := sq.finishedCount.Load()
	return fc == int32(len(sq.tasks))
}

func (sq *taskQueue) pushTask(idx int, taskType TaskType) {
	id := sq.timer.Start("pushTask")
	defer sq.timer.End("pushTask", id)
	TaskLog(sq.getTask(idx), fmt.Sprintf("-> PUSH task (%s/%d)", taskType, sq.getTask(idx).Incarnation))
	sq.out <- idx
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
	idx, open := <-sq.out
	if !open {
		return nil, false
	}
	res := sq.getTask(idx)
	TaskLog(res, fmt.Sprintf("<- POP task (%d)", res.Incarnation))
	return res, true
}

// Close closes the queue, causing NextTask to return false.
func (sq *taskQueue) Close() {
	sq.once.Do(func() {
		close(sq.out)
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
