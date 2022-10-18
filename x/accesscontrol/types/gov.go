package types

import (
	"fmt"
	"strings"

	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalUpdateResourceDependencyMapping = "UpdateResourceDependencyMapping"
	ProposalUpdateWasmDependencyMapping     = "UpdateWasmDependencyMapping"
)

func init() {
	// for routing
	govtypes.RegisterProposalType(ProposalUpdateResourceDependencyMapping)
	// for marshal and unmarshal
	govtypes.RegisterProposalTypeCodec(&MsgUpdateResourceDependencyMappingProposal{}, "tokenfactory/MsgUpdateResourceDependencyMappingProposal")
}

var _ govtypes.Content = &MsgUpdateResourceDependencyMappingProposal{}

func NewMsgUpdateResourceDependencyMappingProposal(title, description string, messageDependencyMapping []acltypes.MessageDependencyMapping) *MsgUpdateResourceDependencyMappingProposal {
	return &MsgUpdateResourceDependencyMappingProposal{title, description, messageDependencyMapping}
}

func (p *MsgUpdateResourceDependencyMappingProposal) GetTitle() string { return p.Title }

func (p *MsgUpdateResourceDependencyMappingProposal) GetDescription() string { return p.Description }

func (p *MsgUpdateResourceDependencyMappingProposal) ProposalRoute() string { return RouterKey }

func (p *MsgUpdateResourceDependencyMappingProposal) ProposalType() string {
	return ProposalUpdateResourceDependencyMapping
}

func (p *MsgUpdateResourceDependencyMappingProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	return err
}

func (p MsgUpdateResourceDependencyMappingProposal) String() string {
	var b strings.Builder
	b.WriteString(
		fmt.Sprintf(`Add Creators to Denom Fee Whitelist Proposal:
			Title:       %s
			Description: %s
			Changes:
			`,
			p.Title, p.Description))

	for _, depMapping := range p.MessageDependencyMapping {
		b.WriteString(fmt.Sprintf(`		Change:
			MessageDependencyMapping: %s
		`, depMapping.String()))
	}
	return b.String()
}

func NewMsgUpdateWasmDependencyMappingProposal(title, description, contractAddr string, messageDependencyMapping []acltypes.WasmDependencyMapping) *MsgUpdateWasmDependencyMappingProposal {
	return &MsgUpdateWasmDependencyMappingProposal{title, description, contractAddr, messageDependencyMapping}
}

func (p *MsgUpdateWasmDependencyMappingProposal) GetTitle() string { return p.Title }

func (p *MsgUpdateWasmDependencyMappingProposal) GetDescription() string { return p.Description }

func (p *MsgUpdateWasmDependencyMappingProposal) ProposalRoute() string { return RouterKey }

func (p *MsgUpdateWasmDependencyMappingProposal) ProposalType() string {
	return ProposalUpdateWasmDependencyMapping
}

func (p *MsgUpdateWasmDependencyMappingProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	return err
}

func (p MsgUpdateWasmDependencyMappingProposal) String() string {
	var b strings.Builder
	b.WriteString(
		fmt.Sprintf(`Add Creators to Denom Fee Whitelist Proposal:
			Title:       %s
			Description: %s
			ContractAddress: %s
			Changes:
			`,
			p.Title, p.Description, p.ContractAddress))

	for _, depMapping := range p.WasmDependencyMapping {
		b.WriteString(fmt.Sprintf(`		Change:
			MessageDependencyMapping: %s
		`, depMapping.String()))
	}
	return b.String()
}
