package tasks

import (
	"fmt"
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
	NextTask(workerID int) (*TxTask, bool)
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
	// ValidateAll marks all tasks as pending validation.
	ValidateAll()
	// ValidateLaterTasks marks all tasks after the given index as pending validation.
	ValidateLaterTasks(afterIdx int)
	// IsCompleted returns true if all tasks have been executed and validated.
	IsCompleted() bool
	// DependenciesFinished returns whether all dependencies are finished
	DependenciesFinished(idx int) bool
}

type taskQueue struct {
	lockTimerID string
	qmx         sync.RWMutex
	once        sync.Once
	executing   IntSet
	finished    IntSet
	queueLen    atomic.Int64
	closed      bool
	workers     int
	shards      []chan int
	tasks       []*TxTask
}

func NewTaskQueue(tasks []*TxTask, workers int) Queue {
	shards := make([]chan int, 0, workers)
	for i := 0; i < workers; i++ {
		shards = append(shards, make(chan int, len(tasks)*2))
	}
	sq := &taskQueue{
		workers:   workers,
		tasks:     tasks,
		shards:    shards,
		finished:  newIntSet(len(tasks)), // newSyncSetMap(), //(len(tasks)),
		executing: newIntSet(len(tasks)),
	}
	return sq
}

func (sq *taskQueue) execute(idx int) {
	if sq.getTask(idx).SetTaskType(TypeExecution) {
		TaskLog(sq.getTask(idx), "-> execute")
		sq.finished.Delete(idx)
		sq.executing.Add(idx)
		sq.pushTask(idx, TypeExecution)
	}
}

func (sq *taskQueue) getTask(idx int) *TxTask {
	return sq.tasks[idx]
}

func (sq *taskQueue) validate(idx int) {
	task := sq.getTask(idx)
	if sq.getTask(idx).SetTaskType(TypeValidation) {
		TaskLog(task, "-> validate")
		sq.pushTask(idx, TypeValidation)
	}
}

func (sq *taskQueue) isExecuting(idx int) bool {
	return sq.executing.Exists(idx)
}

// FinishExecute marks a task as finished executing and transitions directly validation
func (sq *taskQueue) FinishExecute(idx int) {
	defer TaskLog(sq.getTask(idx), "-> finished task execute")

	//if !sq.isExecuting(idx) {
	//	TaskLog(sq.getTask(idx), "not executing, but trying to finish execute")
	//	panic("not executing, but trying to finish execute")
	//}

	sq.executing.Delete(idx)
	sq.validate(idx)
}

// FinishTask marks a task as finished if nothing else queued it
// this drives whether the queue thinks everything is done processing
func (sq *taskQueue) FinishTask(idx int) {
	sq.finished.Add(idx)
	TaskLog(sq.getTask(idx), "FinishTask -> task is FINISHED (for now)")
}

// ReValidate re-validates a task (back to queue from validation)
func (sq *taskQueue) ReValidate(idx int) {
	//if sq.isExecuting(idx) {
	//	TaskLog(sq.getTask(idx), "task is executing (unexpected)")
	//	panic("cannot re-validate an executing task")
	//}
	sq.validate(idx)
}

func (sq *taskQueue) Execute(idx int) {
	task := sq.tasks[idx]
	TaskLog(task, fmt.Sprintf("-> Execute (%d)", sq.getTask(idx).Incarnation))
	task.Increment()
	sq.execute(idx)
}

func (sq *taskQueue) ValidateAll() {
	for idx := 0; idx < len(sq.tasks); idx++ {
		sq.validate(idx)
	}
}

// ValidateLaterTasks marks all tasks after the given index as pending validation.
// any executing tasks are skipped
func (sq *taskQueue) ValidateLaterTasks(afterIdx int) {
	for idx := afterIdx + 1; idx < len(sq.tasks); idx++ {
		sq.validate(idx)
	}
}

func (sq *taskQueue) isFinished(idx int) bool {
	return sq.finished.Exists(idx) && sq.getTask(idx).IsStatus(statusValidated)
}

func (sq *taskQueue) DependenciesFinished(idx int) bool {
	for _, dep := range sq.getTask(idx).Dependencies {
		if !sq.isFinished(dep) {
			return false
		}
	}
	return true
}

// IsCompleted returns true if all tasks are "finished"
func (sq *taskQueue) IsCompleted() bool {
	queued := sq.queueLen.Load()
	if queued > 0 {
		return false
	}
	finished := sq.finished.Length()
	tasks := len(sq.tasks)
	if finished != tasks {
		return false
	}
	return true
}

func (sq *taskQueue) pushTask(idx int, taskType TaskType) {
	TaskLog(sq.getTask(idx), fmt.Sprintf("-> PUSH task (%s/%d)", taskType, sq.getTask(idx).Incarnation))
	sq.queueLen.Add(1)
	sq.qmx.RLock()
	defer sq.qmx.RUnlock()
	if sq.closed {
		TaskLog(sq.getTask(idx), "queue is closed")
		return
	}
	sq.shards[idx%sq.workers] <- idx
}

// ExecuteAll executes all tasks in the queue (called to start processing)
func (sq *taskQueue) ExecuteAll() {
	for idx := range sq.tasks {
		sq.execute(idx)
	}
}

// NextTask returns the next task to be executed, or nil if the queue is closed.
// this hangs if no tasks are ready because it's possible a new task might arrive
// closing the queue causes NextTask to return false immediately
func (sq *taskQueue) NextTask(workerID int) (*TxTask, bool) {
	idx, open := <-sq.shards[workerID]
	if !open {
		return nil, false
	}
	defer sq.queueLen.Add(-1)
	res := sq.getTask(idx)
	TaskLog(res, fmt.Sprintf("<- POP task (%d)", res.Incarnation))
	return res, true
}

// Close closes the queue, causing NextTask to return false.
func (sq *taskQueue) Close() {
	sq.once.Do(func() {
		sq.qmx.Lock()
		defer sq.qmx.Unlock()
		sq.closed = true
		for _, shard := range sq.shards {
			close(shard)
		}
	})
}
