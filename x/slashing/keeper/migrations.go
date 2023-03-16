package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v043 "github.com/cosmos/cosmos-sdk/x/slashing/legacy/v043"

	"github.com/cosmos/cosmos-sdk/x/slashing/types"

	gogotypes "github.com/gogo/protobuf/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v043.MigrateStore(ctx, m.keeper.storeKey)
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	store := ctx.KVStore(m.keeper.storeKey)
	valMissedMap := make(map[string]types.ValidatorMissedBlockArray)
	iter := sdk.KVStorePrefixIterator(store, types.ValidatorMissedBlockBitArrayKeyPrefix)

	signingWindow := m.keeper.SignedBlocksWindow(ctx)

	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		// need to use the key to extract validator cons addr
		// last 8 bytes are the index
		// remove the store prefix + length prefix
		key := iter.Key()
		consAddrBytes, indexBytes := key[2:len(key)-8], key[len(key)-8:]

		consAddr := sdk.ConsAddress(consAddrBytes)
		index := int64(binary.LittleEndian.Uint64(indexBytes))

		arr, ok := valMissedMap[consAddr.String()]
		if !ok {
			arr = types.ValidatorMissedBlockArray{
				Address:      consAddr.String(),
				MissedBlocks: make([]bool, signingWindow),
			}
		}
		var missed gogotypes.BoolValue
		m.keeper.cdc.MustUnmarshal(iter.Value(), &missed)
		arr.MissedBlocks[index] = missed.Value
		valMissedMap[consAddr.String()] = arr
		store.Delete(iter.Key())
	}

	for key, missedBlockArray := range valMissedMap {
		consAddrKey, err := sdk.ConsAddressFromBech32(key)
		if err != nil {
			return err
		}
		m.keeper.SetValidatorMissedBlocks(ctx, consAddrKey, missedBlockArray)
	}
	return nil
}
