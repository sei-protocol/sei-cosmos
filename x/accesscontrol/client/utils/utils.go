package utils

import (
	"io/ioutil"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/types"
)

func ParseUpdateResourceDepedencyMappingProposalFile(cdc codec.JSONCodec, proposalFile string) (types.UpdateResourceDepedencyMappingProposalJsonFile, error) {
	proposal := types.UpdateResourceDepedencyMappingProposalJsonFile{}

	contents, err := ioutil.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	cdc.MustUnmarshalJSON(contents, &proposal)

	return proposal, nil
}
