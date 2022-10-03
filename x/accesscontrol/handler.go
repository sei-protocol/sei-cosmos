package accesscontrol

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/cosmos-sdk/x/accesscontrol/keeper"
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/types"
)

func HandleUpdateResourceDependencyMappingProposal(ctx sdk.Context, k *keeper.Keeper, p *types.UpdateResourceDependencyMappingProposal) error {
	for _, resourceDepMapping := range p.MessageDependencyMapping {
		k.SetResourceDependencyMapping(ctx, resourceDepMapping)
	}
	return nil
}

func NewProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.UpdateResourceDependencyMappingProposal:
			return HandleUpdateResourceDependencyMappingProposal(ctx, &k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized accesscontrol proposal content type: %T", c)
		}
	}
}
