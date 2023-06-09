package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v043 "github.com/cosmos/cosmos-sdk/x/distribution/legacy/v043"
	v304 "github.com/cosmos/cosmos-sdk/x/distribution/legacy/v304"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
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

func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	type infoTuple struct {
		val  sdk.ValAddress
		del  sdk.AccAddress
		info types.DelegatorStartingInfo
	}
	infoList := []infoTuple{}
	cnt := 0
	v304.IterateDelegatorStartingInfos(ctx, func(val sdk.ValAddress, del sdk.AccAddress, info types.DelegatorStartingInfo) (stop bool) {
		infoList = append(infoList, infoTuple{val: val, del: del, info: info})
		cnt++
		if cnt%10000 == 0 {
			fmt.Printf("Iterated %d info\n", cnt)
		}
		return false
	}, m.keeper.storeKey, m.keeper.cdc)

	cnt = 0
	for _, info := range infoList {
		v304.DeleteDelegatorStartingInfo(ctx, info.val, info.del, m.keeper.storeKey)
		m.keeper.SetDelegatorStartingInfo(ctx, info.val, info.del, info.info)
		cnt++
		if cnt%10000 == 0 {
			fmt.Printf("Processed %d info\n", cnt)
		}
	}
	return nil
}
