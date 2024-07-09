package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const chainUpgradeGuide = "https://docs.cosmos.network/master/migrations/chain-upgrade-guide-040.html"

// ValidateGenesisCmd takes a genesis file, and makes sure that it is valid.
func ValidateGenesisCmd(mbm module.BasicManager) *cobra.Command {
	return &cobra.Command{
		Use:   "validate-genesis [file]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "validates the genesis file at the default location or at the location passed as an arg",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx := client.GetClientContextFromCmd(cmd)

			cdc := clientCtx.Codec

			// Load default if passed no args, otherwise load passed file
			var genesis string
			if len(args) == 0 {
				genesis = serverCtx.Config.GenesisFile()
			} else {
				genesis = args[0]
			}

			genDoc, err := validateGenDoc(genesis)
			if err != nil {
				return err
			}

			var genState map[string]json.RawMessage
			if err = json.Unmarshal(genDoc.AppState, &genState); err != nil {
				return fmt.Errorf("error unmarshalling genesis doc %s: %s", genesis, err.Error())
			}

			if err = mbm.ValidateGenesis(cdc, clientCtx.TxConfig, genState); err != nil {
				return fmt.Errorf("error validating genesis file %s: %s", genesis, err.Error())
			}

			fmt.Printf("File at %s is a valid genesis file\n", genesis)
			return nil
		},
	}
}

type AppState struct {
	Module string          `json:"module"`
	Data   json.RawMessage `json:"data"`
}

type ModuleState struct {
	AppState AppState `json:"app_state"`
}

func parseModule(jsonStr string) (*ModuleState, error) {
	var module ModuleState
	err := json.Unmarshal([]byte(jsonStr), &module)
	if err != nil {
		return nil, err
	}
	if module.AppState.Module == "" {
		return nil, fmt.Errorf("module name is empty")
	}
	return &module, nil
}

// ValidateGenesisCmd takes a genesis file, and makes sure that it is valid.
func ValidateGenesisStreamCmd(mbm module.BasicManager) *cobra.Command {
	return &cobra.Command{
		Use:   "validate-genesis-stream [file]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "validates the genesis file at the default location or at the location passed as an arg",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx := client.GetClientContextFromCmd(cmd)

			cdc := clientCtx.Codec

			// Load default if passed no args, otherwise load passed file
			var genesis string
			if len(args) == 0 {
				genesis = serverCtx.Config.GenesisFile()
			} else {
				genesis = args[0]
			}

			lines := ingestGenesisFileLineByLine(genesis)
			if err != nil {
				return err
			}
			fmt.Println("lines = ", lines)

			// for each line, parse it as a module first
			// if it's a new module then open up a new channel for this module
			// call mbm.ValidateGenesisStream for this new module
			// mbm.ValidateGenesisStream should be in a goroutine??
			// it should remain open until we see the next module OR GenDoc
			// if we see the same module again, we should send the data to the same channel
			// also make sure you don't see a module twice--that's an error

			genesisCh := make(chan json.RawMessage)
			doneCh := make(chan struct{})
			errCh := make(chan error, 1)
			seenModules := make(map[string]bool)
			prevModule := ""
			var moduleName string
			var genDoc *tmtypes.GenesisDoc
			go func() {
				for line := range lines {
					moduleState, err := parseModule(line)
					// determine module name or genesisDoc
					if err != nil {
						genDoc, err = tmtypes.GenesisDocFromJSON([]byte(line))
						if err != nil {
							errCh <- fmt.Errorf("error unmarshalling genesis doc %s: %s", genesis, err.Error())
							return
						}
						moduleName = "genesisDoc"
					} else {
						moduleName = moduleState.AppState.Module
					}
					if seenModules[moduleName] {
						errCh <- fmt.Errorf("module %s seen twice in genesis file", moduleName)
						return
					}
					if prevModule != moduleName { // new module
						if prevModule != "" && prevModule != "genesisDoc" {
							doneCh <- struct{}{}
						}
						seenModules[prevModule] = true
						if moduleName != "genesisDoc" {
							fmt.Println("In ValidateGenesisStreamCmd, kicking off mbm.ValidateGenesisStream for module ", moduleName)
							go mbm.ValidateGenesisStream(cdc, clientCtx.TxConfig, moduleName, genesisCh, doneCh, errCh)
							genesisCh <- moduleState.AppState.Data
						} else {
							err = genDoc.ValidateAndComplete()
							if err != nil {
								errCh <- fmt.Errorf("error validating genesis doc %s: %s", genesis, err.Error())
							}
						}
					} else { // same module
						fmt.Println("Sending data to channel for module ", moduleName)
						genesisCh <- moduleState.AppState.Data
					}
					prevModule = moduleName
				}
				errCh <- nil
			}()
			err = <-errCh
			return err
		},
	}
}

const bufferSize = 100000

func ingestGenesisFileLineByLine(filename string) <-chan string {
	lines := make(chan string)

	go func() {
		defer close(lines)

		file, err := os.Open(filename)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		buffer := make([]byte, bufferSize)
		lineBuf := new(strings.Builder)

		for {
			bytesRead, err := reader.Read(buffer)
			if err != nil && err != io.EOF {
				fmt.Println("Error reading file:", err)
				return
			}

			chunk := buffer[:bytesRead]
			for len(chunk) > 0 {
				i := bytes.IndexByte(chunk, '\n')
				if i >= 0 {
					lineBuf.Write(chunk[:i])
					lines <- lineBuf.String()
					lineBuf.Reset()
					chunk = chunk[i+1:]
				} else {
					lineBuf.Write(chunk)
					break
				}
			}

			if err == io.EOF {
				if lineBuf.Len() > 0 {
					lines <- lineBuf.String()
				}
				break
			}
		}
	}()

	return lines
}

// validateGenDoc reads a genesis file and validates that it is a correct
// Tendermint GenesisDoc. This function does not do any cosmos-related
// validation.
func validateGenDoc(importGenesisFile string) (*tmtypes.GenesisDoc, error) {
	genDoc, err := tmtypes.GenesisDocFromFile(importGenesisFile)
	if err != nil {
		return nil, fmt.Errorf("%s. Make sure that"+
			" you have correctly migrated all Tendermint consensus params, please see the"+
			" chain migration guide at %s for more info",
			err.Error(), chainUpgradeGuide,
		)
	}

	return genDoc, nil
}
