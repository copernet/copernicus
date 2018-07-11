package utxo

import (
	"testing"

	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/opcodes"
	"bytes"
	"github.com/davecgh/go-spew/spew"
	"reflect"
	"math"
	"fmt"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
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

	if ga != txout1.GetValue() {
		t.Error("get amount value is error, please check..")
	}

	if gto != *txout1 || gh != 10 || c.isCoinBase != false {
		t.Error("get value is faild...")
	}

	if c.isCoinBase != false {
		t.Error("the coin is coinbase , please check coin ")
	}

	exceptScript := c.GetScriptPubKey()

	if !reflect.DeepEqual(exceptScript, script1) {
		t.Error("get script pubkey is not equal script1, please check...")
	}

	if c.DynamicMemoryUsage() > 0 {
		t.Error("DynamicMemoryUsage not need test...")
	}

	if !reflect.DeepEqual(c.DeepCopy(), c) {
		t.Error("after deep copy, the value should equal coin")
	}

	c.Clear()
	if c.GetHeight() == 0 && c.GetAmount() == 0 {
		t.Error("there is one error in clear func...")
	}

	if c.isCoinBase != false && c.isMempoolCoin != false {
		t.Error("isCoinBase and isMempoolCoin value should false")
	}

	if c.GetScriptPubKey() != nil {
		t.Error("the script pubkey should nil")
	}

	script2 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout2 := txout.NewTxOut(3, script2)
	c2 := NewCoin(txout2, 10, false)

	if c2.GetTxOut() != *txout2 || c2.GetHeight() != 10 {
		t.Error("get coin value is failed, please check..")
	}

	if c2.GetAmount() != txout2.GetValue() {
		t.Error("get amount failed, please check...")
	}

	if c2.isCoinBase != false {
		t.Error("the tx not is coinbase, please check...")
	}

	if !reflect.DeepEqual(c2.GetScriptPubKey(), script2) {
		t.Error("get script error,the value should equal script2, please check..")
	}

	if !reflect.DeepEqual(c2.DeepCopy(), c2) {
		t.Error("deep copy coin should equal c2")
	}

	c2.Clear()

	if reflect.DeepEqual(c2.DeepCopy(),c2){
		t.Error("after clear, the value of deep copy coin not equal coin.")
	}

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

	spew.Dump("unserlize value is :%v \n", target)
}

// new a insecure rand creator from crypto/rand seed
func newInsecureRand() []byte {
	randByte := make([]byte, 32)
	_, err := rand.Read(randByte)
	if err != nil {
		panic("init rand number creator failed...")
	}
	return randByte
}

// GetRandHash create a random Hash(utils.Hash)
func GetRandHash() *util.Hash {
	tmpStr := hex.EncodeToString(newInsecureRand())
	return util.HashFromString(tmpStr)
}

// InsecureRandRange create a random number in [0, limit]
func InsecureRandRange(limit uint64) uint64 {
	if limit == 0 {
		fmt.Println("param 0 will be insignificant")
		return 0
	}
	r := newInsecureRand()
	return binary.LittleEndian.Uint64(r) % (limit + 1)
}

// InsecureRand32 create a random number in [0 math.MaxUint32]
func InsecureRand32() uint32 {
	r := newInsecureRand()
	return binary.LittleEndian.Uint32(r)
}

// InsecureRandBits create a random number following  specified bit count
func InsecureRandBits(bit uint8) uint64 {
	r := newInsecureRand()
	maxNum := uint64(((1<<(bit-1))-1)*2 + 1 + 1)
	return binary.LittleEndian.Uint64(r) % maxNum
}

// InsecureRandBool create true or false randomly
func InsecureRandBool() bool {
	r := newInsecureRand()
	remainder := binary.LittleEndian.Uint16(r) % 2
	return remainder == 1
}

func TestRandomFunction(t *testing.T) {
	trueCount := 0
	falseCount := 0

	for i := 0; i < 10000; i++ {
		NumUint64 := InsecureRandRange(100)
		if NumUint64 > 100 {
			t.Error("InsecureRandRange() create a random number bigger than 10000")
		}

		NumUint32 := InsecureRand32()
		if NumUint32 > math.MaxUint32 {
			t.Error("InsecureRand32() creates a random number bigger than math.MaxUint32")
		}

		NumFromRandBit := InsecureRandBits(6)
		if NumFromRandBit > (((1<<(6-1))-1)*2 + 1) {
			t.Error("InsecureRandBits() creates a random numner bigger than bit-specific MaxNumber")
		}

		BoolFromRandFunc := InsecureRandBool()
		if BoolFromRandFunc {
			trueCount++
		} else {
			falseCount++
		}
	}

	if trueCount == 0 || falseCount == 0 {
		t.Error("InsecureRandBool() maybe needed to check")
	}
}
