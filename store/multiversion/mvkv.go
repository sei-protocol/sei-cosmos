package multiversion

import (
	"io"
	"sync"

	"github.com/cosmos/cosmos-sdk/internal/conv"
	"github.com/cosmos/cosmos-sdk/store/types"
	scheduler "github.com/cosmos/cosmos-sdk/types/occ"
	dbm "github.com/tendermint/tm-db"
)

// TODO: when integrating, this store needs to be wrapped by a gaskv so that we can appropriately track ALL of the calls that get directed to the various KV layers. We can store this gas usage calculation in the overall tx execution level, and then use the gas calculation for the transactions that succeed validation

// should manage a parent store AND a multiversion store AND a cache kv that's used for generating read / write sets
// reads should waterfall from cache kv to multiversion store to parent store
// writes should only be applied to the cache kv, and later we'll update the multiversion store after tx execution

// Version Indexed Store wraps the multiversion store in a way that implements the KVStore interface, but also stores the index of the transaction, and so store actions are applied to the multiversion store using that index
type VersionIndexedStore struct {
	mtx sync.Mutex
	// used for tracking reads and writes for eventual validation + persistence into multi-version store
	readset  map[string][]byte // contains the key -> value mapping for all keys read from the store (not mvkv, underlying store)
	writeset map[string][]byte // contains the key -> value mapping for all keys written to the store

	// TODO: do we need this? - I think so? / maybe we just treat `nil` value in the writeset as a delete
	deleted *sync.Map
	// dirty keys that haven't been sorted yet for iteration
	dirtySet map[string]struct{}
	// used for iterators - populated at the time of iterator instantiation
	sortedCache *dbm.MemDB // always ascending sorted
	// parent stores (both multiversion and underlying parent store)
	multiVersionStore MultiVersionStore
	parent            types.KVStore
	// transaction metadata for versioned operations
	transactionIndex int
	incarnation      int
	// have abort channel here for aborting transactions
	abortChannel chan scheduler.Abort
}

var _ types.KVStore = (*VersionIndexedStore)(nil)

func NewVersionIndexedStore(parent types.KVStore, multiVersionStore MultiVersionStore, transactionIndex, incarnation int, abortChannel chan scheduler.Abort) *VersionIndexedStore {
	return &VersionIndexedStore{
		readset:           make(map[string][]byte),
		writeset:          make(map[string][]byte),
		deleted:           &sync.Map{},
		dirtySet:          make(map[string]struct{}),
		sortedCache:       dbm.NewMemDB(),
		parent:            parent,
		multiVersionStore: multiVersionStore,
		transactionIndex:  transactionIndex,
		incarnation:       incarnation,
		abortChannel:      abortChannel,
	}
}

// GetReadset returns the readset
func (store *VersionIndexedStore) GetReadset() map[string][]byte {
	return store.readset
}

// GetWriteset returns the writeset
func (store *VersionIndexedStore) GetWriteset() map[string][]byte {
	return store.writeset
}

// Get implements types.KVStore.
func (store *VersionIndexedStore) Get(key []byte) []byte {
	// first try to get from writeset cache, if cache miss, then try to get from multiversion store, if that misses, then get from parent store
	// if the key is in the cache, return it

	// don't have RW mutex because we have to update readset
	store.mtx.Lock()
	defer store.mtx.Unlock()

	types.AssertValidKey(key)
	strKey := conv.UnsafeBytesToStr(key)
	// first check the MVKV writeset, and return that value if present
	cacheValue, ok := store.writeset[strKey]
	if ok {
		// return the value from the cache, no need to update any readset stuff
		return cacheValue
	}
	// read the readset to see if the value exists - and return if applicable
	if readsetVal, ok := store.readset[strKey]; ok {
		return readsetVal
	}

	// if we didn't find it, then we want to check the multivalue store + add to readset if applicable
	mvsValue := store.multiVersionStore.GetLatestBeforeIndex(store.transactionIndex, key)
	if mvsValue != nil {
		// found something,
		if mvsValue.IsEstimate() {
			store.abortChannel <- scheduler.NewEstimateAbort(mvsValue.Index())
			// TODO: is it safe to return nil here?
			return nil
		} else {
			// This handles both detecting readset conflicts and updating readset if applicable
			return store.parseValueAndUpdateReadset(strKey, mvsValue)
		}
	}
	// if we didn't find it in the multiversion store, then we want to check the parent store + add to readset
	parentValue := store.parent.Get(key)
	store.readset[strKey] = parentValue
	return parentValue
}

// This functions handles reads with deleted items and values and verifies that the data is consistent to what we currently have in the readset (IF we have a readset value for that key)
func (store *VersionIndexedStore) parseValueAndUpdateReadset(strKey string, mvsValue MultiVersionValueItem) []byte {
	value := mvsValue.Value()
	if mvsValue.IsDeleted() {
		value = nil
	}
	store.readset[strKey] = value
	return value
}

// Delete implements types.KVStore.
func (v *VersionIndexedStore) Delete(key []byte) {
	// v.multiVersionStore.Delete(v.transactionIndex, key)
	panic("unimplemented")
}

// Has implements types.KVStore.
func (v *VersionIndexedStore) Has(key []byte) bool {
	panic("unimplemented")
}

// Set implements types.KVStore.
func (*VersionIndexedStore) Set(key []byte, value []byte) {
	panic("unimplemented")
}

// Iterator implements types.KVStore.
func (v *VersionIndexedStore) Iterator(start []byte, end []byte) dbm.Iterator {
	panic("unimplemented")
}

// ReverseIterator implements types.KVStore.
func (v *VersionIndexedStore) ReverseIterator(start []byte, end []byte) dbm.Iterator {
	panic("unimplemented")
}

// GetStoreType implements types.KVStore.
func (v *VersionIndexedStore) GetStoreType() types.StoreType {
	panic("unimplemented")
}

// CacheWrap implements types.KVStore.
func (*VersionIndexedStore) CacheWrap(storeKey types.StoreKey) types.CacheWrap {
	panic("CacheWrap not supported for version indexed store")
}

// CacheWrapWithListeners implements types.KVStore.
func (*VersionIndexedStore) CacheWrapWithListeners(storeKey types.StoreKey, listeners []types.WriteListener) types.CacheWrap {
	panic("CacheWrapWithListeners not supported for version indexed store")
}

// CacheWrapWithTrace implements types.KVStore.
func (*VersionIndexedStore) CacheWrapWithTrace(storeKey types.StoreKey, w io.Writer, tc types.TraceContext) types.CacheWrap {
	panic("CacheWrapWithTrace not supported for version indexed store")
}

// GetWorkingHash implements types.KVStore.
func (v *VersionIndexedStore) GetWorkingHash() ([]byte, error) {
	panic("should never attempt to get working hash from version indexed store")
}
