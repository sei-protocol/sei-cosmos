package tasks

import (
	"context"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/cachemulti"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/store/cachekv"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockDeliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx

var testStoreKey = sdk.NewKVStoreKey("mock")
var itemKey = []byte("key")

func requestList(n int, addStores bool) sdk.DeliverTxBatchRequest {
	tasks := make([]*sdk.DeliverTxEntry, n)
	for i := 0; i < n; i++ {
		tasks[i] = &sdk.DeliverTxEntry{
			Request: types.RequestDeliverTx{
				Tx: []byte(fmt.Sprintf("%d", i)),
			},
			Context: initTestCtx(addStores),
		}
	}
	return sdk.DeliverTxBatchRequest{
		TxEntries: tasks,
	}
}

func initTestCtx(injectStores bool) sdk.Context {
	ctx := sdk.Context{}.WithContext(context.Background())
	keys := make(map[string]sdk.StoreKey)
	stores := make(map[sdk.StoreKey]sdk.CacheWrapper)
	db := dbm.NewMemDB()
	if injectStores {
		mem := dbadapter.Store{DB: db}
		stores[testStoreKey] = cachekv.NewStore(mem, testStoreKey, 1000)
		keys[testStoreKey.Name()] = testStoreKey
	}
	store := cachemulti.NewStore(db, stores, keys, nil, nil, nil)
	ctx = ctx.WithMultiStore(&store)
	return ctx
}

func TestProcessAll(t *testing.T) {
	tests := []struct {
		name          string
		workers       int
		runs          int
		transactions  int
		addStores     bool
		deliverTxFunc mockDeliverTxFunc
		expectedErr   error
		assertions    func(t *testing.T, ctx sdk.Context, res sdk.DeliverTxBatchResponse)
	}{
		{
			name:         "Test every tx accesses same key",
			workers:      50,
			runs:         25,
			transactions: 50,
			addStores:    true,
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				// all txs read and write to the same key to maximize conflicts
				kv := ctx.MultiStore().GetKVStore(testStoreKey)
				val := string(kv.Get(itemKey))

				// write to the store with this tx's index
				kv.Set(itemKey, req.Tx)

				// return what was read from the store (final attempt should be index-1)
				return types.ResponseDeliverTx{
					Info: val,
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res sdk.DeliverTxBatchResponse) {
				for idx, resp := range res.Results {
					response := resp.Response
					if idx == 0 {
						require.Equal(t, "", response.Info)
					} else {
						// the info is what was read from the kv store by the tx
						// each tx writes its own index, so the info should be the index of the previous tx
						require.Equal(t, fmt.Sprintf("%d", idx-1), response.Info)
					}
				}
				// confirm last write made it to the parent store
				latest := ctx.MultiStore().GetKVStore(testStoreKey).Get(itemKey)
				require.Equal(t, []byte("49"), latest)
			},
			expectedErr: nil,
		},
		{
			name:         "Test no stores on context should not panic",
			workers:      50,
			runs:         1,
			transactions: 50,
			addStores:    false,
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				return types.ResponseDeliverTx{
					Info: fmt.Sprintf("%d", ctx.TxIndex()),
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res sdk.DeliverTxBatchResponse) {
				for idx, response := range res.Results {
					require.Equal(t, fmt.Sprintf("%d", idx), response.Response.Info)
				}
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.runs; i++ {
				requests := requestList(tt.transactions, tt.addStores)
				s := NewScheduler(tt.workers, tt.deliverTxFunc)
				// use first ctx
				ctx := requests.TxEntries[0].Context

				res, err := s.ProcessAll(ctx, requests)
				require.Len(t, res.Results, len(requests.TxEntries))

				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				} else {
					tt.assertions(t, ctx, res)
				}
			}
		})
	}
}
