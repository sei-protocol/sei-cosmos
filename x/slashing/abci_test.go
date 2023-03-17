package slashing_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/slashing/testslashing"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/teststaking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestBeginBlocker(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	pks := simapp.CreateTestPubKeys(5)
	simapp.AddTestAddrsFromPubKeys(app, ctx, pks, app.StakingKeeper.TokensFromConsensusPower(ctx, 200))
	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	trueVotes := []abci.VoteInfo{}
	falseVotes := []abci.VoteInfo{}

	for i := 0; i < 5; i++ {
		addr, pk := sdk.ValAddress(pks[i].Address()), pks[i]
		// bond the validator
		power := int64(100)
		amt := tstaking.CreateValidatorWithValPower(addr, pk, power, true)
		staking.EndBlocker(ctx, app.StakingKeeper)
		require.Equal(
			t, app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr)),
			sdk.NewCoins(sdk.NewCoin(app.StakingKeeper.GetParams(ctx).BondDenom, InitTokens.Sub(amt))),
		)
		require.Equal(t, amt, app.StakingKeeper.Validator(ctx, addr).GetBondedTokens())
		val := abci.Validator{
			Address: pk.Address(),
			Power:   power,
		}
		trueVotes = append(trueVotes, abci.VoteInfo{
			Validator:       val,
			SignedLastBlock: true,
		})
		falseVotes = append(falseVotes, abci.VoteInfo{
			Validator:       val,
			SignedLastBlock: i != 0,
		})
	}

	params := app.SlashingKeeper.GetParams(ctx)
	params.SignedBlocksWindow = 10000
	params.MinSignedPerWindow = sdk.MustNewDecFromStr("0.5")
	app.SlashingKeeper.SetParams(ctx, params)

	// mark the validator as having signed
	req := abci.RequestBeginBlock{
		LastCommitInfo: abci.LastCommitInfo{
			Votes: trueVotes,
		},
	}
	slashing.BeginBlocker(ctx, req, app.SlashingKeeper)

	for i := 0; i < 5; i++ {
		info, found := app.SlashingKeeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(pks[i].Address()))
		require.True(t, found)
		require.Equal(t, ctx.BlockHeight(), info.StartHeight)
		require.Equal(t, int64(1), info.IndexOffset)
		require.Equal(t, time.Unix(0, 0).UTC(), info.JailedUntil)
		require.Equal(t, int64(0), info.MissedBlocksCounter)
	}

	height := int64(0)

	// for 10000 blocks, mark the validator as having signed
	for ; height < app.SlashingKeeper.SignedBlocksWindow(ctx); height++ {
		ctx = ctx.WithBlockHeight(height)
		req = abci.RequestBeginBlock{
			LastCommitInfo: abci.LastCommitInfo{
				Votes: trueVotes,
			},
		}

		slashing.BeginBlocker(ctx, req, app.SlashingKeeper)
	}

	// for 5000 blocks, mark the validator as having not signed
	for ; height < ((app.SlashingKeeper.SignedBlocksWindow(ctx) * 2) - app.SlashingKeeper.MinSignedPerWindow(ctx) + 1); height++ {
		ctx = ctx.WithBlockHeight(height)
		req = abci.RequestBeginBlock{
			LastCommitInfo: abci.LastCommitInfo{
				Votes: falseVotes,
			},
		}

		slashing.BeginBlocker(ctx, req, app.SlashingKeeper)
	}

	// end block
	staking.EndBlocker(ctx, app.StakingKeeper)

	// validator should be jailed
	for i := 0; i < 5; i++ {
		validator, found := app.StakingKeeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(pks[i]))
		require.True(t, found)
		if i == 0 {
			require.Equal(t, stakingtypes.Unbonding, validator.GetStatus())
		} else {
			require.Equal(t, stakingtypes.Bonded, validator.GetStatus())
		}
	}
}

