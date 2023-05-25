package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		FeesParams: FeesParams{
			GlobalMinimumGasPrices: sdk.DecCoins{
				sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, sdk.NewDecWithPrec(1, 2)),
			},
		},
	}
}

func (gs GenesisState) Validate() error {
	return gs.FeesParams.Validate()
}
