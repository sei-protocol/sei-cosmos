package keeper

import (
	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set params.
func (k Keeper) GetParams(ctx sdk.Context) (params acltypes.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the total set of params.
func (k Keeper) SetParams(ctx sdk.Context, params acltypes.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
