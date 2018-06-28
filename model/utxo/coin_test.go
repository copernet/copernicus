package utxo

import (
	"testing"

	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
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
