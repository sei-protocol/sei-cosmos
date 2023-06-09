package keeper

import (
	"encoding/binary"

	"github.com/gogo/protobuf/grpc"

	v043 "github.com/cosmos/cosmos-sdk/x/auth/legacy/v043"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper      AccountKeeper
	queryServer grpc.Server
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper AccountKeeper, queryServer grpc.Server) Migrator {
	return Migrator{keeper: keeper, queryServer: queryServer}
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	var iterErr error

	m.keeper.IterateAccounts(ctx, func(account types.AccountI) (stop bool) {
		wb, err := v043.MigrateAccount(ctx, account, m.queryServer)
		if err != nil {
			iterErr = err
			return true
		}

		if wb == nil {
			return false
		}

		m.keeper.SetAccount(ctx, wb)
		return false
	})

	return iterErr
}

func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	store := ctx.KVStore(m.keeper.key)
	m.keeper.IterateAccounts(ctx, func(account types.AccountI) (stop bool) {
		accNumBz := make([]byte, 8)
		binary.BigEndian.PutUint64(accNumBz, account.GetAccountNumber())
		store.Set(types.AddressByIDKey(accNumBz), account.GetAddress())
		return false
	})
	return nil
}
