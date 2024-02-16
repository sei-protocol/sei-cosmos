package ante

import (
	"github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	evmante "github.com/evmos/ethermint/app/ante"
)

// Web3ExtensionOptionsDecorator is an AnteDecorator that is a standard cosmos tx with an Ethereum signature
// This overrides x/auth/ext.go in cosmos which is a RejectExtensionOptionsDecorator
type Web3ExtensionOptionsDecorator struct {
	ak      AccountKeeper
	handler signing.SignModeHandler
}

// NewWeb3ExtensionOptionsDecorator creates a new Web3ExtensionOptionsDecorator
func NewWeb3ExtensionOptionsDecorator(ak AccountKeeper, handler signing.SignModeHandler) Web3ExtensionOptionsDecorator {
	return Web3ExtensionOptionsDecorator{
		ak:      ak,
		handler: handler,
	}
}

var _ types.AnteDecorator = Web3ExtensionOptionsDecorator{}

// AnteHandle implements the AnteDecorator.AnteHandle method
func (r Web3ExtensionOptionsDecorator) AnteHandle(ctx types.Context, tx types.Tx, simulate bool, next types.AnteHandler) (newCtx types.Context, err error) {
	isExt, err := IsExtensionOptionsWeb3Tx(tx)
	if err != nil {
		return ctx, err
	}
	// Do an EIP-712 signature verification
	if isExt {
		sigVerification := evmante.NewLegacyEip712SigVerificationDecorator(r.ak, r.handler)
		return sigVerification.AnteHandle(ctx, tx, simulate, next)
	}
	return next(ctx, tx, simulate)
}

func IsExtensionOptionsWeb3Tx(tx types.Tx) (isExt bool, err error) {
	if hasExtOptsTx, ok := tx.(HasExtensionOptionsTx); ok {
		opts := hasExtOptsTx.GetExtensionOptions()
		if len(opts) == 1 && opts[0].GetTypeUrl() == "/ethermint.types.v1.ExtensionOptionsWeb3Tx" {
			return true, nil
		} else {
			return true, sdkerrors.ErrUnknownExtensionOptions
		}
	}
	return false, nil
}
