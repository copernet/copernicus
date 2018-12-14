package persist

import (
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/util"
	"reflect"
	"testing"
)

func createBlkIdx() *blockindex.BlockIndex {
	blkHeader := block.NewBlockHeader()
	blkHeader.Time = uint32(1534822771)
	blkHeader.Version = 536870912
	blkHeader.Bits = 486604799
	preHash := util.HashFromString("00000000000001bcd6b635a1249dfbe76c0d001592a7219a36cd9bbd002c7238")
	merkleRoot := util.HashFromString("7e814211a7de289a490380c0c20353e0fd4e62bf55a05b38e1628e0ea0b4fd3d")
	blkHeader.HashPrevBlock = *preHash
	blkHeader.Nonce = 1391785674
	blkHeader.MerkleRoot = *merkleRoot
	//init block index
	blkidx := blockindex.NewBlockIndex(blkHeader)
	return blkidx
}

func TestGetInstance(t *testing.T) {
	InitPersistGlobal(blkdb.GetInstance())
	GetInstance()
	prstGloal := new(PersistGlobal)
	prstGloal.GlobalBlockFileInfo = make([]*block.BlockFileInfo, 0, 1000)
	prstGloal.GlobalDirtyFileInfo = make(map[int32]bool)
	prstGloal.GlobalDirtyBlockIndex = make(map[util.Hash]*blockindex.BlockIndex)
	prstGloal.GlobalMapBlocksUnlinked = make(map[*blockindex.BlockIndex][]*blockindex.BlockIndex)
	if !reflect.DeepEqual(prstGloal, persistGlobal) {
		t.Error("the global variable should eaual.")
	}

	for i := 0; i < 10; i++ {
		prstGloal.AddBlockSequenceID()
		if prstGloal.GlobalBlockSequenceID != int32(i+1) {
			t.Errorf("the GlobalBlockSequenceID:%d should equal i", prstGloal.GlobalBlockSequenceID)
		}
	}

	blkidx := createBlkIdx()
	prstGloal.AddDirtyBlockIndex(blkidx)
	mapDirtyBlkIdx := prstGloal.GlobalDirtyBlockIndex[*blkidx.GetBlockHash()]
	if !reflect.DeepEqual(blkidx, mapDirtyBlkIdx) {
		t.Errorf("the GlobalDirtyBlockIndex value should equal.")
	}
}

func TestInitPruneState(t *testing.T) {
	initps := InitPruneState()
	ps := &PruneState{
		PruneMode:       false,
		HavePruned:      false,
		CheckForPruning: false,
		PruneTarget:     0,
	}
	if !reflect.DeepEqual(initps, ps) {
		t.Errorf("the prune state should equal")
	}
}
