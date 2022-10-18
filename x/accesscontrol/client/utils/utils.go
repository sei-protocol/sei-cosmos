package utils

import (
	"io/ioutil"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkacltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/types"
)

type (
	UpdateWasmDependencyMappingProposalJSON struct {
		Title                  string                              `json:"title" yaml:"title"`
		Description            string                              `json:"description" yaml:"description"`
		ContractAddress        string                              `json:"contract_address" yaml:"contract_address"`
		WasmDependencyMappings []sdkacltypes.WasmDependencyMapping `json:"wasm_dependency_mappings" yaml:"wasm_dependency_mappings"`
		Deposit                string                              `json:"deposit" yaml:"deposit"`
	}
)

func ParseMsgUpdateResourceDependencyMappingProposalFile(cdc codec.JSONCodec, proposalFile string) (types.MsgUpdateResourceDependencyMappingProposalJsonFile, error) {
	proposal := types.MsgUpdateResourceDependencyMappingProposalJsonFile{}

	contents, err := ioutil.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	cdc.MustUnmarshalJSON(contents, &proposal)

	return proposal, nil
}

func ParseUpdateWasmDependencyMappingProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (UpdateWasmDependencyMappingProposalJSON, error) {
	proposal := UpdateWasmDependencyMappingProposalJSON{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}
