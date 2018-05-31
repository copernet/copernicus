package main

import (
	"crypto/rand"
	"github.com/btccom/secp256k1-go/secp256k1"
	"io"
	"log"
)

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func Rand32() [32]byte {
	key := [32]byte{}
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		panic(err)
	}
	return key
}

func main() {
	log.Println("HI")

	params := uint(secp256k1.ContextSign | secp256k1.ContextVerify)
	ctx, err := secp256k1.ContextCreate(params)
	if err != nil {
		panic(err)
	}
	log.Printf("%+v\n", ctx)

	clone, err := secp256k1.ContextClone(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("%+v\n", clone)

	secp256k1.ContextDestroy(clone)
	log.Printf("%+v\n", clone)

	res := secp256k1.ContextRandomize(ctx, Rand32())
	log.Printf("Result of randomize: %d \n", res)

}
