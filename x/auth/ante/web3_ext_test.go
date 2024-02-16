package ante_test

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	types2 "github.com/evmos/ethermint/types"
)

func (suite *AnteTestSuite) TestWeb3ExtensionOptionsDecorator() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	web3ExtOD := ante.NewWeb3ExtensionOptionsDecorator(suite.app.AccountKeeper, signing.NewSignModeHandlerMap(
		signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
		[]signing.SignModeHandler{
			legacytx.NewStdTxSignModeHandler(),
		},
	))
	antehandler, _ := sdk.ChainAnteDecorators(sdk.DefaultWrappedAnteDecorator(web3ExtOD))

	// no extension options should not trigger an error
	theTx := suite.txBuilder.GetTx()
	_, err := antehandler(suite.ctx, theTx, false)
	suite.Require().NoError(err)

	extOptsTxBldr, ok := suite.txBuilder.(tx.ExtensionOptionsTxBuilder)
	if !ok {
		// if we can't set extension options, this decorator doesn't apply and we're done
		return
	}
	option, err := types.NewAnyWithValue(&types2.ExtensionOptionsWeb3Tx{})
	extOptsTxBldr.SetExtensionOptions(option)
	theTx = suite.txBuilder.GetTx()
	_, err = antehandler(suite.ctx, theTx, false)
	suite.Require().NoError(err)
}
