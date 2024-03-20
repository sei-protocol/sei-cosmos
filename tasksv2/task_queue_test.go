package tasksv2

import (
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/abci/types"
	"testing"
)

func generateTasks(count int) []*TxTask {
	var res []*TxTask
	for i := 0; i < count; i++ {
		res = append(res, &TxTask{AbsoluteIndex: i,
			Parents: &intSetMap{
				m: make(map[int]struct{}),
			},
			Dependents: &intSetMap{
				m: make(map[int]struct{}),
			}})
	}
	return res
}

func assertExecuting(t *testing.T, task *TxTask) {
	assert.Equal(t, TypeExecution, task.taskType)
}

func assertValidating(t *testing.T, task *TxTask) {
	assert.Equal(t, TypeValidation, task.taskType)
}

func testQueue() (Queue, []*TxTask) {
	tasks := generateTasks(10)
	return NewTaskQueue(tasks, 1), tasks
}

func TestSchedulerQueue(t *testing.T) {
	queue, tasks := testQueue()

	// Test ExecuteAll
	queue.ExecuteAll()
	for _, task := range tasks {
		assertExecuting(t, task)
	}

	// Test NextTask
	nextTask, ok := queue.NextTask(0)
	assert.True(t, ok)
	assert.Equal(t, tasks[0], nextTask)

	// Test Close
	queue.Close()
	for ok {
		nextTask, ok = queue.NextTask(0)
	}
	assert.False(t, ok)

	// Test FinishExecute leads to Validation
	queue, tasks = testQueue()
	queue.ExecuteAll()
	nextTask, ok = queue.NextTask(0)
	assert.True(t, ok)
	nextTask.PopTaskType()
	// mock so it's not nil, because no execution actually happened here
	nextTask.Response = &types.ResponseDeliverTx{}
	queue.FinishExecute(nextTask.AbsoluteIndex)
	assertValidating(t, nextTask)

	// Test that validation doesn't happen for executing task
	queue, tasks = testQueue()
	queue.ExecuteAll()
	queue.ValidateLaterTasks(-1)
	nextTask, ok = queue.NextTask(0)
	assert.True(t, ok)
	assertExecuting(t, nextTask) // still executing

	// Test that validation happens for finished tasks
	queue, tasks = testQueue()
	queue.ExecuteAll()
	queue.ValidateLaterTasks(-1)
	nextTask, ok = queue.NextTask(0)
	assert.True(t, ok)
	assertExecuting(t, nextTask)

	// Test IsCompleted
	queue, tasks = testQueue()
	queue.ExecuteAll()

	for idx, task := range tasks {
		task.SetStatus(statusValidated)
		queue.NextTask(0)
		queue.FinishTask(idx)
		if idx == len(tasks)-1 {
			queue.Close()
		}
	}
	assert.True(t, queue.IsCompleted())
}
