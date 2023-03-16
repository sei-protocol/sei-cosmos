package keeper_test

import (
	"encoding/binary"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/teststaking"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestMigrate2to3(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addrDels := simapp.AddTestAddrsIncremental(app, ctx, 2, app.StakingKeeper.TokensFromConsensusPower(ctx, 200))
	valAddrs := simapp.ConvertAddrsToValAddrs(addrDels)
	pks := simapp.CreateTestPubKeys(2)
	addr, val := valAddrs[0], pks[0]
	addr2, val2 := valAddrs[1], pks[1]
	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	params := app.SlashingKeeper.GetParams(ctx)
	params.SignedBlocksWindow = 10
	app.SlashingKeeper.SetParams(ctx, params)

	ctx = ctx.WithBlockHeight(app.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

	// Validator created
	amt := tstaking.CreateValidatorWithValPower(addr, val, 100, true)
	amt2 := tstaking.CreateValidatorWithValPower(addr2, val2, 100, true)

	staking.EndBlocker(ctx, app.StakingKeeper)
	require.Equal(
		t, app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr)),
		sdk.NewCoins(sdk.NewCoin(app.StakingKeeper.GetParams(ctx).BondDenom, InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, app.StakingKeeper.Validator(ctx, addr).GetBondedTokens())
	require.Equal(
		t, app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr2)),
		sdk.NewCoins(sdk.NewCoin(app.StakingKeeper.GetParams(ctx).BondDenom, InitTokens.Sub(amt))),
	)
	require.Equal(t, amt2, app.StakingKeeper.Validator(ctx, addr2).GetBondedTokens())

	consAddr := sdk.GetConsAddress(val)
	consAddr2 := sdk.GetConsAddress(val2)

	_, found := app.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr)
	require.False(t, found)
	_, found2 := app.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr2)
	require.False(t, found2)

	store := ctx.KVStore(app.SlashingKeeper.GetStoreKey())
	for i := 0; i < 5; i++ {
		keyPrefix := types.ValidatorMissedBlockBitArrayKey(consAddr)
		index := make([]byte, 8)
		binary.LittleEndian.PutUint64(index, uint64(i))
		key := append(keyPrefix, index...)
		bz := app.AppCodec().MustMarshal(&gogotypes.BoolValue{Value: true})
		store.Set(key, bz)

		keyPrefix2 := types.ValidatorMissedBlockBitArrayKey(consAddr2)
		index2 := make([]byte, 8)
		binary.LittleEndian.PutUint64(index2, uint64(i+1))
		key2 := append(keyPrefix2, index2...)
		bz2 := app.AppCodec().MustMarshal(&gogotypes.BoolValue{Value: true})
		store.Set(key2, bz2)
	}

	m := keeper.NewMigrator(app.SlashingKeeper)
	err := m.Migrate2to3(ctx)
	require.NoError(t, err)

	missedArray, found := app.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, app.SlashingKeeper.SignedBlocksWindow(ctx), int64(len(missedArray.MissedBlocks)))

	for i, val := range missedArray.MissedBlocks {
		if i < 5 {
			require.True(t, val)
		} else {
			require.False(t, val)
		}
	}

	missedArray2, found2 := app.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr2)
	require.True(t, found2)
	require.Equal(t, app.SlashingKeeper.SignedBlocksWindow(ctx), int64(len(missedArray2.MissedBlocks)))

	for i, val := range missedArray2.MissedBlocks {
		if i > 0 && i < 6 {
			require.True(t, val)
		} else {
			require.False(t, val)
		}
	}
}
