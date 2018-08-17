package chain

import (
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"

	"testing"
	"fmt"
)

var testchain *Chain


/*

func TestGetBlockScriptFlags(t *testing.T) {

	InitGlobalChain(conf.Cfg)
	testchain = GetInstance()


	testblockheader := block.NewBlockHeader()
	testblockheader.Time = 1333238401
	testblockindex := blockindex.NewBlockIndex(testblockheader)
	testblockindex.Height = 1155877 //581885 //330776

	testchain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	testchain.AddToIndexMap(testblockindex)

	flag := testchain.GetBlockScriptFlags(testblockindex)

	if flag != 66055 {
		t.Error("sth wrong with:")
	}

	switch flag {
	case 66054: t.Error("bip16 switch")
	case 66051: t.Error("bip66 switch")
	case 65543: t.Error("bip65 switch")
	case 517 :t.Error("UAHF")
	}

	var testblheader [11]*block.BlockHeader
	var testblindex [11]*blockindex.BlockIndex
	for i:=0 ;i<11 ;i++ {

		testblheader[i] = block.NewBlockHeader()
		testblheader[i].Time = 1510600000
		testblindex[i] = blockindex.NewBlockIndex(testblheader[i])
		testblindex[i].Height = int32(1155875 - i)
		testchain.AddToIndexMap(testblindex[i])
	}

	for i:=0;i<10;i++ {
		 testblindex[i].Prev = testblindex[i+1]
	}

	testblockindex.Prev = testblindex[0]



	flag = testchain.GetBlockScriptFlags(testblockindex)
	fmt.Println(flag)

}

*/
func TestFindHashInActive(t *testing.T) {

	InitGlobalChain(conf.Cfg)
	testchain = GetInstance()
	testchain.active = make([]*blockindex.BlockIndex,2000000)

	testblockheader := block.NewBlockHeader()
	testblockheader.Time = 1333238401
	testblockindex := blockindex.NewBlockIndex(testblockheader)
	testblockindex.Height = 1155877 //581885 //330776

	testchain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	//testchain.AddToIndexMap(testblockindex)

	var testblheader [11]*block.BlockHeader
	var testblindex [11]*blockindex.BlockIndex
	for i:=0 ;i<11 ;i++ {

		testblheader[i] = block.NewBlockHeader()
		testblheader[i].Time = 1510600000
		testblindex[i] = blockindex.NewBlockIndex(testblheader[i])
		testblindex[i].Height = int32(1155877 - i)

	 	testchain.AddToIndexMap(testblindex[i])
		testchain.active[testblindex[i].Height] = testblindex[i]
	}

	testchain.active[testblockindex.Height] = testblockindex

	for i:=0;i<10;i++ {
		testblindex[i].Prev = testblindex[i+1]
	}

	testblockindex.Prev = testblindex[0]

	ans := testchain.FindHashInActive(*testblindex[3].GetBlockHash()).Height

	if testblindex[3].GetBlockHash() != testblindex[2].GetBlockHash() {fmt.Println(ans)}




}
















