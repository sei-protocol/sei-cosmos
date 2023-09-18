package multiversion_test

import (
	"testing"

	mv "github.com/cosmos/cosmos-sdk/store/multiversion"
	"github.com/stretchr/testify/require"
)

func TestMultiversionItemGetLatest(t *testing.T) {
	mvItem := mv.NewMultiVersionItem()
	// We have no value, should get found == false and a nil value
	value, found := mvItem.GetLatest()
	require.False(t, found)
	require.Nil(t, value)

	// assert that we find a value after it's set
	one := []byte("one")
	mvItem.Set(1, one)
	value, found = mvItem.GetLatest()
	require.True(t, found)
	require.Equal(t, one, value)

	// assert that we STILL get the "one" value since it is the latest
	zero := []byte("zero")
	mvItem.Set(0, zero)
	value, found = mvItem.GetLatest()
	require.True(t, found)
	require.Equal(t, one, value)

	// we should see a deletion as the latest now, aka nil value and found == true
	mvItem.Delete(2)
	value, found = mvItem.GetLatest()
	require.True(t, found)
	require.Nil(t, value)

	// Overwrite the deleted value with some data
	two := []byte("two")
	mvItem.Set(2, two)
	value, found = mvItem.GetLatest()
	require.True(t, found)
	require.Equal(t, two, value)
}

func TestMultiversionItemGetByIndex(t *testing.T) {
	mvItem := mv.NewMultiVersionItem()
	// We have no value, should get found == false and a nil value
	value, found := mvItem.GetValueByIndex(9)
	require.False(t, found)
	require.Nil(t, value)

	// assert that we find a value after it's set
	one := []byte("one")
	mvItem.Set(1, one)
	value, found = mvItem.GetValueByIndex(1)
	require.True(t, found)
	require.Equal(t, one, value)

	// verify that querying for a later index properly returns the `one`
	value, found = mvItem.GetValueByIndex(3)
	require.True(t, found)
	require.Equal(t, one, value)

	// verify that querying for an earlier index returns nil
	value, found = mvItem.GetValueByIndex(0)
	require.False(t, found)
	require.Nil(t, value)

	// assert that we STILL get the "one" value when querying with a later index
	zero := []byte("zero")
	mvItem.Set(0, zero)
	value, found = mvItem.GetValueByIndex(2)
	require.True(t, found)
	require.Equal(t, one, value)
	// verify we get zero when querying with an earlier index
	value, found = mvItem.GetValueByIndex(0)
	require.True(t, found)
	require.Equal(t, zero, value)

	// we should see a deletion as the latest now, aka nil value and found == true
	mvItem.Delete(4)
	value, found = mvItem.GetValueByIndex(4)
	require.True(t, found)
	require.Nil(t, value)
	// should get deletion item for a later index as well
	value, found = mvItem.GetValueByIndex(5)
	require.True(t, found)
	require.Nil(t, value)
	// verify that we still read the proper underlying item for an older index
	value, found = mvItem.GetValueByIndex(3)
	require.True(t, found)
	require.Equal(t, one, value)

	// Overwrite the deleted value with some data and verify we read it properly
	four := []byte("four")
	mvItem.Set(4, four)
	value, found = mvItem.GetValueByIndex(4)
	require.True(t, found)
	require.Equal(t, four, value)
	// also reads the four
	value, found = mvItem.GetValueByIndex(6)
	require.True(t, found)
	require.Equal(t, four, value)
	// still reads the `one`
	value, found = mvItem.GetValueByIndex(3)
	require.True(t, found)
	require.Equal(t, one, value)
}
