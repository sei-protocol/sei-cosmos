package store

import (
	"github.com/cosmos/cosmos-sdk/store/types"
)

// Import cosmos-sdk/types/store.go for convenience.
type (
	PruningOptions   = types.PruningOptions
	Store            = types.Store
	Committer        = types.Committer
	CommitStore      = types.CommitStore
	MultiStore       = types.MultiStore
	CacheMultiStore  = types.CacheMultiStore
	CommitMultiStore = types.CommitMultiStore
	ConcurrentCacheMultiStore = types.ConcurrentCacheMultiStore
	KVStore          = types.KVStore
	KVPair           = types.KVPair
	Iterator         = types.Iterator
	CacheKVStore     = types.CacheKVStore
	ConcurrentCacheKV = types.ConcurrentCacheKVStore
	CommitKVStore    = types.CommitKVStore
	CacheWrapper     = types.CacheWrapper
	CacheWrap        = types.CacheWrap
	CommitID         = types.CommitID
	Key              = types.StoreKey
	Type             = types.StoreType
	Queryable        = types.Queryable
	TraceContext     = types.TraceContext
	Gas              = types.Gas
	GasMeter         = types.GasMeter
	GasConfig        = types.GasConfig
)
