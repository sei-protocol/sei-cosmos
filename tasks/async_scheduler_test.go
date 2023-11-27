package tasks

import (
	"errors"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
)

func TestAsyncProcessAll(t *testing.T) {
	tests := []struct {
		name          string
		workers       int
		runs          int
		requests      []*sdk.DeliverTxEntry
		deliverTxFunc deliverTxFunc
		addStores     bool
		expectedErr   error
		assertions    func(t *testing.T, ctx sdk.Context, res []types.ResponseDeliverTx)
	}{
		{
			name:      "Test every tx accesses same key",
			workers:   10,
			runs:      1,
			addStores: true,
			requests:  requestList(10),
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				// all txs read and write to the same key to maximize conflicts
				kv := ctx.MultiStore().GetKVStore(testStoreKey)
				val := string(kv.Get(itemKey))

				// if this is the first tx, the store should be empty
				status := "✅"
				if ctx.TxIndex() == 0 && val != "" {
					status = "❌"
				} else if ctx.TxIndex() > 0 && val != fmt.Sprintf("%d", ctx.TxIndex()-1) {
					status = "❌"
				}
				fmt.Printf("\t\t\t\t\tTASK(%s): IDX: %d, READ: %s, STATUS: %s\n", string(req.Tx), ctx.TxIndex(), val, status)

				// write to the store with this tx's index
				kv.Set(itemKey, req.Tx)

				// return what was read from the store (final attempt should be index-1)
				return types.ResponseDeliverTx{
					Info: val,
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []types.ResponseDeliverTx) {
				for idx, response := range res {
					if idx == 0 {
						require.Equal(t, "", response.Info)
					} else {
						// the info is what was read from the kv store by the tx
						// each tx writes its own index, so the info should be the index of the previous tx
						require.Equal(t, fmt.Sprintf("%d", idx-1), response.Info, fmt.Sprintf("INDEX %d", idx))
					}
				}
				// confirm last write made it to the parent store
				latest := ctx.MultiStore().GetKVStore(testStoreKey).Get(itemKey)
				require.Equal(t, fmt.Sprintf("%d", len(res)-1), string(latest))
			},
			expectedErr: nil,
		},
		//{
		//	name:      "Test no stores on context should not panic",
		//	workers:   1,
		//	runs:      1,
		//	addStores: false,
		//	requests:  requestList(1),
		//	deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
		//		return types.ResponseDeliverTx{
		//			Info: fmt.Sprintf("%d", ctx.TxIndex()),
		//		}
		//	},
		//	assertions: func(t *testing.T, ctx sdk.Context, res []types.ResponseDeliverTx) {
		//		for idx, response := range res {
		//			require.Equal(t, fmt.Sprintf("%d", idx), response.Info)
		//		}
		//	},
		//	expectedErr: nil,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.runs; i++ {
				s := NewAsyncScheduler(tt.workers, tt.deliverTxFunc)
				ctx := initTestCtx(tt.addStores)

				res, err := s.ProcessAll(ctx, tt.requests)
				require.Len(t, res, len(tt.requests))

				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				} else {
					tt.assertions(t, ctx, res)
				}
			}
		})
	}
}