func TestResizeTrimValidatorMissedBlocksArray(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addrDels := simapp.AddTestAddrsIncremental(app, ctx, 1, app.StakingKeeper.TokensFromConsensusPower(ctx, 200))
	valAddrs := simapp.ConvertAddrsToValAddrs(addrDels)

	pks := simapp.CreateTestPubKeys(1)

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	addr, val := valAddrs[0], pks[0]
	tstaking.CreateValidatorWithValPower(addr, val, 200, true)

	staking.EndBlocker(ctx, app.StakingKeeper)

	consAddr := sdk.GetConsAddress(val)

	newInfo := types.NewValidatorSigningInfo(
		consAddr,
		int64(4),
		int64(17), // will get bumped to 18 in beginblock
		time.Unix(2, 0),
		false,
		int64(3),
	)
	app.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)

	tooLargeArray := types.ValidatorMissedBlockArray{
		Address:      consAddr.String(),
		MissedBlocks: []bool{false, false, false, false, false, true, false, true, false, true},
	}
	app.SlashingKeeper.SetValidatorMissedBlocks(ctx, consAddr, tooLargeArray)

	params := app.SlashingKeeper.GetParams(ctx)
	params.SignedBlocksWindow = 8
	app.SlashingKeeper.SetParams(ctx, params)

	slashing.BeginBlocker(ctx, testslashing.CreateBeginBlockReq(val.Address(), 200, true), app.SlashingKeeper)

	missedInfo, found := app.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, 8, len(missedInfo.MissedBlocks))
	require.Equal(t, []bool{false, true, false, false, false, false, false, true}, missedInfo.MissedBlocks)

	info, found := app.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, int64(2), info.MissedBlocksCounter)

}

func TestResizeTrimWraparoundValidatorMissedBlocksArray(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addrDels := simapp.AddTestAddrsIncremental(app, ctx, 1, app.StakingKeeper.TokensFromConsensusPower(ctx, 200))
	valAddrs := simapp.ConvertAddrsToValAddrs(addrDels)

	pks := simapp.CreateTestPubKeys(1)

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	addr, val := valAddrs[0], pks[0]
	tstaking.CreateValidatorWithValPower(addr, val, 200, true)

	staking.EndBlocker(ctx, app.StakingKeeper)

	params := app.SlashingKeeper.GetParams(ctx)
	params.SignedBlocksWindow = 8
	app.SlashingKeeper.SetParams(ctx, params)

	consAddr := sdk.GetConsAddress(val)

	newInfo := types.NewValidatorSigningInfo(
		consAddr,
		int64(4),
		int64(32), // will get bumped to 33 in beginblock
		time.Unix(2, 0),
		false,
		int64(2),
	)
	app.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)

	tooLargeArray := types.ValidatorMissedBlockArray{
		Address:      consAddr.String(),
		MissedBlocks: []bool{false, false, false, false, false, true, false, true, false, false},
	}
	app.SlashingKeeper.SetValidatorMissedBlocks(ctx, consAddr, tooLargeArray)

	slashing.BeginBlocker(ctx, testslashing.CreateBeginBlockReq(val.Address(), 200, true), app.SlashingKeeper)

	missedInfo, found := app.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, 8, len(missedInfo.MissedBlocks))
	require.Equal(t, []bool{false, false, false, true, false, false, false, false}, missedInfo.MissedBlocks)

	info, found := app.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, int64(1), info.MissedBlocksCounter)

}

func TestResizeExpandValidatorMissedBlocksArray(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addrDels := simapp.AddTestAddrsIncremental(app, ctx, 1, app.StakingKeeper.TokensFromConsensusPower(ctx, 200))
	valAddrs := simapp.ConvertAddrsToValAddrs(addrDels)

	pks := simapp.CreateTestPubKeys(1)

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	addr, val := valAddrs[0], pks[0]
	tstaking.CreateValidatorWithValPower(addr, val, 200, true)

	staking.EndBlocker(ctx, app.StakingKeeper)

	params := app.SlashingKeeper.GetParams(ctx)
	params.SignedBlocksWindow = 8
	app.SlashingKeeper.SetParams(ctx, params)

	consAddr := sdk.GetConsAddress(val)

	newInfo := types.NewValidatorSigningInfo(
		consAddr,
		int64(4),
		int64(38), // bumped to 39 the next block
		time.Unix(2, 0),
		false,
		int64(1),
	)
	app.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)

	tooLargeArray := types.ValidatorMissedBlockArray{
		Address:      consAddr.String(),
		MissedBlocks: []bool{false, true, false, false},
	}
	app.SlashingKeeper.SetValidatorMissedBlocks(ctx, consAddr, tooLargeArray)

	slashing.BeginBlocker(ctx, testslashing.CreateBeginBlockReq(val.Address(), 200, true), app.SlashingKeeper)

	missedInfo, found := app.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, 8, len(missedInfo.MissedBlocks))
	require.Equal(t, []bool{false, false, false, false, false, true, false, false}, missedInfo.MissedBlocks)

	info, found := app.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, int64(1), info.MissedBlocksCounter)

}
