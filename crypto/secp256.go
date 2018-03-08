package crypto

import (
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/secp256k1-go/secp256k1"
)

var (
	secp256k1Context *secp256k1.Context
	log              = logs.NewLogger()
)

func init() {
	secp256k1Context, _ = secp256k1.ContextCreate(secp256k1.ContextSign | secp256k1.ContextVerify)
	log.Info("elliptic curve cryptography context init")
}
