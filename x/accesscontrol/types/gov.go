package types

import (
	"fmt"
	"strings"

	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalUpdateResourceDepedencyMapping = "UpdateResourceDepedencyMapping"
)

func init() {
	// for routing
	govtypes.RegisterProposalType(ProposalUpdateResourceDepedencyMapping)
	// for marshal and unmarshal
	govtypes.RegisterProposalTypeCodec(&UpdateResourceDepedencyMappingProposal{}, "tokenfactory/UpdateResourceDepedencyMappingProposal")
}

var _ govtypes.Content = &UpdateResourceDepedencyMappingProposal{}

func NewRegisterPairsProposal(title, description string, messageDependencyMapping []acltypes.MessageDependencyMapping) UpdateResourceDepedencyMappingProposal {
	return UpdateResourceDepedencyMappingProposal{
		Title:       title,
		Description: description,
		MessageDependencyMapping : messageDependencyMapping,
	}
}

func NewUpdateResourceDepedencyMappingProposal(title, description string, messageDependencyMapping []acltypes.MessageDependencyMapping) *UpdateResourceDepedencyMappingProposal {
	return &UpdateResourceDepedencyMappingProposal{title, description, messageDependencyMapping}
}

func (p *UpdateResourceDepedencyMappingProposal) GetTitle() string { return p.Title }

func (p *UpdateResourceDepedencyMappingProposal) GetDescription() string { return p.Description }

func (p *UpdateResourceDepedencyMappingProposal) ProposalRoute() string { return RouterKey }

func (p *UpdateResourceDepedencyMappingProposal) ProposalType() string {
	return ProposalUpdateResourceDepedencyMapping
}

func (p *UpdateResourceDepedencyMappingProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	return err
}

func (p UpdateResourceDepedencyMappingProposal) String() string {
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
