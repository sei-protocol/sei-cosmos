package types_test

import (
	"testing"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/types"
	"github.com/stretchr/testify/require"
)

func TestWasmDependencyInvalidBaseContractReference(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	wasmContractAddresses := simapp.AddTestAddrsIncremental(app, ctx, 1, sdk.NewInt(30000000))
	wasmContractAddress := wasmContractAddresses[0]
	wasmDependencyMapping := acltypes.WasmDependencyMapping{
		BaseAccessOps: []*acltypes.WasmAccessOperation{
			{
				Operation:    types.CommitAccessOp(),
				SelectorType: acltypes.AccessOperationSelectorType_NONE,
			},
		},
		BaseContractReferences: []*acltypes.WasmContractReference{
			{
				ContractAddress: wasmContractAddress.String(),
				MessageType:     acltypes.WasmMessageSubtype_QUERY,
				MessageName:     "some_message",
			},
			{
				ContractAddress: "gibberish",
				MessageType:     acltypes.WasmMessageSubtype_EXECUTE,
				MessageName:     "some_message",
			},
		},
	}
	require.Error(t, types.ValidateWasmDependencyMapping(wasmDependencyMapping))
}

func TestWasmDependencyInvalidExecuteContractReference(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	wasmContractAddresses := simapp.AddTestAddrsIncremental(app, ctx, 1, sdk.NewInt(30000000))
	wasmContractAddress := wasmContractAddresses[0]
	wasmDependencyMapping := acltypes.WasmDependencyMapping{
		BaseAccessOps: []*acltypes.WasmAccessOperation{
			{
				Operation:    types.CommitAccessOp(),
				SelectorType: acltypes.AccessOperationSelectorType_NONE,
			},
		},
		ExecuteContractReferences: []*acltypes.WasmContractReferences{
			{
				MessageName: "some_message",
				ContractReferences: []*acltypes.WasmContractReference{
					{
						ContractAddress: wasmContractAddress.String(),
						MessageType:     acltypes.WasmMessageSubtype_EXECUTE,
						MessageName:     "some_message",
					},
					{
						ContractAddress: "gibberish",
						MessageType:     acltypes.WasmMessageSubtype_EXECUTE,
						MessageName:     "some_message",
					},
				},
			},
		},
	}
	require.Error(t, types.ValidateWasmDependencyMapping(wasmDependencyMapping))
}

func TestWasmDependencyInvalidQueryContractReference(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	wasmContractAddresses := simapp.AddTestAddrsIncremental(app, ctx, 1, sdk.NewInt(30000000))
	wasmContractAddress := wasmContractAddresses[0]
	wasmDependencyMapping := acltypes.WasmDependencyMapping{
		BaseAccessOps: []*acltypes.WasmAccessOperation{
			{
				Operation:    types.CommitAccessOp(),
				SelectorType: acltypes.AccessOperationSelectorType_NONE,
			},
		},
		QueryContractReferences: []*acltypes.WasmContractReferences{
			{
				MessageName: "some_message",
				ContractReferences: []*acltypes.WasmContractReference{
					{
						ContractAddress: wasmContractAddress.String(),
						MessageType:     acltypes.WasmMessageSubtype_QUERY,
						MessageName:     "some_message",
					},
					{
						ContractAddress: "gibberish",
						MessageType:     acltypes.WasmMessageSubtype_QUERY,
						MessageName:     "some_message",
					},
				},
			},
		},
	}
	require.Error(t, types.ValidateWasmDependencyMapping(wasmDependencyMapping))
}

func TestWasmDependencyQueryContractReferenceIncorrectMessageType(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	wasmContractAddresses := simapp.AddTestAddrsIncremental(app, ctx, 1, sdk.NewInt(30000000))
	wasmContractAddress := wasmContractAddresses[0]
	wasmDependencyMapping := acltypes.WasmDependencyMapping{
		BaseAccessOps: []*acltypes.WasmAccessOperation{
			{
				Operation:    types.CommitAccessOp(),
				SelectorType: acltypes.AccessOperationSelectorType_NONE,
			},
		},
		QueryContractReferences: []*acltypes.WasmContractReferences{
			{
				MessageName: "some_message",
				ContractReferences: []*acltypes.WasmContractReference{
					{
						ContractAddress: wasmContractAddress.String(),
						MessageType:     acltypes.WasmMessageSubtype_EXECUTE,
						MessageName:     "some_message",
					},
				},
			},
		},
	}
	require.Error(t, types.ValidateWasmDependencyMapping(wasmDependencyMapping))
}
