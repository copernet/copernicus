package utxo

import (
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"reflect"
	"testing"
)

func TestCoinsMap(t *testing.T) {
	necm := NewEmptyCoinsMap()

	//if len(necm.cacheCoins) != 0 || necm.hashBlock != util.HashZero {
	//	t.Error("init empty coin map failed.")
	//}

	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	script2 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout2 := txout.NewTxOut(3, script2)

	coin1 := &Coin{
		txOut:         *txout2,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         false,
	}

	necm.AddCoin(&outpoint1, coin1, false)

	c := necm.GetCoin(&outpoint1)

	if c == nil {
		t.Error("the coin is nil")
	}

	if !reflect.DeepEqual(c, coin1) {
		t.Error("get coin failed.")
	}

	cc := necm.FetchCoin(&outpoint1)
	if cc == nil {
		t.Error("the coin is nil")
	}
	if !reflect.DeepEqual(cc, coin1) {
		t.Error("fetch coin failed.")
	}

	if !reflect.DeepEqual(necm.SpendCoin(&outpoint1), coin1) {
		t.Error("spend coin failed, please check...")
	}

	ccc := necm.GetCoin(&outpoint1)
	if ccc != nil {
		t.Error("get coin should nil, because the coin has been spend ")
	}

	necm.AddCoin(&outpoint1, coin1, false)
	if !reflect.DeepEqual(necm.SpendGlobalCoin(&outpoint1), coin1) {
		t.Error("spend coin should equal coin1, please check.")
	}

	nc := necm.GetCoin(&outpoint1)
	if nc != nil {
		t.Error("get coin should nil, because the coin has been spend ")
	}

	necm.AddCoin(&outpoint1, coin1, false)
	necm.UnCache(&outpoint1)
	ncc := necm.GetCoin(&outpoint1)
	if ncc != nil {
		t.Error("get coin should nil, because the coin has been uncache ")
	}

	hash2 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a7")
	outpoint2 := outpoint.OutPoint{Hash: *hash2, Index: 0}

	script1 := script.NewEmptyScript()
	txout1 := txout.NewTxOut(2, script1)

	coin2 := &Coin{
		txOut:         *txout1,
		height:        100012,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         true,
		fresh:         false,
	}

	necm.AddCoin(&outpoint2, coin2, false)

	c2 := necm.GetCoin(&outpoint2)

	if c2 == nil {
		t.Error("the coin is nil")
	}

	if !reflect.DeepEqual(c2, coin2) {
		t.Error("get coin failed.")
	}

	cc2 := necm.FetchCoin(&outpoint2)
	if cc2 == nil {
		t.Error("the coin is nil")
	}
	if !reflect.DeepEqual(cc2, coin2) {
		t.Error("fetch coin failed.")
	}

	if !reflect.DeepEqual(necm.SpendCoin(&outpoint2), coin2) {
		t.Error("spend coin failed, please check...")
	}

	ccc2 := necm.GetCoin(&outpoint2)
	if ccc2 != nil {
		t.Error("get coin should nil, because the coin has been spend ")
	}

	necm.AddCoin(&outpoint2, coin2, false)
	if !reflect.DeepEqual(necm.SpendGlobalCoin(&outpoint2), coin2) {
		t.Error("spend coin should equal coin1, please check.")
	}

	nc2 := necm.GetCoin(&outpoint2)
	if nc2 != nil {
		t.Error("get coin should nil, because the coin has been spend ")
	}

	necm.AddCoin(&outpoint2, coin2, false)
	necm.UnCache(&outpoint2)
	ncc2 := necm.GetCoin(&outpoint2)
	if ncc2 != nil {
		t.Error("get coin should nil, because the coin has been uncache ")
	}
}
