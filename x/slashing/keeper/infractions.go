package keeper

import (
	"fmt"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

type SlashInfo struct {
	height             int64
	power              int64
	distributionHeight int64
	minHeight          int64
	minSignedPerWindow int64
}

// This performs similar logic to the above HandleValidatorSignature, but only performs READs such that it can be performed in parallel for all validators.
// Instead of updating appropriate validator bit arrays / signing infos, this will return the pending values to be written in a consistent order
func (k Keeper) HandleValidatorSignatureConcurrent(ctx sdk.Context, addr cryptotypes.Address, power int64, signed bool) (consAddr sdk.ConsAddress, missedInfo types.ValidatorMissedBlockArray, signInfo types.ValidatorSigningInfo, shouldSlash bool, slashInfo SlashInfo) {
	logger := k.Logger(ctx)
	height := ctx.BlockHeight()

	// fetch the validator public key
	consAddr = sdk.ConsAddress(addr)
	if _, err := k.GetPubkey(ctx, addr); err != nil {
		panic(fmt.Sprintf("Validator consensus-address %s not found", consAddr))
	}

	// fetch signing info
	signInfo, found := k.GetValidatorSigningInfo(ctx, consAddr)
	if !found {
		panic(fmt.Sprintf("Expected signing info for validator %s but not found", consAddr))
	}

	window := k.SignedBlocksWindow(ctx)

	index := signInfo.IndexOffset % k.SignedBlocksWindow(ctx)

	missedInfo, found = k.GetValidatorMissedBlocks(ctx, consAddr)

	if !found {
		arrLen := window / 64
		if window%64 > 0 {
			arrLen += 1
		}
		missedInfo = types.ValidatorMissedBlockArray{
			Address:      consAddr.String(),
			WindowSize:   window,
			MissedBlocks: make([]uint64, arrLen),
		}
	}
	// TODO:  resize if necessary?
	if found && missedInfo.WindowSize != window {
		// we need to resize the missed block array AND update the signing info accordingly
		switch {
		case missedInfo.WindowSize < window:
			// missed block array too short, lets expand it
			boolArray := k.ParseBitGroupsToBoolArray(missedInfo.MissedBlocks, missedInfo.WindowSize)
			oldIndex := signInfo.IndexOffset % missedInfo.WindowSize
			newArray := make([]bool, window)
			copy(newArray[0:oldIndex+1], boolArray[0:oldIndex+1])
			if oldIndex+1 < missedInfo.WindowSize {
				copy(newArray[oldIndex+1+(window-missedInfo.WindowSize):], boolArray[oldIndex+1:])
			}
			// rotate array from oldIndex to newIndex
			shiftDiff := ((index - oldIndex) + window) % window
			finalBoolArr := make([]bool, window)
			copy(finalBoolArr, newArray[window-shiftDiff:])
			copy(finalBoolArr[shiftDiff:], newArray[:window-shiftDiff])
			missedInfo.MissedBlocks = k.ParseBoolArrayToBitGroups(finalBoolArr)
			missedInfo.WindowSize = window
			// TODO: set missed blocks later
			// k.SetValidatorMissedBlocks(ctx, writeInfo.ConsAddr, missedInfo)
		case missedInfo.WindowSize > window:
			// missed block array too long, we need to trim
			// we need to keep the last N blocks prior to the validator index offset (wrapping around backwards if necessary)
			relativeIndexOffset := signInfo.IndexOffset % missedInfo.WindowSize
			oldMissedBlocksBools := k.ParseBitGroupsToBoolArray(missedInfo.MissedBlocks, missedInfo.WindowSize)
			newMissedBlocks := make([]bool, window)
			// start from relative index offset, go back window blocks (using mod with arr size for proper indexing)
			// save into a new array starting from index offset, going back (modding by window)
			// add missed block len so modulus doesnt go negative
			indexOffsetCounter := index + window
			for i := relativeIndexOffset + missedInfo.WindowSize; i > relativeIndexOffset+missedInfo.WindowSize-window; i-- {
				missedBlockIdx := i % missedInfo.WindowSize
				newMissedBlocks[indexOffsetCounter%window] = oldMissedBlocksBools[missedBlockIdx]
				indexOffsetCounter--
			}
			missedInfo.MissedBlocks = k.ParseBoolArrayToBitGroups(newMissedBlocks)
			newMissedCount := 0
			for _, b := range newMissedBlocks {
				if b {
					newMissedCount++
				}
			}
			signInfo.MissedBlocksCounter = int64(newMissedCount)
			missedInfo.WindowSize = window
			// TODO: set missed blocks later
			// k.SetValidatorMissedBlocks(ctx, writeInfo.ConsAddr, missedInfo)
		}
	}
	// bump index offset after performing potential resizing
	signInfo.IndexOffset++
	previous := k.GetValidatorMissedBlockBitFromArray(missedInfo.MissedBlocks, index)
	missed := !signed
	switch {
	case !previous && missed:
		// Array value has changed from not missed to missed, increment counter
		signInfo.MissedBlocksCounter++
		missedInfo.MissedBlocks = k.SetValidatorMissedBlockBitForArray(missedInfo.MissedBlocks, index, true)
	case previous && !missed:
		// Array value has changed from missed to not missed, decrement counter
		signInfo.MissedBlocksCounter--
		missedInfo.MissedBlocks = k.SetValidatorMissedBlockBitForArray(missedInfo.MissedBlocks, index, false)
	default:
		// Array value at this index has not changed, no need to update counter
	}

	// cutoff := height - window
	// // go through and increment number evicted as necessary
	// numberEvicted := 0
	// for _, missedHeight := range missedInfo.MissedHeights {
	// 	if missedHeight <= cutoff {
	// 		numberEvicted += 1
	// 	} else {
	// 		break
	// 	}
	// }
	// // update Missed Heights by excluding heights outside of the window
	// missedInfo.MissedHeights = missedInfo.MissedHeights[numberEvicted:]
	// missed := !signed
	// if missed {
	// 	missedInfo.MissedHeights = append(missedInfo.MissedHeights, height-1)
	// }
	// signInfo.MissedBlocksCounter = int64(len(missedInfo.MissedHeights))

	minSignedPerWindow := k.MinSignedPerWindow(ctx)
	if missed {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeLiveness,
				sdk.NewAttribute(types.AttributeKeyAddress, consAddr.String()),
				sdk.NewAttribute(types.AttributeKeyMissedBlocks, fmt.Sprintf("%d", signInfo.MissedBlocksCounter)),
				sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", height)),
			),
		)

		logger.Debug(
			"absent validator",
			"height", height,
			"validator", consAddr.String(),
			"missed", signInfo.MissedBlocksCounter,
			"threshold", minSignedPerWindow,
		)
	}

	minHeight := signInfo.StartHeight + k.SignedBlocksWindow(ctx)
	maxMissed := window - minSignedPerWindow
	shouldSlash = false
	// if we are past the minimum height and the validator has missed too many blocks, punish them
	if height > minHeight && signInfo.MissedBlocksCounter > maxMissed {
		validator := k.sk.ValidatorByConsAddr(ctx, consAddr)
		if validator != nil && !validator.IsJailed() {
			// Downtime confirmed: slash and jail the validator
			// We need to retrieve the stake distribution which signed the block, so we subtract ValidatorUpdateDelay from the evidence height,
			// and subtract an additional 1 since this is the LastCommit.
			// Note that this *can* result in a negative "distributionHeight" up to -ValidatorUpdateDelay-1,
			// i.e. at the end of the pre-genesis block (none) = at the beginning of the genesis block.
			// That's fine since this is just used to filter unbonding delegations & redelegations.
			shouldSlash = true
			distributionHeight := height - sdk.ValidatorUpdateDelay - 1
			slashInfo = SlashInfo{
				height:             height,
				power:              power,
				distributionHeight: distributionHeight,
				minHeight:          minHeight,
				minSignedPerWindow: minSignedPerWindow,
			}
			// This value is passed back and the validator is slashed and jailed appropriately
		} else {
			// validator was (a) not found or (b) already jailed so we do not slash
			logger.Info(
				"validator would have been slashed for downtime, but was either not found in store or already jailed",
				"validator", consAddr.String(),
			)
		}
	}
	return
}

