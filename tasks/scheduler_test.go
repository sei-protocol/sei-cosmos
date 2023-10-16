package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/store/cachekv"
	"github.com/cosmos/cosmos-sdk/store/cachemulti"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockDeliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx

func (f mockDeliverTxFunc) DeliverTx(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
	return f(ctx, req)
}

func requestList(n int) []types.RequestDeliverTx {
	tasks := make([]types.RequestDeliverTx, n)
	for i := 0; i < n; i++ {
		tasks[i] = types.RequestDeliverTx{}
	}
	return tasks
}

func TestProcessAll(t *testing.T) {
	tests := []struct {
		name          string
		workers       int
		requests      []types.RequestDeliverTx
		deliverTxFunc mockDeliverTxFunc
		expectedErr   error
	}{
		{
			name:     "All tasks processed without aborts",
			workers:  2,
			requests: requestList(5),
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				return types.ResponseDeliverTx{}
			},
			expectedErr: nil,
		},
		//TODO: Add more test cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScheduler(tt.workers, tt.deliverTxFunc.DeliverTx)
			ctx := sdk.Context{}.WithContext(context.Background())

			db := dbm.NewMemDB()
			storeKey := sdk.NewKVStoreKey("mock")
			mem := dbadapter.Store{DB: db}
			stores := make(map[sdk.StoreKey]sdk.CacheWrapper)
			stores[storeKey] = cachekv.NewStore(mem, storeKey, 1000)
			keys := make(map[string]sdk.StoreKey)
			store := cachemulti.NewStore(db, stores, keys, nil, nil, nil)
			ctx = ctx.WithMultiStore(store)

			res, err := s.ProcessAll(ctx, tt.requests)
			if err != tt.expectedErr {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			} else {
				// response for each request exists
				assert.Len(t, res, len(tt.requests))
			}
		})
	}
}
