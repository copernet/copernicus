package chainparams

import (
	"testing"
	"fmt"
	"encoding/hex"
	"time"
)

func TestBitcoinParamsTxData(t *testing.T) {
	hash := TestNet3GenesisBlock.Header.GetHash()
	fmt.Println("genesis hash : ", hash.String() )
	fmt.Println("hash 000 : ",  hash.String())
	fmt.Println("hash : ",  hex.EncodeToString(TestNet3GenesisHash[:]))

	fmt.Println("time : ", time.Unix(1296688602, 0).UnixNano())
}