package occ

import (
	"errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
	"golang.org/x/sync/errgroup"
)

var (
	ErrReadEstimate = errors.New("multiversion store value contains estimate, cannot read, aborting")
)

// Abort contains the information for a transaction's conflict
type Abort struct {
	DependentTxIdx int
	Err            error
}

func NewEstimateAbort(dependentTxIdx int) Abort {
	return Abort{
		DependentTxIdx: dependentTxIdx,
		Err:            ErrReadEstimate,
	}
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusExecuted  TaskStatus = "executed"
	TaskStatusAborted   TaskStatus = "aborted"
	TaskStatusValidated TaskStatus = "validated"
)

type Task struct {
	Status      TaskStatus
	Index       int
	Incarnation int
	Request     types.RequestDeliverTx
	Response    *types.ResponseDeliverTx
}

// TODO: (TBD) this might not be necessary to externalize, unless others
func NewTask(request types.RequestDeliverTx, index int) *Task {
	return &Task{
		Request: request,
		Index:   index,
		Status:  TaskStatusPending,
	}
}

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, tasks []*Task) error
}

type scheduler struct {
	deliverTx func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)
	workers   int
}

// NewScheduler creates a new scheduler
func NewScheduler(workers int, deliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)) Scheduler {
	return &scheduler{
		workers:   workers,
		deliverTx: deliverTxFunc,
	}
}

func (s *scheduler) ProcessAll(ctx sdk.Context, tasks []*Task) error {
	toExecute := tasks
	for len(toExecute) > 0 {

		// execute sets statuses of tasks to either executed or aborted
		err := s.executeAll(ctx, toExecute)
		if err != nil {
			return err
		}

		// validate returns any that should be re-executed
		// note this processes ALL tasks, not just those recently executed
		toExecute, err = s.validateAll(ctx, tasks)
		if err != nil {
			return err
		}
		for _, t := range toExecute {
			t.Incarnation++
			t.Status = TaskStatusPending
			//TODO: reset anything that needs resetting
		}
	}
	return nil
}

// TODO: validate each task
// TODO: return list of tasks that are invalid
func (s *scheduler) validateAll(ctx sdk.Context, tasks []*Task) ([]*Task, error) {
	var res []*Task
	for _, t := range tasks {
		// any aborted tx is known to be suspect here
		if t.Status == TaskStatusAborted {
			res = append(res, t)
		} else {
			//TODO: validate the task and add it if invalid
			//TODO: create and handle abort for validation
			t.Status = TaskStatusValidated
		}
	}
	return res, nil
}

// ExecuteAll (SHELL) executes all tasks concurrently, and returns a result with all completed tasks and all conflicts
// TODO: retries on aborted tasks
// TODO: error scenarios
func (s *scheduler) executeAll(ctx sdk.Context, tasks []*Task) error {
	ch := make(chan *Task, len(tasks))
	grp, gCtx := errgroup.WithContext(ctx.Context())

	// a workers value < 1 means no limit
	workers := s.workers
	if s.workers < 1 {
		workers = len(tasks)
	}

	for i := 0; i < workers; i++ {
		grp.Go(func() error {
			for {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				case task := <-ch:
					//TODO: ensure version multi store is on context
					abortCh := make(chan *Abort)
					resp := s.deliverTx(ctx, task.Request)

					if _, ok := <-abortCh; ok {
						task.Status = TaskStatusAborted
					} else {
						task.Status = TaskStatusExecuted
						task.Response = &resp
					}
				}
			}
		})
	}
	grp.Go(func() error {
		defer close(ch)
		for _, task := range tasks {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case ch <- task:
			}
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		return err
	}

	return nil
}
