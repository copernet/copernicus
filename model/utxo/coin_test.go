package utxo

import (
	"testing"

	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/script"
	"fmt"
	"github.com/copernet/copernicus/model/opcodes"
	"bytes"
	"github.com/davecgh/go-spew/spew"
)

// test whether get the expected item by OutPoint struct with a pointer
// in it or not
func TestGetCoinByPointerOrValue(t *testing.T) {
	type OutPoint struct {
		Hash  *util.Hash
		Index int
	}

	map1 := make(map[outpoint.OutPoint]*Coin)
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}
	// store one item
	map1[outpoint1] = &Coin{}
	hash11 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	if _, ok := map1[outpoint.OutPoint{Hash: *hash11, Index: 0}]; !ok {
		t.Error("the key without pointer should point to a exist value")
	}

	map2 := make(map[OutPoint]*Coin)
	hash2 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint2 := OutPoint{Hash: hash2, Index: 0}
	//store one item
	map2[outpoint2] = &Coin{}
	hash22 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	if _, ok := map2[OutPoint{Hash: hash22, Index: 0}]; ok {
		t.Error("there should not be a item as the different pointer value in the struct")
	}
}

func TestCoin(t *testing.T) {
	script1 := script.NewEmptyScript()
	txout1 := txout.NewTxOut(2, script1)
	c := NewCoin(txout1, 10, false)
	gto := c.GetTxOut()
	gh := c.GetHeight()
	ga := c.GetAmount()
	fmt.Printf("txout value is :%v\nheight is: %v\namount is: %v\n", gto, gh, ga)
	coinbase := c.isCoinBase
	fmt.Printf("whether the tx is coinbase: %v\n", coinbase)
	gsp := c.GetScriptPubKey()
	fmt.Printf("the script pub key of tx: %v \n", gsp)
	dmu := c.DynamicMemoryUsage()
	fmt.Printf("dmu is : %v\n", dmu)

	c.Clear()
	fmt.Println("==========after clear=============")
	cgto := c.GetTxOut()
	cgh := c.GetHeight()
	cga := c.GetAmount()
	fmt.Printf("txout value is :%v\nheight is: %v\namount is: %v\n", cgto, cgh, cga)
	ccoinbase := c.isCoinBase
	cmemcoin := c.isMempoolCoin
	fmt.Printf("whether the tx is coinbase: %v, is mempool coin:%v\n", ccoinbase, cmemcoin)
	cgsp := c.GetScriptPubKey()
	fmt.Printf("the script pub key of tx: %v \n", cgsp)
	dc := c.DeepCopy()
	fmt.Printf("deep copy coin : %v \n", dc)

	fmt.Println("============test script2 case===========")
	script2 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout2 := txout.NewTxOut(3, script2)
	c2 := NewCoin(txout2, 10, false)
	gto2 := c2.GetTxOut()
	gh2 := c2.GetHeight()
	ga2 := c2.GetAmount()
	fmt.Printf("txout value is :%v\nheight is: %v\namount is: %v\n", gto2, gh2, ga2)
	coinbase2 := c2.isCoinBase
	fmt.Printf("whether the tx is coinbase: %v\n", coinbase2)
	gsp2 := c2.GetScriptPubKey()
	fmt.Printf("the script pub key of tx: %v \n", gsp2)
	dc2 := c2.DeepCopy()
	fmt.Printf("deep copy coin : %v \n", dc2)

	c2.Clear()
	fmt.Println("==========after clear=============")
	cgto2 := c2.GetTxOut()
	cgh2 := c2.GetHeight()
	cga2 := c2.GetAmount()
	fmt.Printf("txout value is :%v\nheight is: %v\namount is: %v\n", cgto2, cgh2, cga2)
	ccoinbase2 := c2.isCoinBase
	cmemcoin2 := c.isMempoolCoin
	fmt.Printf("whether the tx is coinbase: %v, is mempool coin:%v\n", ccoinbase2, cmemcoin2)
	cgsp2 := c2.GetScriptPubKey()
	fmt.Printf("the script pub key of tx: %v \n", cgsp2)
	fmt.Println("=========test mempool coin========")
	nmc := NewMempoolCoin(txout2)
	mc := nmc.isMempoolCoin
	fmt.Printf("whether the coin is mempool coin: %v \n", mc)
	dc22 := c2.DeepCopy()
	fmt.Printf("deep copy coin : %v \n", dc22)
}

func TestCoinSec(t *testing.T) {
	script3 := script.NewScriptRaw([]byte{opcodes.OP_2DROP, opcodes.OP_2MUL})
	txout3 := txout.NewTxOut(4, script3)

	c3 := NewCoin(txout3, 1000, true)
	spew.Dump("the coin  is: %v \n ", c3)


	w := bytes.NewBuffer(nil)
	c3.Serialize(w)

	var target Coin
	err := target.Unserialize(bytes.NewReader(w.Bytes()))
	if err != nil {
		t.Errorf("unserlize failed...%v\n", err)
	}

	spew.Dump("unserlize value is :%v \n",target)
}
