package tasks

import (
	"container/heap"
	"fmt"
	"sync"
)

type TaskType int

var exists = struct{}{}

const (
	TypeIdle TaskType = iota
	TypeExecution
	TypeValidation
)

type taskHeap []int

func (h taskHeap) Len() int           { return len(h) }
func (h taskHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h taskHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *taskHeap) Push(x interface{}) {
	*h = append(*h, x.(int))
}

func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type SchedulerQueue struct {
	tasks   []*TxTask
	queue   taskHeap
	taskMap map[int]struct{}
	mx      sync.Mutex
	cond    *sync.Cond
	closed  bool
}

func NewSchedulerQueue(tasks []*TxTask) *SchedulerQueue {
	q := &SchedulerQueue{
		tasks:   tasks,
		taskMap: make(map[int]struct{}),
	}
	q.cond = sync.NewCond(&q.mx)
	return q
}

func (sq *SchedulerQueue) AddToExecution(idx int, increment bool) {

	sq.Lock()
	defer sq.Unlock()

	sq.tasks[idx].Lock()
	defer sq.tasks[idx].Unlock()

	if sq.tasks[idx].TaskType == TypeIdle {
		fmt.Printf("TASK(%d): -> EXECUTE\n", idx)
		sq.tasks[idx].TaskType = TypeExecution
		sq.tasks[idx].SetStatus(statusExecuting)
		if increment {
			sq.tasks[idx].Increment()
		}
		sq.taskMap[idx] = exists
		heap.Push(&sq.queue, idx)
		sq.cond.Broadcast()
	} else {
		fmt.Printf("TASK(%d): (not executing (status=%s, type=%d/%d))\n", idx, sq.tasks[idx].Status(), sq.tasks[idx].TaskType)
	}
}

func (sq *SchedulerQueue) Exists(i int) bool {
	// must be called within a sq lock
	_, ok := sq.taskMap[i]
	return ok
}

func (sq *SchedulerQueue) AddAllTasksToValidation() {
	sq.AddLaterTasksToValidation(-1)
}

func (sq *SchedulerQueue) AddLaterTasksToValidation(after int) {
	sq.Lock()
	defer sq.Unlock()

	var anyAdded bool
	for idx := after + 1; idx < len(sq.tasks); idx++ {
		if sq.Exists(idx) || sq.tasks[idx].IsExecuting() {
			continue
		}
		sq.tasks[idx].Lock()

		sq.taskMap[idx] = exists
		fmt.Printf("TASK(%d): -> (after %d) VALIDATE\n", after, idx)
		sq.tasks[idx].TaskType = TypeValidation
		sq.tasks[idx].SetStatus(statusValidating)
		heap.Push(&sq.queue, idx)
		anyAdded = true

		sq.tasks[idx].Unlock()

	}
	if anyAdded {
		sq.cond.Broadcast()
	}
}

func (sq *SchedulerQueue) Lock() {
	sq.mx.Lock()
}

func (sq *SchedulerQueue) Unlock() {
	sq.mx.Unlock()
}

func (sq *SchedulerQueue) AddToValidation(idx int) {
	fmt.Printf("TASK(%d): AddToValidation\n", idx)

	sq.Lock()
	defer sq.Unlock()

	sq.tasks[idx].Lock()
	defer sq.tasks[idx].Unlock()

	if sq.Exists(idx) || sq.tasks[idx].IsExecuting() {
		return
	}

	fmt.Printf("TASK(%d): -> VALIDATE\n", idx)
	sq.tasks[idx].TaskType = TypeValidation
	sq.tasks[idx].SetStatus(statusValidating)
	heap.Push(&sq.queue, idx)
	sq.cond.Broadcast()
}

func (sq *SchedulerQueue) Complete(idx int) {
	sq.Lock()
	defer sq.Unlock()

	//fmt.Println(fmt.Sprintf("TASK(%d): COMPLETE ", idx))
	sq.tasks[idx].TaskType = TypeIdle
}

func (sq *SchedulerQueue) NextTask() (*TxTask, bool) {
	cancel := waitWithMsg("NextTask()")
	defer cancel()

	sq.Lock()
	defer sq.Unlock()

	for len(sq.queue) == 0 && !sq.closed {
		sq.cond.Wait()
	}

	if sq.closed {
		return nil, false
	}

	idx := heap.Pop(&sq.queue).(int)
	delete(sq.taskMap, idx)
	return sq.tasks[idx], true
}

func (sq *SchedulerQueue) Len() int {
	cancel := waitWithMsg("Len()")
	defer cancel()

	sq.Lock()
	defer sq.Unlock()
	return sq.queue.Len()
}

func (sq *SchedulerQueue) Close() {
	cancel := waitWithMsg("Close()")
	defer cancel()
	sq.Lock()
	defer sq.Unlock()
	sq.closed = true
	sq.cond.Broadcast()
}
