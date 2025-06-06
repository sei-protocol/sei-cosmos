package upgrade

import (
	"fmt"
	"time"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	"github.com/cosmos/cosmos-sdk/x/upgrade/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// BeginBlock will check if there is a scheduled plan and if it is ready to be executed.
// If the current height is in the provided set of heights to skip, it will skip and clear the upgrade plan.
// If it is ready, it will execute it if the handler is installed, and panic/abort otherwise.
// If the plan is not ready, it will ensure the handler is not registered too early (and abort otherwise).
//
// The purpose is to ensure the binary is switched EXACTLY at the desired block, and to allow
// a migration to be executed if needed upon this switch (migration defined in the new binary)
// skipUpgradeHeightArray is a set of block heights for which the upgrade must be skipped
func BeginBlocker(k keeper.Keeper, ctx sdk.Context, _ abci.RequestBeginBlock) {
	if ctx.IsTracing() {
		return
	}
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	plan, planFound := k.GetUpgradePlan(ctx)

	if !k.DowngradeVerified() {
		k.SetDowngradeVerified(true)
		lastAppliedPlan, _ := k.GetLastCompletedUpgrade(ctx)
		// This check will make sure that we are using a valid binary.
		// It'll panic in these cases if there is no upgrade handler registered for the last applied upgrade.
		// 1. If there is no scheduled upgrade.
		// 2. If the plan is not ready.
		// 3. If the plan is ready and skip upgrade height is set for current height.
		if !planFound || !plan.ShouldExecute(ctx) || (plan.ShouldExecute(ctx) && k.IsSkipHeight(ctx.BlockHeight())) {
			if lastAppliedPlan != "" && !k.HasHandler(lastAppliedPlan) {
				panic(fmt.Sprintf("Wrong app version %d, upgrade handler is missing for %s upgrade plan", ctx.ConsensusParams().Version.AppVersion, lastAppliedPlan))
			}
		}
	}

	if !planFound {
		return
	}

	telemetry.SetGaugeWithLabels(
		[]string{"cosmos", "upgrade", "plan", "height"},
		float32(plan.Height),
		[]metrics.Label{
			{Name: "name", Value: plan.Name},
			{Name: "info", Value: plan.Info},
		},
	)

	// If the plan's block height has passed, then it must be the executed version
	// All major and minor releases are REQUIRED to execute on the scheduled block height
	if plan.ShouldExecute(ctx) {
		// If skip upgrade has been set for current height, we clear the upgrade plan
		if k.IsSkipHeight(ctx.BlockHeight()) {
			skipUpgrade(k, ctx, plan)
			return
		}
		// If we don't have an upgrade handler for this upgrade name, then we need to shutdown
		if !k.HasHandler(plan.Name) {
			panicUpgradeNeeded(k, ctx, plan)
		}
		applyUpgrade(k, ctx, plan)
		return
	}

	details, err := plan.UpgradeDetails()
	if err != nil {
		ctx.Logger().Error("failed to parse upgrade details", "err", err)
	}

	// If running a pending minor release, apply the upgrade if handler is present
	// Minor releases are allowed to run before the scheduled upgrade height, but not required to.
	if details.IsMinorRelease() {
		// if not yet present, then emit a scheduled log (every 100 blocks, to reduce logs)
		if !k.HasHandler(plan.Name) && !k.IsSkipHeight(plan.Height) {
			if ctx.BlockHeight()%100 == 0 {
				ctx.Logger().Info(BuildUpgradeScheduledMsg(plan))
			}
		}
		return
	}

	// if we have a handler for a non-minor upgrade, that means it updated too early and must stop
	if k.HasHandler(plan.Name) {
		downgradeMsg := fmt.Sprintf("BINARY UPDATED BEFORE TRIGGER! UPGRADE \"%s\" - in binary but not executed on chain", plan.Name)
		ctx.Logger().Error(downgradeMsg)
		panic(downgradeMsg)
	}
}

// panicUpgradeNeeded shuts down the node and prints a message that the upgrade needs to be applied.
func panicUpgradeNeeded(k keeper.Keeper, ctx sdk.Context, plan types.Plan) {
	// Write the upgrade info to disk. The UpgradeStoreLoader uses this info to perform or skip
	// store migrations.
	err := k.DumpUpgradeInfoWithInfoToDisk(ctx.BlockHeight(), plan.Name, plan.Info)
	if err != nil {
		panic(fmt.Errorf("unable to write upgrade info to filesystem: %s", err.Error()))
	}

	upgradeMsg := BuildUpgradeNeededMsg(plan)
	ctx.Logger().Error(upgradeMsg)

	panic(upgradeMsg)
}

func applyUpgrade(k keeper.Keeper, ctx sdk.Context, plan types.Plan) {
	ctx.Logger().Info(fmt.Sprintf("applying upgrade \"%s\" at %s", plan.Name, plan.DueAt()))
	k.ApplyUpgrade(ctx, plan)
}

// skipUpgrade logs a message that the upgrade has been skipped and clears the upgrade plan.
func skipUpgrade(k keeper.Keeper, ctx sdk.Context, plan types.Plan) {
	skipUpgradeMsg := fmt.Sprintf("UPGRADE \"%s\" SKIPPED at %d: %s", plan.Name, plan.Height, plan.Info)
	ctx.Logger().Info(skipUpgradeMsg)
	k.ClearUpgradePlan(ctx)
}

// BuildUpgradeNeededMsg prints the message that notifies that an upgrade is needed.
func BuildUpgradeNeededMsg(plan types.Plan) string {
	return fmt.Sprintf("UPGRADE \"%s\" NEEDED at %s: %s", plan.Name, plan.DueAt(), plan.Info)
}

// BuildUpgradeScheduledMsg prints upgrade scheduled message
func BuildUpgradeScheduledMsg(plan types.Plan) string {
	return fmt.Sprintf("UPGRADE \"%s\" SCHEDULED at %s: %s", plan.Name, plan.DueAt(), plan.Info)
}
