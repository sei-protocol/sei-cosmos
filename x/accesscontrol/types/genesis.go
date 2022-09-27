package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
)

func DefaultMessageDependencyMapping() []acltypes.MessageDependencyMapping {
	return []acltypes.MessageDependencyMapping{
		{
			MessageKey: "",
			AccessOps: []acltypes.AccessOperation{
				{AccessType: acltypes.AccessType_UNKNOWN, ResourceType: acltypes.ResourceType_ANY, IdentifierTemplate: "*"},
				{AccessType: acltypes.AccessType_COMMIT, ResourceType: acltypes.ResourceType_ANY, IdentifierTemplate: "*"},
			},
		},
	}
}

// NewGenesisState creates a new GenesisState object
func NewGenesisState(params acltypes.Params, messageDependencyMapping []acltypes.MessageDependencyMapping) *acltypes.GenesisState {
	return &acltypes.GenesisState{
		Params:                   params,
		MessageDependencyMapping: messageDependencyMapping,
	}
}

// DefaultGenesisState - default GenesisState used by columbus-2
func DefaultGenesisState() *acltypes.GenesisState {
	return &acltypes.GenesisState{
		Params:                   acltypes.DefaultParams(),
		MessageDependencyMapping: DefaultMessageDependencyMapping(),
	}
}

// ValidateGenesis validates the oracle genesis state
func ValidateGenesis(data acltypes.GenesisState) error {
	for _, mapping := range data.MessageDependencyMapping {
		err := ValidateMessageDependencyMapping(mapping)
		if err != nil {
			return err
		}
	}
	return data.Params.Validate()
}

// GetGenesisStateFromAppState returns x/oracle GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *acltypes.GenesisState {
	var genesisState acltypes.GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
