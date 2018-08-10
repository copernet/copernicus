package disk

import (
	"time"
	"testing"

	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/chainparams"
	"github.com/copernet/copernicus/model/blockindex"
	"reflect"
	"github.com/davecgh/go-spew/spew"
)

func TestWRBlockToDisk(t *testing.T) {
	//init block header
	blkHeader := block.NewBlockHeader()
	blkHeader.Time = uint32(time.Now().Unix())
	blkHeader.Hash = blkHeader.GetHash()
	blkHeader.Version = 0
	blkHeader.Bits = 0
	preHash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d011")
	hash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d012")
	merkleRoot := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d013")
	blkHeader.HashPrevBlock = *preHash
	blkHeader.Hash = *hash
	blkHeader.Nonce = 0
	blkHeader.MerkleRoot = *merkleRoot

	//init block
	blk := block.NewBlock()
	blk.Header = *blkHeader
	blk.Checked = false
	pos := block.NewDiskBlockPos(10, 9)
	ret := WriteBlockToDisk(blk, pos)
	if !ret {
		t.Error("write block to disk failed, please check.")
	}

	blkIndex := blockindex.NewBlockIndex(blkHeader)
	blkIndex.File = 10
	blkIndex.DataPos = 9
	aa := blkIndex.GetBlockPos()
	spew.Dump(aa)
	blks, ok := ReadBlockFromDisk(blkIndex, &chainparams.TestNetParams)
	if !reflect.DeepEqual(blks, blk) && !ok {
		t.Errorf("the blks should equal blk\nblks:%v\nblk:%v", blks, blk)
	}
}
