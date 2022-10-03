package types

import (
	"fmt"
	"strings"

	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalUpdateResourceDependencyMapping = "UpdateResourceDependencyMapping"
)

func init() {
	// for routing
	govtypes.RegisterProposalType(ProposalUpdateResourceDependencyMapping)
	// for marshal and unmarshal
	govtypes.RegisterProposalTypeCodec(&UpdateResourceDependencyMappingProposal{}, "tokenfactory/UpdateResourceDependencyMappingProposal")
}

var _ govtypes.Content = &UpdateResourceDependencyMappingProposal{}

func NewRegisterPairsProposal(title, description string, messageDependencyMapping []acltypes.MessageDependencyMapping) UpdateResourceDependencyMappingProposal {
	return UpdateResourceDependencyMappingProposal{
		Title:       title,
		Description: description,
		MessageDependencyMapping : messageDependencyMapping,
	}
}

func NewUpdateResourceDependencyMappingProposal(title, description string, messageDependencyMapping []acltypes.MessageDependencyMapping) *UpdateResourceDependencyMappingProposal {
	return &UpdateResourceDependencyMappingProposal{title, description, messageDependencyMapping}
}

func (p *UpdateResourceDependencyMappingProposal) GetTitle() string { return p.Title }

func (p *UpdateResourceDependencyMappingProposal) GetDescription() string { return p.Description }

func (p *UpdateResourceDependencyMappingProposal) ProposalRoute() string { return RouterKey }

func (p *UpdateResourceDependencyMappingProposal) ProposalType() string {
	return ProposalUpdateResourceDependencyMapping
}

func (p *UpdateResourceDependencyMappingProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	return err
}

func (p UpdateResourceDependencyMappingProposal) String() string {
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
