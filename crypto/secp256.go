package crypto

import (
	"github.com/copernet/secp256k1-go/secp256k1"
)

var (
	secp256k1Context *secp256k1.Context
)

func InitSecp256() {
	secp256k1Context, _ = secp256k1.ContextCreate(secp256k1.ContextSign | secp256k1.ContextVerify)
}
