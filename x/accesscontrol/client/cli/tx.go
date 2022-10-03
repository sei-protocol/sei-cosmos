package cli

import (
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/client/utils"
	"github.com/cosmos/cosmos-sdk/x/accesscontrol/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/spf13/cobra"
)


func UpdateResourceDependencyMappingProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-resource-dependency-mapping [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit an UpdateResourceDependencyMapping proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			println("UpdateResourceDependencyMappingProposalCmd:HANDLING CLI REQUEST")
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := utils.ParseUpdateResourceDependencyMappingProposalFile(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			println("UpdateResourceDependencyMappingProposalCmd:GETTING FROM ADDRESS")
			from := clientCtx.GetFromAddress()

			content := types.UpdateResourceDependencyMappingProposal{
				Title:	proposal.Title,
				Description: proposal.Description,
				MessageDependencyMapping: proposal.MessageDependencyMapping,
			}

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				println("Unable to prase coin from UpdateResourceDependencyMapping from CLI")
				return err
			}

			println("UpdateResourceDependencyMappingProposalCmd:SUBMITTING PROPOSAL")
			msg, err := govtypes.NewMsgSubmitProposal(&content, deposit, from)
			if err != nil {
				println("Unable to submit proposal for UpdateResourceDependencyMapping from CLI")
				return err
			}
			println("UpdateResourceDependencyMappingProposalCmd:SENDING!")
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	return cmd
}
