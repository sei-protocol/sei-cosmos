package slashing

import (
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

	var wg sync.WaitGroup
	// Iterate over all the validators which *should* have signed this block
	// store whether or not they have actually signed it and slash/unbond any
	// which have missed too many blocks in a row (downtime slashing)

	// this allows us to preserve the original ordering for writing purposes
	slashingWriteInfo := make([]*SlashingWriteInfo, len(req.LastCommitInfo.GetVotes()))

	for i, voteInfo := range req.LastCommitInfo.GetVotes() {
		wg.Add(1)
		go func(valIndex int, vInfo abci.VoteInfo) {
			defer wg.Done()
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
		}(i, voteInfo)
	}
	wg.Wait()

	for _, writeInfo := range slashingWriteInfo {
		if writeInfo == nil {
			panic("Expected slashing write info to be non-nil")
		}
		// Check if we need to resize the array if there was recently a change in slashing window size
		window := k.SignedBlocksWindow(ctx)
		missedInfo, found := k.GetValidatorMissedBlocks(ctx, writeInfo.ConsAddr)
		missedBlockLen := int64(len(missedInfo.MissedBlocks))
		if found && window != missedBlockLen {
			// we need to resize the missed block array AND update the signing info accordingly
			switch {
			case missedBlockLen < window:
				// missed block array too short, lets expand it
				newArray := make([]bool, window)
				copy(newArray, missedInfo.MissedBlocks)
				missedInfo.MissedBlocks = newArray
				k.SetValidatorMissedBlocks(ctx, writeInfo.ConsAddr, missedInfo)
			case missedBlockLen > window:
				// missed block array too long, we need to trim
				// we need to keep the last N blocks prior to the validator index offset (wrapping around backwards if necessary)
				indexOffset := writeInfo.SigningInfo.IndexOffset % window
				relativeIndexOffset := writeInfo.SigningInfo.IndexOffset % missedBlockLen
				newMissedBlocks := make([]bool, window)
				// start from relative index offset, go back window blocks (using mod with arr size for proper indexing)
				// save into a new array starting from index offset, going back (modding by window)
				// add missed block len so modulus doesnt go negative
				indexOffsetCounter := indexOffset + window
				for i := relativeIndexOffset + missedBlockLen; i > relativeIndexOffset+missedBlockLen-window; i-- {
					missedBlockIdx := i % missedBlockLen
					newMissedBlocks[indexOffsetCounter%window] = missedInfo.MissedBlocks[missedBlockIdx]
					indexOffsetCounter--
				}
				missedInfo.MissedBlocks = newMissedBlocks
				newMissedCount := 0
				for _, b := range missedInfo.MissedBlocks {
					if b {
						newMissedCount++
					}
				}
				writeInfo.SigningInfo.MissedBlocksCounter = int64(newMissedCount)
				k.SetValidatorMissedBlocks(ctx, writeInfo.ConsAddr, missedInfo)
			}

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
}
