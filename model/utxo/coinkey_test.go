package utxo

import (
	"testing"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/util"
	"bytes"
	"reflect"
	"github.com/davecgh/go-spew/spew"
)

func TestCoinKey(t *testing.T) {
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	op := outpoint.NewOutPoint(*hash1, 0)
	ck := NewCoinKey(op)

	w := bytes.NewBuffer(nil)
	ck.Serialize(w)

	var target CoinKey
	target.outpoint = &outpoint.OutPoint{}
	target.Unserialize(bytes.NewReader(w.Bytes()))
	if reflect.DeepEqual(ck, target) {
		t.Errorf("the target outpoint hash:%v not equal hash1:%v\n", target, hash1)
	}

	gs := ck.GetSerKey()
	spew.Dump("get ser key is : %v \n", gs)
}
