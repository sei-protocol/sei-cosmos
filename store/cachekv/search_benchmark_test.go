package cachekv

import (
	"strconv"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/store/types"
	db "github.com/tendermint/tm-db"
)

func BenchmarkLargeUnsortedMisses(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store := generateStore()
		b.StartTimer()

		for k := 0; k < 10000; k++ {
			// cache has A + Z values
			// these are within range, but match nothing
			store.dirtyItems([]byte("B1"), []byte("B2"))
		}
	}
}

func generateStore() *Store {
	cache := &sync.Map{}
	unsorted := &sync.Map{}
	for i := 0; i < 5000; i++ {
		key := "A" + strconv.Itoa(i)
		unsorted.Store(key, struct{}{})
		cache.Store(key, &types.CValue{})
	}

	for i := 0; i < 5000; i++ {
		key := "Z" + strconv.Itoa(i)
		unsorted.Store(key, struct{}{})
		cache.Store(key, &types.CValue{})
	}

	return &Store{
		cache:         cache,
		unsortedCache: unsorted,
		sortedCache:   db.NewMemDB(),
	}
}
