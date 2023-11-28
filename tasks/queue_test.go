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

func TestAddValidationTask(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	sq.AddValidationTask(1)

	if sq.tasks[1].Type != TypeValidation {
		t.Errorf("Expected task type %d, but got %d", TypeValidation, sq.tasks[1].Type)
	}
}

func TestAddExecutionTask(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	sq.AddExecutionTask(1)

	if sq.tasks[1].Type != TypeExecution {
		t.Errorf("Expected task type %d, but got %d", TypeExecution, sq.tasks[1].Type)
	}
}

func TestSetToIdle(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	sq.AddExecutionTask(1)
	sq.SetToIdle(1)

	if sq.tasks[1].Type != TypeIdle {
		t.Errorf("Expected task type %d, but got %d", TypeIdle, sq.tasks[1].Type)
	}
}

func TestNextTask(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	sq.AddExecutionTask(1)
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
	sq.AddExecutionTask(3)
	sq.AddExecutionTask(1)
	sq.AddExecutionTask(2)
	sq.AddExecutionTask(4)

	// The task with the lowest index should be returned first
	task, _ := sq.NextTask()
	if task != sq.tasks[1] {
		t.Errorf("Expected task %v, but got %v", sq.tasks[1], task)
	}
}

func TestAddValidationTaskWhenActive(t *testing.T) {
	tasks := generateTasks(10)
	sq := NewSchedulerQueue(tasks, 5)

	// Add task to execution queue
	sq.AddExecutionTask(1)
	// Try to add the same task to validation queue
	sq.AddValidationTask(1)

	// Verify that the task's type is still TypeExecution
	if sq.tasks[1].Type != TypeExecution {
		t.Errorf("Expected task type %d, but got %d", TypeExecution, sq.tasks[1].Type)
	}

	// Add task to validation queue
	sq.AddValidationTask(2)
	// Try to add the same task to validation queue again
	sq.AddValidationTask(2)

	// Verify that the task's type is still TypeValidation
	if sq.tasks[2].Type != TypeValidation {
		t.Errorf("Expected task type %d, but got %d", TypeValidation, sq.tasks[2].Type)
	}
}
