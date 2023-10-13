https://github.com/classic-terra/core/blob/v1.1.0-freeze-addr/custom/auth/ante/freeze_addr.go

//custom/auth/ante/freeze_addr_test.go
//code to blacklist wallet
package ante_test

import (
	"github.com/classic-terra/core/custom/auth/ante"
	"github.com/classic-terra/core/types"
	core "github.com/classic-terra/core/types"
	markettypes "github.com/classic-terra/core/x/market/types"
	wasmtypes "github.com/classic-terra/core/x/wasm/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
)

// go test -v -run ^TestAnteTestSuite/TestBlockAddrTx$ github.com/classic-terra/core/custom/auth/ante
func (suite *AnteTestSuite) TestBlockAddrTx() {
	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	priv2, _, addr2 := testdata.KeyTestPubAddr()
	priv3, _, addr3 := testdata.KeyTestPubAddr()

	ante.BlockedAddr[addr1.String()] = true
	ante.BlockedAddr[addr2.String()] = true

	// prepare amount
	sendAmount := int64(1000000)
	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount))

	testCases := []struct {
		name    string
		msg     sdk.Msg
		privs   []cryptotypes.PrivKey
		blocked bool
	}{
		{
			"send from blocked address",
			banktypes.NewMsgSend(addr1, addr3, sendCoins),
			[]cryptotypes.PrivKey{priv1},
			true,
		},
		{
			"multisend from blocked address",
			banktypes.NewMsgMultiSend([]banktypes.Input{
				{Address: addr2.String(), Coins: sendCoins},
			}, []banktypes.Output{
				{Address: addr3.String(), Coins: sendCoins},
			}),
			[]cryptotypes.PrivKey{priv2},
			true,
		},
		{
			"swap from blocked address",
			markettypes.NewMsgSwapSend(addr2, addr3, sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount), core.MicroLunaDenom),
			[]cryptotypes.PrivKey{priv2},
			true,
		},
		{
			"execute contract from blocked address",
			wasmtypes.NewMsgExecuteContract(addr1, nil, nil, sendCoins),
			[]cryptotypes.PrivKey{priv1},
			true,
		},
		{
			"instantiate contract from blocked address",
			wasmtypes.NewMsgInstantiateContract(addr1, nil, 0, nil, sendCoins),
			[]cryptotypes.PrivKey{priv1},
			true,
		},
		{
			"send from not blocked address",
			banktypes.NewMsgSend(addr3, addr1, sendCoins),
			[]cryptotypes.PrivKey{priv3},
			false,
		},
		{
			"transfer from blocked address",
			ibctypes.NewMsgTransfer("transfer", "channel-0", sendCoins[0], addr1.String(), "offchain_addr", ibcclienttypes.ZeroHeight(), 0),
			[]cryptotypes.PrivKey{priv1},
			true,
		},
		{
			"transfer from non blocked address",
			ibctypes.NewMsgTransfer("transfer", "channel-0", sendCoins[0], addr3.String(), "offchain_addr", ibcclienttypes.ZeroHeight(), 0),
			[]cryptotypes.PrivKey{priv3},
			false,
		},
	}

	for _, testcase := range testCases {
		suite.Run(testcase.name, func() {
			suite.SetupTest(true) // setup
			suite.ctx = suite.ctx.WithBlockHeight(types.FreezeAddrHeight + 1)
			suite.ctx = suite.ctx.WithChainID("columbus-5")

			feeAmount := testdata.NewTestFeeAmount()
			gasLimit := testdata.NewTestGasLimit()
			suite.Require().NoError(suite.txBuilder.SetMsgs(testcase.msg))
			suite.txBuilder.SetFeeAmount(feeAmount)
			suite.txBuilder.SetGasLimit(gasLimit)

			tx, err := suite.CreateTestTx(testcase.privs, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
			suite.Require().NoError(err)

			antehandler := sdk.ChainAnteDecorators(ante.NewFreezeAddrDecorator())
			_, err = antehandler(suite.ctx, tx, false)

			if testcase.blocked {
				suite.Require().ErrorContains(err, "blocked address")
			} else {
				suite.Require().NoError(err)
			}
		})
	}
	
	
	
	
}
	//custom/auth/ante/freeze_addr.go
	//code to freeze wallet
package ante

import (
	"fmt"

	"github.com/classic-terra/core/types"
	marketexported "github.com/classic-terra/core/x/market/exported"
	wasmexported "github.com/classic-terra/core/x/wasm/exported"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
)

var (
	BlockedAddr = map[string]bool{}
)

type FreezeAddrDecorator struct{}

func NewFreezeAddrDecorator() FreezeAddrDecorator {
	return FreezeAddrDecorator{}
}

func (fad FreezeAddrDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Do not proceed if you are below this block height
	currHeight := ctx.BlockHeight()

	if simulate || ctx.ChainID() != "columbus-5" || currHeight < types.FreezeAddrHeight {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
		switch v := msg.(type) {
		case *banktypes.MsgSend:
			if _, ok := BlockedAddr[v.FromAddress]; ok {
				return ctx, fmt.Errorf("blocked address %s", v.FromAddress)
			}
		case *banktypes.MsgMultiSend:
			for _, addr := range v.Inputs {
				if _, ok := BlockedAddr[addr.Address]; ok {
					return ctx, fmt.Errorf("blocked address %s", addr.Address)
				}
			}
		case *marketexported.MsgSwapSend:
			if _, ok := BlockedAddr[v.FromAddress]; ok {
				return ctx, fmt.Errorf("blocked address %s", v.FromAddress)
			}
		case *wasmexported.MsgExecuteContract:
			if _, ok := BlockedAddr[v.Sender]; ok {
				return ctx, fmt.Errorf("blocked address %s", v.Sender)
			}
		case *wasmexported.MsgInstantiateContract:
			if _, ok := BlockedAddr[v.Sender]; ok {
				return ctx, fmt.Errorf("blocked address %s", v.Sender)
			}
		case *ibctypes.MsgTransfer:
			if _, ok := BlockedAddr[v.Sender]; ok {
				return ctx, fmt.Errorf("blocked address %s", v.Sender)
			}
		}
	}

	return next(ctx, tx, simulate)
}
