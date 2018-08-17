package chainparams

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestBitcoinParamsTxData(t *testing.T) {
	hash := TestNetGenesisBlock.Header.GetHash()
	fmt.Println("genesis hash : ", hash.String())
	fmt.Println("hash 000 : ", hash.String())
	fmt.Println("hash : ", hex.EncodeToString(TestNetGenesisHash[:]))

	fmt.Println("time : ", time.Unix(1296688602, 0).UnixNano())
}
