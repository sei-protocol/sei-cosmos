package occ

import (
	"errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
	"golang.org/x/sync/errgroup"
)

var (
	AbortCtxKey     = struct{}{}
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

type Task struct {
	Index    int
	Request  types.RequestDeliverTx
	Response types.ResponseDeliverTx
	abort    *Abort
}

// TODO: (TBD) this might not be necessary to externalize, unless others
// need to reason about aborts and dependencies
func NewTask(request types.RequestDeliverTx, index int) *Task {
	return &Task{
		Request: request,
		Index:   index,
	}
}

type Result struct {
	Conflicts []*Task
	Completed []*Task
}

type Scheduler interface {
	ExecuteAll(ctx sdk.Context, tasks []*Task) (*Result, error)
}

type scheduler struct {
	deliverTx func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)
	workers   int
}

// TODO: this may need a reference to the mkv store?
func NewScheduler(workers int, deliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) (res types.ResponseDeliverTx)) Scheduler {
	return &scheduler{
		workers:   workers,
		deliverTx: deliverTxFunc,
	}
}

// ExecuteAll (SHELL) executes all tasks concurrently, and returns a result with all completed tasks and all conflicts
// TODO: retries on aborted tasks
// TODO: validation logic
// TODO: error scenarios
func (s *scheduler) ExecuteAll(ctx sdk.Context, tasks []*Task) (*Result, error) {
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
				case task, ok := <-ch:
					if !ok {
						return nil
					}

					//TODO: putting abort channel on the context for now
					//I'm not yet sure how this should get to the mkv store
					task.abort = nil
					abortCh := make(chan *Abort)
					txCtx := ctx.WithValue(AbortCtxKey, abortCh)
					resp := s.deliverTx(txCtx, task.Request)

					// if this aborted, then mark as aborted
					//TODO: add to a dependnecies list for retry
					if abt, ok := <-abortCh; ok {
						task.abort = abt
					} else {
						task.Response = resp
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
		return nil, err
	}

	resp := &Result{}

	//TODO: actually do validation (TBD)
	//TODO: retries on aborted tasks (TBD)
	for _, task := range tasks {
		if task.abort != nil {
			resp.Conflicts = append(resp.Conflicts, task)
		} else {
			resp.Completed = append(resp.Completed, task)
		}
	}

	return resp, nil
}
