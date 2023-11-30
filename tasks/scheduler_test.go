package tasks

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tm-db"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/cosmos/cosmos-sdk/store/cachekv"
	"github.com/cosmos/cosmos-sdk/store/cachemulti"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/utils/tracing"
)

type mockDeliverTxFunc func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx

var testStoreKey = sdk.NewKVStoreKey("mock")
var itemKey = []byte("key")

func requestList(n int) []*sdk.DeliverTxEntry {
	tasks := make([]*sdk.DeliverTxEntry, n)
	for i := 0; i < n; i++ {
		tasks[i] = &sdk.DeliverTxEntry{
			Request: types.RequestDeliverTx{
				Tx: []byte(fmt.Sprintf("%d", i)),
			},
		}

	}
	return tasks
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

func TestExplicitOrdering(t *testing.T) {
	tests := []struct {
		name     string
		scenario func(s *scheduler, ctx sdk.Context, tasks []*TxTask)
	}{
		{
			name: "Test perfect order",
			scenario: func(s *scheduler, ctx sdk.Context, tasks []*TxTask) {
				// STARTING HERE
				// reads nil, writes 0
				s.executeTask(tasks[0])
				s.validateTask(ctx, tasks[0])

				// reads 0, writes 1
				s.executeTask(tasks[1])
				s.validateTask(ctx, tasks[1])

				// reads the expected things
				require.Equal(t, "", tasks[0].Response.Info)
				require.Equal(t, "0", tasks[1].Response.Info)

				// both validated
				require.Equal(t, statusValidated, tasks[0].status)
				require.Equal(t, statusValidated, tasks[1].status)
			},
		},
	}
	for _, test := range tests {
		deliverTx := func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
			// all txs read and write to the same key to maximize conflicts
			kv := ctx.MultiStore().GetKVStore(testStoreKey)

			val := string(kv.Get(itemKey))
			kv.Set(itemKey, []byte(fmt.Sprintf("%d", ctx.TxIndex())))

			// return what was read from the store (final attempt should be index-1)
			return types.ResponseDeliverTx{
				Info: val,
			}
		}
		tp := trace.NewNoopTracerProvider()
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		tr := tp.Tracer("scheduler-test")
		ti := &tracing.Info{
			Tracer: &tr,
		}
		s := &scheduler{
			deliverTx:   deliverTx,
			tracingInfo: ti,
		}
		ctx := initTestCtx(true)
		s.initMultiVersionStore(ctx)

		tasks := generateTasks(2)
		for _, tsk := range tasks {
			tsk.Ctx = ctx
		}
		test.scenario(s, ctx, tasks)
	}

}

func TestProcessAll(t *testing.T) {
	tests := []struct {
		name          string
		workers       int
		runs          int
		requests      []*sdk.DeliverTxEntry
		deliverTxFunc mockDeliverTxFunc
		addStores     bool
		expectedErr   error
		assertions    func(t *testing.T, ctx sdk.Context, res []types.ResponseDeliverTx)
	}{
		{
			name:      "Test no overlap txs",
			workers:   5,
			runs:      100,
			addStores: true,
			requests:  requestList(100),
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				// all txs read and write to the same key to maximize conflicts
				kv := ctx.MultiStore().GetKVStore(testStoreKey)

				// write to the store with this tx's index
				kv.Set(req.Tx, req.Tx)
				val := string(kv.Get(req.Tx))

				// return what was read from the store (final attempt should be index-1)
				return types.ResponseDeliverTx{
					Info: val,
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []types.ResponseDeliverTx) {
				for idx, response := range res {
					require.Equal(t, fmt.Sprintf("%d", idx), response.Info)
				}
				store := ctx.MultiStore().GetKVStore(testStoreKey)
				for i := 0; i < len(res); i++ {
					val := store.Get([]byte(fmt.Sprintf("%d", i)))
					require.Equal(t, []byte(fmt.Sprintf("%d", i)), val)
				}
			},
			expectedErr: nil,
		},
		{
			name:      "Test every tx accesses same key",
			workers:   5,
			runs:      100,
			addStores: true,
			requests:  requestList(100),
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
			assertions: func(t *testing.T, ctx sdk.Context, res []types.ResponseDeliverTx) {
				for idx, response := range res {
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
				require.Equal(t, []byte(fmt.Sprintf("%d", len(res)-1)), latest)
			},
			expectedErr: nil,
		},
		{
			name:      "Test no stores on context should not panic",
			workers:   50,
			runs:      1,
			addStores: false,
			requests:  requestList(50),
			deliverTxFunc: func(ctx sdk.Context, req types.RequestDeliverTx) types.ResponseDeliverTx {
				return types.ResponseDeliverTx{
					Info: fmt.Sprintf("%d", ctx.TxIndex()),
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []types.ResponseDeliverTx) {
				for idx, response := range res {
					require.Equal(t, fmt.Sprintf("%d", idx), response.Info)
				}
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.runs; i++ {
				// set a tracer provider
				tp := trace.NewNoopTracerProvider()
				otel.SetTracerProvider(trace.NewNoopTracerProvider())
				tr := tp.Tracer("scheduler-test")
				ti := &tracing.Info{
					Tracer: &tr,
				}

				s := NewScheduler(tt.workers, ti, tt.deliverTxFunc)
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
