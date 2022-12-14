package server

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cobra"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
)

var removeBlock = false
var appState = false
var tendermintState = false

// NewRollbackCmd creates a command to rollback tendermint and multistore state by one height.
func NewRollbackCmd(appCreator types.AppCreator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "rollback cosmos-sdk and tendermint state by one height",
		Long: `
A state rollback is performed to recover from an incorrect application state transition,
when Tendermint has persisted an incorrect app hash and is thus unable to make
progress. Rollback overwrites a state at height n with the state at height n - 1.
The application also roll back to height n - 1. No blocks are removed, so upon
restarting Tendermint the transactions in block n will be re-executed against the
application.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := GetServerContextFromCmd(cmd)
			cfg := ctx.Config
			home := cfg.RootDir
			db, err := openDB(home)
			if err != nil {
				return err
			}

			app := appCreator(ctx.Logger, db, nil, ctx.Viper)
			fmt.Printf("Last Commit Height=%d\n", app.CommitMultiStore().LastCommitID().Version)

			rollbackHeight := app.CommitMultiStore().LastCommitID().Version - 1
			// rollback tendermint state (block store and tendermint state)
			if tendermintState {
				height, _, err := tmcmd.RollbackState(ctx.Config, removeBlock)
				rollbackHeight = height
				fmt.Printf("Tendermint state rolledback back to version height=%d, removeBlock=%t\n", height, removeBlock)
				if err != nil {
					return fmt.Errorf("failed to rollback tendermint state: %w", err)
				}
			}

			if appState {
				// rollback the multistore (app state)
				if err := app.CommitMultiStore().RollbackToVersion(rollbackHeight); err != nil {
					return fmt.Errorf("failed to rollback to version: %w", err)
				}
				// app.CommitMultiStore().CacheMultiStore().Write()
				app.CommitMultiStore().Commit()
			}

			fmt.Printf("Rolled back to height=%d, removeBlock=%t\n", rollbackHeight, removeBlock)
			fmt.Printf("Last Commit Height=%d\n", app.CommitMultiStore().LastCommitID().Version)
			err = db.Close()
			return err
		},
	}

	cmd.Flags().String(flags.FlagChainID, "sei-chain", "genesis file chain-id, if left blank will use sei")
	cmd.Flags().BoolVar(&removeBlock, "hard", false, "remove last block as well as state")
	cmd.Flags().BoolVar(&appState, "app", false, "remove last block as well as state")
	cmd.Flags().BoolVar(&tendermintState, "tendermint", false, "remove last block as well as state")
	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	return cmd
}
