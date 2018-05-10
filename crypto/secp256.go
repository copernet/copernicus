package crypto

import (
	"fmt"

	"github.com/btcboost/secp256k1-go/secp256k1"
)

var (
	secp256k1Context *secp256k1.Context
)

func init() {
	secp256k1Context, _ = secp256k1.ContextCreate(secp256k1.ContextSign | secp256k1.ContextVerify)
	fmt.Println("elliptic curve cryptography context init")
}
