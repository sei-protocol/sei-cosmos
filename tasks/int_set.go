package tasks

import (
	"sync"
	"sync/atomic"
)

type IntSet interface {
	Add(idx int)
	Delete(idx int)
	Length() int
	Exists(idx int) bool
}

// points to implementation
func newIntSet(size int) IntSet {
	return newIntSetSyncMap(size)
}

// syncSetMap uses a map with a RW Mutex
type intSetMap struct {
	mx sync.RWMutex
	m  map[int]struct{}
}

func newIntSetMap(size int) IntSet {
	return &intSetMap{
		m: make(map[int]struct{}),
	}
}

func (ss *intSetMap) Add(idx int) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	ss.m[idx] = struct{}{}
}

func (ss *intSetMap) Delete(idx int) {
	if ss.Exists(idx) {
		ss.mx.Lock()
		defer ss.mx.Unlock()
		delete(ss.m, idx)
	}
}

func (ss *intSetMap) Length() int {
	ss.mx.RLock()
	defer ss.mx.RUnlock()
	return len(ss.m)
}

func (ss *intSetMap) Exists(idx int) bool {
	ss.mx.RLock()
	defer ss.mx.RUnlock()
	_, ok := ss.m[idx]
	return ok
}

// intSetSyncMap uses a sync.Map with a length counter
type intSetSyncMap struct {
	m      sync.Map
	length int32
}

func newIntSetSyncMap(size int) IntSet {
	return &intSetSyncMap{}
}

func (ss *intSetSyncMap) Add(idx int) {
	_, loaded := ss.m.LoadOrStore(idx, struct{}{})
	if !loaded {
		atomic.AddInt32(&ss.length, 1)
	}
}

func (ss *intSetSyncMap) Delete(idx int) {
	_, ok := ss.m.Load(idx)
	if ok {
		ss.m.Delete(idx)
		atomic.AddInt32(&ss.length, -1)
	}
}

func (ss *intSetSyncMap) Length() int {
	return int(atomic.LoadInt32(&ss.length))
}

func (ss *intSetSyncMap) Exists(idx int) bool {
	_, ok := ss.m.Load(idx)
	return ok
}

// syncSet holds a set of integers in a thread-safe way.
type intSetByteSlice struct {
	locks  []sync.RWMutex
	state  []byte
	length int32
}

func newIntSetByteSlice(size int) *syncSet {
	return &syncSet{
		state: make([]byte, size),
		locks: make([]sync.RWMutex, size),
	}
}

func (ss *intSetByteSlice) Add(idx int) {
	// First check without locking to reduce contention.
	if ss.state[idx] == byte(0) {
		ss.locks[idx].Lock()
		// Check again to make sure it hasn't changed since acquiring the lock.
		if ss.state[idx] == byte(0) {
			ss.state[idx] = byte(1)
			atomic.AddInt32(&ss.length, 1)
		}
		ss.locks[idx].Unlock()
	}
}

func (ss *intSetByteSlice) Delete(idx int) {
	ss.locks[idx].Lock()
	defer ss.locks[idx].Unlock()

	// Check again to make sure it hasn't changed since acquiring the lock.
	if ss.state[idx] == byte(1) {
		ss.state[idx] = byte(0)
		atomic.AddInt32(&ss.length, -1)
	}

}

func (ss *intSetByteSlice) Length() int {
	return int(atomic.LoadInt32(&ss.length))
}

func (ss *intSetByteSlice) Exists(idx int) bool {
	// Atomic read of a single byte is safe
	return ss.state[idx] == byte(1)
}
