package occ

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
	"testing"
)

type mockDeliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx

func (f mockDeliverTxFunc) DeliverTx(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
	return f(ctx, req)
}

func createTasks(n int) []*Task {
	tasks := make([]*Task, n)
	for i := 0; i < n; i++ {
		tasks[i] = NewTask(types.RequestDeliverTx{}, i)
	}
	return tasks
}

func TestProcessAll(t *testing.T) {
	tests := []struct {
		name           string
		workers        int
		tasks          []*Task
		deliverTxFunc  mockDeliverTxFunc
		expectedStatus TaskStatus
		expectedErr    error
		expectedInc    int
	}{
		{
			name:    "All tasks processed without aborts",
			workers: 2,
			tasks:   createTasks(5),
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				return types.ResponseDeliverTx{}
			},
			expectedStatus: TaskStatusValidated,
			expectedErr:    nil,
			expectedInc:    0,
		},
		//TODO: Add more test cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScheduler(tt.workers, tt.deliverTxFunc.DeliverTx)
			ctx := sdk.Context{}.WithContext(context.Background())

			err := s.ProcessAll(ctx, tt.tasks)
			if err != tt.expectedErr {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}

			for _, task := range tt.tasks {
				if task.Status != tt.expectedStatus {
					t.Errorf("Expected task status to be %s, got %s", tt.expectedStatus, task.Status)
				}
				if task.Incarnation != tt.expectedInc {
					t.Errorf("Expected task incarnation to be %d, got %d", tt.expectedInc, task.Incarnation)
				}
			}
		})
	}
}
