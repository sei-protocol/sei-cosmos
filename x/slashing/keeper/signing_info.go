package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

// GetValidatorSigningInfo retruns the ValidatorSigningInfo for a specific validator
// ConsAddress
func (k Keeper) GetValidatorSigningInfo(ctx sdk.Context, address sdk.ConsAddress) (info types.ValidatorSigningInfo, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ValidatorSigningInfoKey(address))
	if bz == nil {
		found = false
		return
	}
	k.cdc.MustUnmarshal(bz, &info)
	found = true
	return
}

// HasValidatorSigningInfo returns if a given validator has signing information
// persited.
func (k Keeper) HasValidatorSigningInfo(ctx sdk.Context, consAddr sdk.ConsAddress) bool {
	_, ok := k.GetValidatorSigningInfo(ctx, consAddr)
	return ok
}

// SetValidatorSigningInfo sets the validator signing info to a consensus address key
func (k Keeper) SetValidatorSigningInfo(ctx sdk.Context, address sdk.ConsAddress, info types.ValidatorSigningInfo) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&info)
	store.Set(types.ValidatorSigningInfoKey(address), bz)
}

// IterateValidatorSigningInfos iterates over the stored ValidatorSigningInfo
func (k Keeper) IterateValidatorSigningInfos(ctx sdk.Context,
	handler func(address sdk.ConsAddress, info types.ValidatorSigningInfo) (stop bool)) {

	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.ValidatorSigningInfoKeyPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		address := types.ValidatorSigningInfoAddress(iter.Key())
		var info types.ValidatorSigningInfo
		k.cdc.MustUnmarshal(iter.Value(), &info)
		if handler(address, info) {
			break
		}
	}
}

// GetValidatorMissedBlockArray gets the missed blocks array
func (k Keeper) GetValidatorMissedBlocks(ctx sdk.Context, address sdk.ConsAddress) (missedInfo types.ValidatorMissedBlockArray, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ValidatorMissedBlockBitArrayKey(address))
	if bz == nil {
		found = false
		return
	}
	k.cdc.MustUnmarshal(bz, &missedInfo)
	found = true
	return
}

// SetValidatorMissedBlockArray sets the missed blocks array
func (k Keeper) SetValidatorMissedBlocks(ctx sdk.Context, address sdk.ConsAddress, missedInfo types.ValidatorMissedBlockArray) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&missedInfo)
	store.Set(types.ValidatorMissedBlockBitArrayKey(address), bz)
}

// GetValidatorMissedBlockBitArray gets the bit for the missed blocks array
func (k Keeper) GetValidatorMissedBlockBitArray(ctx sdk.Context, address sdk.ConsAddress, index int64) bool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ValidatorMissedBlockBitArrayKey(address))
	var missedInfo types.ValidatorMissedBlockArray
	if bz == nil {
		// lazy: treat empty key as not missed
		return false
	}
	k.cdc.MustUnmarshal(bz, &missedInfo)
	if int64(len(missedInfo.MissedBlocks)-1) < index {
		// the array isn't large enough to include that index yet, assume its false
		return false
	}
	missedIndex := missedInfo.MissedBlocks[index]
	return missedIndex
}

// JailUntil attempts to set a validator's JailedUntil attribute in its signing
// info. It will panic if the signing info does not exist for the validator.
func (k Keeper) JailUntil(ctx sdk.Context, consAddr sdk.ConsAddress, jailTime time.Time) {
	signInfo, ok := k.GetValidatorSigningInfo(ctx, consAddr)
	if !ok {
		panic("cannot jail validator that does not have any signing information")
	}

	signInfo.JailedUntil = jailTime
	k.SetValidatorSigningInfo(ctx, consAddr, signInfo)
}

// Tombstone attempts to tombstone a validator. It will panic if signing info for
// the given validator does not exist.
func (k Keeper) Tombstone(ctx sdk.Context, consAddr sdk.ConsAddress) {
	signInfo, ok := k.GetValidatorSigningInfo(ctx, consAddr)
	if !ok {
		panic("cannot tombstone validator that does not have any signing information")
	}

	if signInfo.Tombstoned {
		panic("cannot tombstone validator that is already tombstoned")
	}

	signInfo.Tombstoned = true
	k.SetValidatorSigningInfo(ctx, consAddr, signInfo)
}

// IsTombstoned returns if a given validator by consensus address is tombstoned.
func (k Keeper) IsTombstoned(ctx sdk.Context, consAddr sdk.ConsAddress) bool {
	signInfo, ok := k.GetValidatorSigningInfo(ctx, consAddr)
	if !ok {
		return false
	}

	return signInfo.Tombstoned
}

// SetValidatorMissedBlockBitArray sets the bit that checks if the validator has
// missed a block in the current window
func (k Keeper) SetValidatorMissedBlockBitArray(ctx sdk.Context, address sdk.ConsAddress, index int64, missed bool) {
	store := ctx.KVStore(k.storeKey)
	// get signed blocks window
	window := k.SignedBlocksWindow(ctx)
	// get info
	missedInfo, found := k.GetValidatorMissedBlocks(ctx, address)
	if !found {
		missedInfo = types.ValidatorMissedBlockArray{
			Address: address.String(),
		}
	}
	switch {
	case int64(len(missedInfo.MissedBlocks)) < window:
		// missed block array too short, lets expand it
		newArray := make([]bool, window)
		copy(newArray, missedInfo.MissedBlocks)
		missedInfo.MissedBlocks = newArray
	case int64(len(missedInfo.MissedBlocks)) > window:
		// missed block array too long, we need to trim
		// TODO: fix this
		missedInfo.MissedBlocks = missedInfo.MissedBlocks[0:window]
	}

	missedInfo.MissedBlocks[index] = missed

	bz := k.cdc.MustMarshal(&missedInfo)
	store.Set(types.ValidatorMissedBlockBitArrayKey(address), bz)
}

// clearValidatorMissedBlockBitArray deletes every instance of ValidatorMissedBlockBitArray in the store
func (k Keeper) ClearValidatorMissedBlockBitArray(ctx sdk.Context, address sdk.ConsAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ValidatorMissedBlockBitArrayKey(address))
}
