package v304

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/kv"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func GetDelegatorStartingInfoKey(v sdk.ValAddress, d sdk.AccAddress) []byte {
	return append(append(types.DelegatorStartingInfoPrefix, address.MustLengthPrefix(v.Bytes())...), address.MustLengthPrefix(d.Bytes())...)
}

// delete the starting info associated with a delegator
func DeleteDelegatorStartingInfo(ctx sdk.Context, val sdk.ValAddress, del sdk.AccAddress, storeKey sdk.StoreKey) {
	store := ctx.KVStore(storeKey)
	store.Delete(GetDelegatorStartingInfoKey(val, del))
}

// iterate over delegator starting infos
func IterateDelegatorStartingInfos(
	ctx sdk.Context,
	handler func(val sdk.ValAddress, del sdk.AccAddress, info types.DelegatorStartingInfo) (stop bool),
	storeKey sdk.StoreKey,
	cdc codec.BinaryCodec,
) {
	store := ctx.KVStore(storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.DelegatorStartingInfoPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var info types.DelegatorStartingInfo
		cdc.MustUnmarshal(iter.Value(), &info)
		val, del := GetDelegatorStartingInfoAddresses(iter.Key())
		if handler(val, del, info) {
			break
		}
	}
}

// GetDelegatorStartingInfoAddresses creates the addresses from a delegator starting info key.
func GetDelegatorStartingInfoAddresses(key []byte) (valAddr sdk.ValAddress, delAddr sdk.AccAddress) {
	// key is in the format:
	// 0x04<valAddrLen (1 Byte)><valAddr_Bytes><accAddrLen (1 Byte)><accAddr_Bytes>
	kv.AssertKeyAtLeastLength(key, 2)
	valAddrLen := int(key[1])
	kv.AssertKeyAtLeastLength(key, 3+valAddrLen)
	valAddr = sdk.ValAddress(key[2 : 2+valAddrLen])
	delAddrLen := int(key[2+valAddrLen])
	kv.AssertKeyAtLeastLength(key, 4+valAddrLen)
	delAddr = sdk.AccAddress(key[3+valAddrLen:])
	kv.AssertKeyLength(delAddr.Bytes(), delAddrLen)

	return
}