func (k Keeper) SlashJailAndUpdateSigningInfo(ctx sdk.Context, consAddr sdk.ConsAddress, slashInfo SlashInfo, signInfo types.ValidatorSigningInfo) types.ValidatorSigningInfo {
	logger := k.Logger(ctx)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlash,
			sdk.NewAttribute(types.AttributeKeyAddress, consAddr.String()),
			sdk.NewAttribute(types.AttributeKeyPower, fmt.Sprintf("%d", slashInfo.power)),
			sdk.NewAttribute(types.AttributeKeyReason, types.AttributeValueMissingSignature),
			sdk.NewAttribute(types.AttributeKeyJailed, consAddr.String()),
		),
	)

	k.sk.Slash(ctx, consAddr, slashInfo.distributionHeight, slashInfo.power, k.SlashFractionDowntime(ctx))
	k.sk.Jail(ctx, consAddr)
	signInfo.JailedUntil = ctx.BlockHeader().Time.Add(k.DowntimeJailDuration(ctx))
	signInfo.MissedBlocksCounter = 0
	signInfo.IndexOffset = 0
	logger.Info(
		"slashing and jailing validator due to liveness fault",
		"height", slashInfo.height,
		"validator", consAddr.String(),
		"min_height", slashInfo.minHeight,
		"threshold", slashInfo.minSignedPerWindow,
		"slashed", k.SlashFractionDowntime(ctx).String(),
		"jailed_until", signInfo.JailedUntil,
	)
	return signInfo
}
