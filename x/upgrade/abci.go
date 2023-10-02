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

// BeginBlocker checks for a scheduled upgrade plan and determines whether it's time to execute the upgrade.
// If DowngradeVerified is false, it sets it to true and performs a check to ensure that the binary version is correct.
// If no plan is found, or the plan should not execute yet, it returns early.
// If the plan is set to be skipped at the current block height, it clears the upgrade plan.
// If the plan should execute and a handler for the upgrade is present, it applies the upgrade.
// If no handler is found, it panics indicating that an upgrade is needed.
// For minor releases that are pending, if a handler is not found and the plan height is not set to be skipped,
// it logs a scheduled upgrade message every 100 blocks.
// If a handler for a non-minor upgrade is found before the plan should execute, it panics indicating that the binary
// has been updated too early.
//
// The purpose of BeginBlocker is to ensure that the upgrade binary is switched at the exact block height specified
// in the plan, and to coordinate the execution of any necessary migrations defined in the new binary.
// skipUpgradeHeightArray is a set of block heights for which the upgrade must be skipped.
func BeginBlocker(k keeper.Keeper, ctx sdk.Context, _ abci.RequestBeginBlock) {
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
		return
	}

	details, err := plan.UpgradeDetails()
	if err != nil {
		ctx.Logger().Error("error parsing upgrade info", "err", err.Error())
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
	ctx = ctx.WithBlockGasMeter(sdk.NewInfiniteGasMeter())
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
