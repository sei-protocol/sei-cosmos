package keeper

// DONTCOVER

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
)

// RegisterInvariants registers all governance invariants
func RegisterInvariants(ir sdk.InvariantRegistry, keeper Keeper, bk types.BankKeeper) {
	ir.RegisterRoute(types.ModuleName, "module-account", ModuleAccountInvariant(keeper, bk))
}

// AllInvariants runs all invariants of the governance module
func AllInvariants(keeper Keeper, bk types.BankKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		return ModuleAccountInvariant(keeper, bk)(ctx)
	}
}

// ModuleAccountInvariant checks that the module account coins are at least
// sufficient to cover the sum of deposit amounts held on store
func ModuleAccountInvariant(keeper Keeper, bk types.BankKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var expectedDeposits sdk.Coins

		keeper.IterateAllDeposits(ctx, func(deposit types.Deposit) bool {
			expectedDeposits = expectedDeposits.Add(deposit.Amount...)
			return false
		})

		macc := keeper.GetGovernanceAccount(ctx)
		balances := bk.GetAllBalances(ctx, macc.GetAddress())
		broken := !balances.IsAllGTE(expectedDeposits)

		return sdk.FormatInvariant(types.ModuleName, "deposits",
			fmt.Sprintf("\tactual gov ModuleAccount coins: %s\n\tminimum required deposit amounts: %s\n",
				balances, expectedDeposits)), broken
	}
}
