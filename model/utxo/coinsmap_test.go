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

	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	coinScript := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout2 := txout.NewTxOut(3, coinScript)

	coin1 := &Coin{
		txOut:         *txout2,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	coin1Spent := &Coin{
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

	cByAccess := necm.AccessCoin(&outpoint1)
	if !reflect.DeepEqual(c, cByAccess) {
		t.Error("coins art got by GetCoin and AccessCoin should be equal")
	}

	outpointNotExist := outpoint.OutPoint{Hash: *hash1, Index: 1}
	emptyCoin := necm.AccessCoin(&outpointNotExist)
	if !reflect.DeepEqual(emptyCoin, NewEmptyCoin()) {
		t.Error("empty Coin that is got by AccessCoin should deeply equal to which return by NewEmptyCoin")
	}

	coinsMap := necm.GetMap()
	if len(coinsMap) == 0 {
		t.Error("GetMap invoked failed")
	}

	cc := necm.FetchCoin(&outpoint1)
	if cc == nil {
		t.Error("the coin should in coinsMap")
	}
	if !reflect.DeepEqual(cc, coin1) {
		t.Error("fetch coin failed.")
	}

	if reflect.DeepEqual(necm.SpendCoin(&outpoint1), coin1Spent) {
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

	necm.AddCoin(&outpoint1, coin1, false)
	necm.UnCache(&outpoint1)
	ncc := necm.GetCoin(&outpoint1)
	if ncc != nil {
		t.Error("get coin should nil, because the coin has been uncache ")
	}

	DisplayCoinMap(necm)
}
