package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	tmcfg "github.com/tendermint/tendermint/config"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.True(t, cfg.GetMinGasPrices().IsZero())
}

func TestSetMinimumFees(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetMinGasPrices(sdk.DecCoins{sdk.NewInt64DecCoin("foo", 5)})
	require.Equal(t, "5.000000000000000000foo", cfg.MinGasPrices)
}

func TestSetSnapshotDirectory(t *testing.T) {
	cfg := DefaultConfig()
	require.Equal(t, "", cfg.StateSync.SnapshotDirectory)
}

func TestValidateStatesync(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetMinGasPrices(sdk.DecCoins{sdk.NewInt64DecCoin("foo", 5)})
	// test statesync enabled
	cfg.StateSync.SnapshotInterval = 2000
	err := cfg.ValidateBasic(tmcfg.DefaultConfig())
	require.NoError(t, err)
	// test invalid snapshot interval but noerror because SC not enabled
	cfg.StateCommit.SnapshotInterval = 1000
	err = cfg.ValidateBasic(tmcfg.DefaultConfig())
	require.NoError(t, err)
	// test invalid snapshot interval with err
	cfg.StateCommit.Enable = true
	err = cfg.ValidateBasic(tmcfg.DefaultConfig())
	require.Error(t, err)
}

func TestSetConcurrencyWorkers(t *testing.T) {
	cfg := DefaultConfig()
	require.Equal(t, DefaultConcurrencyWorkers, cfg.ConcurrencyWorkers)
}

func TestOCCEnabled(t *testing.T) {
	cfg := DefaultConfig()
	require.False(t, cfg.OccEnabled)

	cfg.BaseConfig.OccEnabled = true
	require.True(t, cfg.OccEnabled)
}
