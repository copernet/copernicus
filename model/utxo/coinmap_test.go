package utxo

import (
	"testing"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/txout"
)

func TestCoinMap(t *testing.T) {
	necm := NewEmptyCoinsMap()

	if len(necm.cacheCoins) != 0 || necm.hashBlock != util.HashZero {
		t.Error("init empty coin map failed.")
	}

	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	script2 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout2 := txout.NewTxOut(3, script2)

	necm.cacheCoins[outpoint1] = &Coin{
		txOut:         *txout2,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         false,
	}

	c := necm.GetCoin(&outpoint1)
	if c != necm.cacheCoins[outpoint1] {
		t.Error("get coin failed.")
	}

	hash2 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a7")
	outpoint2 := outpoint.OutPoint{Hash: *hash2, Index: 0}

	script1 := script.NewEmptyScript()
	txout1 := txout.NewTxOut(2, script1)

	necm.cacheCoins[outpoint2] = &Coin{
		txOut:         *txout1,
		height:        100012,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         true,
		fresh:         false,
	}

	necm.AddCoin(&outpoint1, necm.cacheCoins[outpoint1])
	necm.AddCoin(&outpoint2, necm.cacheCoins[outpoint2])

	if necm.SpendCoin(&outpoint1) != c {
		t.Error("spend coin failed, please check...")
	}

	//ok := necm.Flush(*hash1)
	//if ok {
	//	fmt.Println()
	//	fmt.Printf("flushing=====%v\n",necm.cacheCoins[outpoint1])
	//}

	necm.UnCache(&outpoint1)
	if necm.SpendCoin(&outpoint1) != nil {
		t.Error("the uncache func is failed.")
	}

	if necm.GetCoin(&outpoint1) != nil {
		t.Error("after spend coin, the coin is empty, please check...")
	}
}
