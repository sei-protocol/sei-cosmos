package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		FeeParams: FeesParams{
			GlobalMinimumFees: sdk.Coins{
				sdk.NewCoin(sdk.DefaultBondDenom, sdk.Int(sdk.NewDecWithPrec(1, 2))),
			},
		},
	}
}
