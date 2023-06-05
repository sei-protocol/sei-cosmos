package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestParams(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	keeper := app.AccessControlKeeper

	response, err := app.AccessControlKeeper.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, keeper.GetParams(ctx), response.Params)
}

func TestResourceDependencyMappingFromMessageKey(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	keeper := app.AccessControlKeeper
	_, err := keeper.ResourceDependencyMappingFromMessageKey(
		sdk.WrapSDKContext(ctx),
		&types.ResourceDependencyMappingFromMessageKeyRequest{MessageKey: "key"},
	)

	require.NoError(t, err)
	require.Equal(t, keeper.GetResourceDependencyMapping(ctx, types.MessageKey("key")), types.ResourceDependencyMappingFromMessageKeyResponse{})
}

func TestWasmDependencyMappingCall(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	keeper := app.AccessControlKeeper
	_, err := keeper.WasmDependencyMapping(sdk.WrapSDKContext(ctx), &types.WasmDependencyMappingRequest{ContractAddress: "sei1qy352eufqy352eufqy352eufqy35neufqy35"})
	require.NoError(t, err)
}

func TestListResourceDependencyMapping(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	keeper := app.AccessControlKeeper
	_, err := keeper.ListResourceDependencyMapping(sdk.WrapSDKContext(ctx), &types.ListResourceDependencyMappingRequest{})
	require.NoError(t, err)
}

func TestListWasmDependencyMapping(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	keeper := app.AccessControlKeeper
	_, err := keeper.ListWasmDependencyMapping(sdk.WrapSDKContext(ctx), &types.ListWasmDependencyMappingRequest{})
	require.NoError(t, err)
}
