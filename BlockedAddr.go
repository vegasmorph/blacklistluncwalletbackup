package main

import (
	"github.com/classic-terra/core/custom/auth/ante" // Import appropriate package path
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func main() {
	// Replace 'ante.BlockedAddr' with the correct package path
	ante.BlockedAddr["TERRA1QYW695VAXJ7JL6S4U564C6XKFE59KERCG0H88W"] = true
}
