package slashing

import (
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

type SlashingWriteInfo struct {
	ConsAddr    sdk.ConsAddress
	Index       int64
	Previous    bool
	Missed      bool
	SigningInfo types.ValidatorSigningInfo
	ShouldSlash bool
	SlashInfo   keeper.SlashInfo
}

// BeginBlocker check for infraction evidence or downtime of validators
// on every begin block
func BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	startTime := time.Now().UnixMicro()
	var wg sync.WaitGroup
	// Iterate over all the validators which *should* have signed this block
	// store whether or not they have actually signed it and slash/unbond any
	// which have missed too many blocks in a row (downtime slashing)

	// this allows us to preserve the original ordering for writing purposes
	slashingWriteInfo := make([]*SlashingWriteInfo, len(req.LastCommitInfo.GetVotes()))
	latencyInfo := make([]int64, len(req.LastCommitInfo.GetVotes()))

	for i, voteInfo := range req.LastCommitInfo.GetVotes() {
		wg.Add(1)
		go func(valIndex int, vInfo abci.VoteInfo) {
			defer wg.Done()
			startTime = time.Now().UnixMicro()
			consAddr, index, previous, missed, signInfo, shouldSlash, slashInfo := k.HandleValidatorSignatureConcurrent(ctx, vInfo.Validator.Address, vInfo.Validator.Power, vInfo.SignedLastBlock)
			slashingWriteInfo[valIndex] = &SlashingWriteInfo{
				ConsAddr:    consAddr,
				Index:       index,
				Previous:    previous,
				Missed:      missed,
				SigningInfo: signInfo,
				ShouldSlash: shouldSlash,
				SlashInfo:   slashInfo,
			}
			latencyInfo[valIndex] = time.Now().UnixMicro() - startTime
			// TODO: panic handling?
		}(i, voteInfo)
	}
	wg.Wait()
	endWaitGroupTime := time.Now().UnixMicro()
	ctx.Logger().Info(fmt.Sprintf("[Cosmos-Debug] Slahsing WaitGroup total latency: %d, per goroutine latency: %v", endWaitGroupTime-startTime, latencyInfo))
	ctx.Logger().Info(fmt.Sprintf("[Cosmos-Debug] Slashing Breakdown TotalGetPubkey latency: %d, TotalGetValidatorSigningInfo latency: %d, TotalSignedBlocksWindow latency: %d, TotalGetValidatorMissedBlockBitArray latency: %d, TotalMinSignedPerWindow latency: %d, TotalCheckPunish latency: %d",
		keeper.TotalGetPubkey.Load(), keeper.TotalGetValidatorSigningInfo.Load(), keeper.TotalSignedBlocksWindow.Load(), keeper.TotalGetValidatorMissedBlockBitArray.Load(), keeper.TotalMinSignedPerWindow.Load(), keeper.TotalCheckPunish.Load()))

	keeper.TotalGetPubkey.Store(0)
	keeper.TotalGetValidatorSigningInfo.Store(0)
	keeper.TotalSignedBlocksWindow.Store(0)
	keeper.TotalGetValidatorMissedBlockBitArray.Store(0)
	keeper.TotalMinSignedPerWindow.Store(0)
	keeper.TotalCheckPunish.Store(0)

	for _, writeInfo := range slashingWriteInfo {
		if writeInfo == nil {
			panic("Expected slashing write info to be non-nil")
		}
		// Update the validator missed block bit array by index if different from last value at the index
		switch {
		case writeInfo.ShouldSlash:
			// this differs from the original switch, since we know that we are going to be slashing + jailing the validator, we can proactively just clear their bit array instead of updating it and THEN clearing it
			k.ClearValidatorMissedBlockBitArray(ctx, writeInfo.ConsAddr)
		case !writeInfo.Previous && writeInfo.Missed:
			k.SetValidatorMissedBlockBitArray(ctx, writeInfo.ConsAddr, writeInfo.Index, true)
		case writeInfo.Previous && !writeInfo.Missed:
			k.SetValidatorMissedBlockBitArray(ctx, writeInfo.ConsAddr, writeInfo.Index, false)
		default:
			// noop
		}
		if writeInfo.ShouldSlash {
			writeInfo.SigningInfo = k.SlashJailAndUpdateSigningInfo(ctx, writeInfo.ConsAddr, writeInfo.SlashInfo, writeInfo.SigningInfo)
		}
		k.SetValidatorSigningInfo(ctx, writeInfo.ConsAddr, writeInfo.SigningInfo)
	}

	endTime := time.Now().UnixMicro()
	ctx.Logger().Info(fmt.Sprintf("[Cosmos-Debug] Slashing total latency: %d", endTime-startTime))
}
