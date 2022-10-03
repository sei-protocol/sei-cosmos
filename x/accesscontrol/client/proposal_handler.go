package client

import (
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var ProposalHandler = govclient.NewProposalHandler(cli.UpdateResourceDepedencyMappingProposalCmd, nil)
