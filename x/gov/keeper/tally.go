package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TODO: Break into several smaller functions for clarity

// Tally iterates over the votes and updates the tally of a proposal based on the voting power of the
// voters
func (keeper Keeper) Tally(ctx sdk.Context, proposal types.Proposal) (passes bool, burnDeposits bool, tallyResults types.TallyResult) {
	results := make(map[types.VoteOption]sdk.Dec)
	results[types.OptionYes] = sdk.ZeroDec()
	results[types.OptionAbstain] = sdk.ZeroDec()
	results[types.OptionNo] = sdk.ZeroDec()
	results[types.OptionNoWithVeto] = sdk.ZeroDec()

	totalVotingPower := sdk.ZeroDec()
	currValidators := make(map[string]types.ValidatorGovInfo)
	// fetch all the bonded validators, insert them into currValidators
	keeper.sk.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		currValidators[validator.GetOperator().String()] = types.NewValidatorGovInfo(
			validator.GetOperator(),
			validator.GetBondedTokens(),
			validator.GetDelegatorShares(),
			sdk.ZeroDec(),
			types.WeightedVoteOptions{},
		)

		ctx.Logger().Info(fmt.Sprintf("Tally:: validator=%s delegatorshares=%d", validator.GetOperator().String(), validator.GetDelegatorShares()))

		return false
	})

	keeper.IterateVotes(ctx, proposal.ProposalId, func(vote types.Vote) bool {
		// if validator, just record it in the map
		voter, err := sdk.AccAddressFromBech32(vote.Voter)

		if err != nil {
			panic(err)
		}

		valAddrStr := sdk.ValAddress(voter.Bytes()).String()
		if val, ok := currValidators[valAddrStr]; ok {
			val.Vote = vote.Options
			currValidators[valAddrStr] = val
		}

		// iterate over all delegations from voter, deduct from any delegated-to validators
		keeper.sk.IterateDelegations(ctx, voter, func(index int64, delegation stakingtypes.DelegationI) (stop bool) {
			valAddrStr := delegation.GetValidatorAddr().String()
			ctx.Logger().Info(fmt.Sprintf("Tally:: delegator addr=%s, voter=%s", valAddrStr, voter))
			if val, ok := currValidators[valAddrStr]; ok {
				// There is no need to handle the special case that validator address equal to voter address.
				// Because voter's voting power will tally again even if there will deduct voter's voting power from validator.
				val.DelegatorDeductions = val.DelegatorDeductions.Add(delegation.GetShares())
				currValidators[valAddrStr] = val

				// delegation shares * bonded / total shares
				votingPower := delegation.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
				ctx.Logger().Info(fmt.Sprintf("Tally:: Before Options val=%s votingPower=%s", val.Address.String(), votingPower.String()))
				for _, option := range vote.Options {
					subPower := votingPower.Mul(option.Weight)
					results[option.Option] = results[option.Option].Add(subPower)
					ctx.Logger().Info(fmt.Sprintf("Tally:: val=%s votingPower=%s, subPower=%s", val.Address.String(), votingPower.String(), subPower.String()))
				}
				totalVotingPower = totalVotingPower.Add(votingPower)
			}

			return false
		})

		keeper.deleteVote(ctx, vote.ProposalId, voter)
		return false
	})
	ctx.Logger().Info(fmt.Sprintf("Tally:: totalVotingPower=%s", totalVotingPower.String()))

	// iterate over the validators again to tally their voting power
	for _, val := range currValidators {
		if len(val.Vote) == 0 {
			ctx.Logger().Info(fmt.Sprintf("Tally:: validtor no vote=%s", val.Address.String()))
			continue
		}

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

		for _, option := range val.Vote {
			ctx.Logger().Info(fmt.Sprintf("Tally:: validtor vote weight=%s", option.Weight.String()))
			subPower := votingPower.Mul(option.Weight)
			results[option.Option] = results[option.Option].Add(subPower)
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}

	ctx.Logger().Info(fmt.Sprintf("Tally:: totalVotingPower=%s", totalVotingPower.String()))

	tallyParams := keeper.GetTallyParams(ctx)
	tallyResults = types.NewTallyResultFromMap(results)

	// TODO: Upgrade the spec to cover all of these cases & remove pseudocode.
	// If there is no staked coins, the proposal fails
	if keeper.sk.TotalBondedTokens(ctx).IsZero() {
		ctx.Logger().Info(fmt.Sprintf("Tally:: No Bonded Tokens :("))
		return false, false, tallyResults
	}

	// If there is not enough quorum of votes, the proposal fails
	percentVoting := totalVotingPower.Quo(keeper.sk.TotalBondedTokens(ctx).ToDec())
	if percentVoting.LT(tallyParams.Quorum) {
		ctx.Logger().Info(fmt.Sprintf("Tally:: NOT ENOUGH QUORAM=%s percentageVoting=%s, tokens=%s", tallyParams.Quorum.String(), percentVoting.String(), keeper.sk.TotalBondedTokens(ctx).String()))
		return false, true, tallyResults
	}

	// If no one votes (everyone abstains), proposal fails
	if totalVotingPower.Sub(results[types.OptionAbstain]).Equal(sdk.ZeroDec()) {
		ctx.Logger().Info(fmt.Sprintf("Tally:: No one voted :("))
		return false, false, tallyResults
	}

	// If more than 1/3 of voters veto, proposal fails
	if results[types.OptionNoWithVeto].Quo(totalVotingPower).GT(tallyParams.VetoThreshold) {
		ctx.Logger().Info(fmt.Sprintf("Tally:: Veto!"))
		return false, true, tallyResults
	}

	// If more than 1/2 of non-abstaining voters vote Yes, proposal passes
	if results[types.OptionYes].Quo(totalVotingPower.Sub(results[types.OptionAbstain])).GT(tallyParams.Threshold) {
		ctx.Logger().Info(fmt.Sprintf("Tally:: Passed!"))
		return true, false, tallyResults
	}

	ctx.Logger().Info(fmt.Sprintf("Tally:: Failed cus less than 1/2!"))

	// If more than 1/2 of non-abstaining voters vote No, proposal fails
	return false, false, tallyResults
}
