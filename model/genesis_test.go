package model

import (
	"github.com/copernet/copernicus/util"
	"testing"
)

func TestGenesis(t *testing.T) {
	tempGenesisHash, _ := util.GetHashFromStr("000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f")
	if !GenesisBlockHash.IsEqual(tempGenesisHash) {
		t.Error("GensisBlockHash error")
		return
	}
	tempGenesisMerkleRoot, _ := util.GetHashFromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b")
	if !GenesisBlock.Header.MerkleRoot.IsEqual(tempGenesisMerkleRoot) {
		t.Error("GensisBlockMerkleRoot error")
		return
	}

	testGenesisHash, _ := util.GetHashFromStr("000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943")
	if !TestNetGenesisHash.IsEqual(testGenesisHash) {
		t.Error("TestNetGensisBlockHash error")
		return
	}
	testGenesisMerkleRoot, _ := util.GetHashFromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b")
	if !TestNetGenesisBlock.Header.MerkleRoot.IsEqual(testGenesisMerkleRoot) {
		t.Error("TestNetGensisBlockHash error")
		return
	}

	regTestGenesisHash, _ := util.GetHashFromStr("0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206")
	if !RegTestGenesisHash.IsEqual(regTestGenesisHash) {
		t.Error("RegTestGensisBlockHash error")
		return
	}
	regTestGenesisMerkleRoot, _ := util.GetHashFromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b")
	if !RegTestGenesisBlock.Header.MerkleRoot.IsEqual(regTestGenesisMerkleRoot) {
		t.Error("RegTestGensisBlockHash error")
		return
	}
}
