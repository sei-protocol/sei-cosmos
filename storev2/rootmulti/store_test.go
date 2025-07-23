package rootmulti

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store/cache"
	"github.com/cosmos/cosmos-sdk/store/types"
	"github.com/sei-protocol/sei-db/config"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestLastCommitID(t *testing.T) {
	store := NewStore(t.TempDir(), log.NewNopLogger(), config.StateCommitConfig{}, config.StateStoreConfig{}, false)
	require.Equal(t, types.CommitID{}, store.LastCommitID())
}

func TestInterBlockCacheMismatchDuringCommit(t *testing.T) {
	// This test reproduces the cache consistency issue that occurs during commit
	// when the underlying stores are reloaded but cache retains stale data

	tempDir := t.TempDir()
	logger := log.NewNopLogger()

	scConfig := config.StateCommitConfig{
		Enable:    true,
		Directory: tempDir,
		CacheSize: 100,
	}
	ssConfig := config.StateStoreConfig{Enable: false}

	store := NewStore(tempDir, logger, scConfig, ssConfig, false)

	// Set up inter-block cache BEFORE loading stores - this is crucial
	cacheManager := cache.NewCommitKVStoreCacheManager(
		cache.DefaultCommitKVStoreCacheSize,
		types.DefaultCacheSizeLimit,
	)
	store.SetInterBlockCache(cacheManager)

	// Mount and load store
	storeKey := types.NewKVStoreKey("test")
	store.MountStoreWithDB(storeKey, types.StoreTypeIAVL, nil)
	err := store.LoadLatestVersion()
	require.NoError(t, err)

	// Get the store and check cache manager state
	kvStore := store.GetCommitKVStore(storeKey)
	t.Logf("Store type after loading: %T, %p", kvStore, kvStore)

	// Check cache manager state
	unwrapped := cacheManager.Unwrap(storeKey)
	t.Logf("Cache manager unwrap result: %v, %p", unwrapped != nil, unwrapped)

	// Force cache creation by calling GetStoreCache directly
	cachedStore := cacheManager.GetStoreCache(storeKey, kvStore)
	t.Logf("Cached store type: %T", cachedStore)

	// Check if store is now cached after explicit cache creation
	newUnwrapped := cacheManager.Unwrap(storeKey)
	t.Logf("Cache manager unwrap after explicit cache: %v, %p", newUnwrapped != nil, newUnwrapped)

	// Write and read data using the explicitly cached store
	testKey := []byte("test-key")
	testValue := []byte("test-value")

	// Use CacheMultiStore to write data (proper storev2 pattern)
	cms := store.CacheMultiStore()
	cmsStore := cms.GetKVStore(storeKey)
	cmsStore.Set(testKey, testValue)
	cms.Write() // Write to underlying stores

	// Now commit - this triggers store reload and potential cache mismatch
	// The cache has testValue, but after reload, there might be a mismatch
	commitID := store.Commit(true)
	require.NotEqual(t, types.CommitID{}, commitID)

	// Read from the cached store to populate inter-block cache
	readValue := cachedStore.Get(testKey)
	require.Equal(t, testValue, readValue, "Should read written value from cached store")

	// After commit, get the store again and check types
	newKvStore := store.GetCommitKVStore(storeKey)
	t.Logf("Store type after commit: %T, %p", newKvStore, newKvStore)

	// Check if the cache still has the store
	postCommitUnwrapped := cacheManager.Unwrap(storeKey)
	t.Logf("Cache manager unwrap after commit: %v", postCommitUnwrapped != nil)

	// This read should work without cache consistency errors
	// If there's a cache mismatch, we might see error logs or incorrect values
	postCommitValue := newKvStore.Get(testKey)
	require.Equal(t, testValue, postCommitValue, "Should read same value after commit without cache errors")

	// Also test reading from the cached store directly
	cachedPostCommitValue := cachedStore.Get(testKey)
	t.Logf("Cached store read result: %v", cachedPostCommitValue != nil)

	// If we got here without errors, test what we can
	t.Log("Basic cache functionality test completed")
}
