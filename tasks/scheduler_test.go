package tasks

import (
	"context"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	"github.com/cosmos/cosmos-sdk/types/occ"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/store/cachekv"
	"github.com/cosmos/cosmos-sdk/store/cachemulti"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockDeliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx

var testStoreKey = sdk.NewKVStoreKey("mock")
var itemKey = []byte("key")

func (f mockDeliverTxFunc) DeliverTx(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
	return f(ctx, req)
}

func requestList(n int) []types.RequestDeliverTx {
	tasks := make([]types.RequestDeliverTx, n)
	for i := 0; i < n; i++ {
		tasks[i] = types.RequestDeliverTx{
			Tx: []byte(fmt.Sprintf("%d", i)),
		}
	}
	return tasks
}

func initTestCtx() sdk.Context {
	ctx := sdk.Context{}.WithContext(context.Background())
	db := dbm.NewMemDB()
	mem := dbadapter.Store{DB: db}
	stores := make(map[sdk.StoreKey]sdk.CacheWrapper)
	stores[testStoreKey] = cachekv.NewStore(mem, testStoreKey, 1000)
	keys := make(map[string]sdk.StoreKey)
	keys[testStoreKey.Name()] = testStoreKey
	store := cachemulti.NewStore(db, stores, keys, nil, nil, nil)
	ctx = ctx.WithMultiStore(&store)
	return ctx
}

func toInt(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return res
}

type tx struct {
	abortCh chan occ.Abort
	vs      *multiversion.VersionIndexedStore
}

func TestSpecificScenario(t *testing.T) {
	ctx := initTestCtx()
	mvs := multiversion.NewMultiVersionStore(ctx.MultiStore().GetKVStore(testStoreKey))

	var txs []tx
	for i := 0; i < 2; i++ {
		a := make(chan occ.Abort, 100)
		v := mvs.VersionedIndexedStore(0, i, a)
		txs = append(txs, tx{a, v})
	}
	txs[1].vs.Get(itemKey)
	txs[1].vs.Set(itemKey, []byte("1"))
	txs[1].vs.WriteToMultiVersionStore()

	res := txs[0].vs.Get(itemKey)
	require.Len(t, res, 0)

	txs[0].vs.Set(itemKey, []byte("0"))

	c0 := mvs.ValidateTransactionState(0)
	c1 := mvs.ValidateTransactionState(1)

	fmt.Println(c0)
	fmt.Println(c1)
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
			name:     "Test for conflicts",
			workers:  5,
			requests: requestList(5),
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				kv := ctx.MultiStore().GetKVStore(testStoreKey)
				val := string(kv.Get(itemKey))

				// write to the store with this tx's index
				kv.Set(itemKey, req.Tx)

				// return what was read from the store
				return types.ResponseDeliverTx{
					Info: val,
				}
			},
			expectedErr: nil,
		},
		//TODO: Add more test cases
	}

	//TODO: remove logs once we figure out why this fails
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				s := NewScheduler(tt.workers, tt.deliverTxFunc.DeliverTx)
				ctx := initTestCtx()

				res, err := s.ProcessAll(ctx, tt.requests)
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				} else {
					require.Len(t, res, len(tt.requests))
					for idx, response := range res {
						if idx == 0 {
							require.Equal(t, "", response.Info)
						} else {
							// the info is what was read from the kv store by the tx
							// each tx writes its own index, so the info should be the index of the previous tx
							require.Equal(t, fmt.Sprintf("%d", idx-1), response.Info)
						}
					}
				}
			}

		})
	}
}
