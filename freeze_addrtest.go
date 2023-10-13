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
