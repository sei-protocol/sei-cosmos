package multiversion

import (
	"sync"

	"github.com/cosmos/cosmos-sdk/store/types"
	"github.com/google/btree"
)

const (
	// The approximate number of items and children per B-tree node. Tuned with benchmarks.
	multiVersionBTreeDegree = 2 // should be equivalent to a binary search tree TODO: benchmark this
)

type multiVersionItem struct {
	valueTree *btree.BTree // contains versions values written to this key
	mtx       sync.RWMutex // manages read + write accesses
}

type valueItem struct {
	index int
	value []byte
}

// implement Less for btree.Item for valueItem
func (i valueItem) Less(other btree.Item) bool {
	return i.index < other.(valueItem).index
}

type MultiVersionValue interface {
	GetLatest() (value []byte, found bool)
	GetValueByIndex(index int) (value []byte, found bool)
	Set(index int, value []byte)
	Delete(index int)
}

var _ MultiVersionValue = (*multiVersionItem)(nil)

func NewMultiVersionItem() *multiVersionItem {
	return &multiVersionItem{
		valueTree: btree.New(multiVersionBTreeDegree),
	}
}

func NewValueItem(index int, value []byte) valueItem {
	return valueItem{
		index: index,
		value: value,
	}
}

func NewDeletedItem(index int) valueItem {
	return valueItem{
		index: index,
		value: nil,
	}
}

// GetLatest returns the latest written value to the btree, and returns a boolean indicating whether it was found.
//
// A `nil` value along with `found=true` indicates a deletion that has occurred and the underlying parent store doesn't need to be hit.
func (item *multiVersionItem) GetLatest() ([]byte, bool) {
	item.mtx.RLock()
	defer item.mtx.RUnlock()

	bTreeItem := item.valueTree.Max()
	if bTreeItem == nil {
		return nil, false
	}
	valueItem := bTreeItem.(valueItem)
	return valueItem.value, true
}

// GetLatest returns the latest written value to the btree prior to the index passed in, and returns a boolean indicating whether it was found.
//
// A `nil` value along with `found=true` indicates a deletion that has occurred and the underlying parent store doesn't need to be hit.
func (item *multiVersionItem) GetValueByIndex(index int) ([]byte, bool) {
	item.mtx.RLock()
	defer item.mtx.RUnlock()

	pivot := NewDeletedItem(index)

	var vItem valueItem
	var found bool
	// start from pivot which contains our current index, and return on first item we hit.
	// This will ensure we get the latest indexed value relative to our current index
	item.valueTree.DescendLessOrEqual(pivot, func(bTreeItem btree.Item) bool {
		vItem = bTreeItem.(valueItem)
		found = true
		return false
	})
	return vItem.value, found
}

func (item *multiVersionItem) Set(index int, value []byte) {
	types.AssertValidValue(value)
	item.mtx.Lock()
	defer item.mtx.Unlock()

	valueItem := NewValueItem(index, value)
	item.valueTree.ReplaceOrInsert(valueItem)
}

func (item *multiVersionItem) Delete(index int) {
	item.mtx.Lock()
	defer item.mtx.Unlock()

	deletedItem := NewDeletedItem(index)
	item.valueTree.ReplaceOrInsert(deletedItem)
}
