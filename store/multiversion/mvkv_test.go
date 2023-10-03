package multiversion_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store/cachekv"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	"github.com/cosmos/cosmos-sdk/store/types"
	scheduler "github.com/cosmos/cosmos-sdk/types/occ"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"
)

type MockParentStore struct{}

func TestVersionIndexedStoreGet(t *testing.T) {
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	parentKVStore := cachekv.NewStore(mem, types.NewKVStoreKey("mock"), 1000)
	mvs := multiversion.NewMultiVersionStore()
	// initialize a new VersionIndexedStore
	vis := multiversion.NewVersionIndexedStore(parentKVStore, mvs, 1, 2, make(chan scheduler.Abort))

	// mock a value in the parent store
	parentKVStore.Set([]byte("key1"), []byte("value1"))

	// read key that doesn't exist
	val := vis.Get([]byte("key2"))
	require.Nil(t, val)

	// read key that falls down to parent store
	val2 := vis.Get([]byte("key1"))
	require.Equal(t, []byte("value1"), val2)
	// verify value now in readset
	require.Equal(t, []byte("value1"), vis.GetReadset()["key1"])

	// read the same key that should now be served from the readset (can be verified by setting a different value for the key in the parent store)
	parentKVStore.Set([]byte("key1"), []byte("value2")) // realistically shouldn't happen, modifying to verify readset access
	val3 := vis.Get([]byte("key1"))
	require.Equal(t, []byte("value1"), val3)

	// test deleted value written to MVS but not parent store
	mvs.Delete(0, 2, []byte("delKey"))
	parentKVStore.Set([]byte("delKey"), []byte("value4"))
	valDel := vis.Get([]byte("delKey"))
	require.Nil(t, valDel)

	// set different key in MVS - for various indices
	mvs.Set(0, 2, []byte("key3"), []byte("value3"))
	mvs.Set(2, 1, []byte("key3"), []byte("value4"))
	mvs.SetEstimate(5, 0, []byte("key3"))

	// read the key that falls down to MVS
	val4 := vis.Get([]byte("key3"))
	// should equal value3 because value4 is later than the key in question
	require.Equal(t, []byte("value3"), val4)

	// try a read that falls through to MVS with a later tx index
	vis2 := multiversion.NewVersionIndexedStore(parentKVStore, mvs, 3, 2, make(chan scheduler.Abort))
	val5 := vis2.Get([]byte("key3"))
	// should equal value3 because value4 is later than the key in question
	require.Equal(t, []byte("value4"), val5)

	// test estimate values writing to abortChannel
	abortChannel := make(chan scheduler.Abort)
	vis3 := multiversion.NewVersionIndexedStore(parentKVStore, mvs, 6, 2, abortChannel)
	go func() {
		vis3.Get([]byte("key3"))
	}()
	abort := <-abortChannel // read the abort from the channel
	require.Equal(t, 5, abort.DependentTxIdx)
	require.Equal(t, scheduler.ErrReadEstimate, abort.Err)

	// TODO: also test values that are served from writeset - needs to be AFTER `set` is implemented
}
