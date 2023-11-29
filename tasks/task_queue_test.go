package tasks

import (
	"testing"
)

func generateTasks(count int) []*deliverTxTask {
	var res []*deliverTxTask
	for i := 0; i < count; i++ {
		res = append(res, &deliverTxTask{Index: i})
	}
	return res
}

func TestNewSchedulerQueue(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	if len(sq.tasks) != len(tasks) {
		t.Errorf("Expected tasks length %d, but got %d", len(tasks), len(sq.tasks))
	}
}

func TestSetToIdle(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	sq.AddAllTasksToExecutionQueue()
	sq.SetToIdle(1)

	if !sq.tasks[1].IsTaskType(TypeIdle) {
		t.Errorf("Expected task type %d, but got %d", TypeIdle, sq.tasks[1].TaskType())
	}
}

func TestNextTask(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	sq.AddAllTasksToExecutionQueue()
	task, _ := sq.NextTask()

	if task != sq.tasks[1] {
		t.Errorf("Expected task %v, but got %v", sq.tasks[1], task)
	}
}

func TestClose(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	sq.Close()

	if sq.closed != true {
		t.Errorf("Expected closed to be true, but got %v", sq.closed)
	}
}

func TestNextTaskOrder(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	// Add tasks in non-sequential order
	sq.AddAllTasksToExecutionQueue()
	tsk, _ := sq.NextTask()
	sq.ReExecute(tsk.Index)

	// The task with the lowest index should be returned first
	task, _ := sq.NextTask()
	if task != sq.tasks[1] {
		t.Errorf("Expected task %v, but got %v", sq.tasks[1], task)
	}
}
