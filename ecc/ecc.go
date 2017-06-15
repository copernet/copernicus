package ecc

import (
	"github.com/btccom/secp256k1-go/secp256k1"
	"fmt"
)

var secp256k1Context *secp256k1.Context

func init() {
	var err error
	secp256k1Context, err = secp256k1.ContextCreate(secp256k1.ContextSign | secp256k1.ContextVerify)
	if err != nil {
		fmt.Println(err)
	}
}
