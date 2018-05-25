package block

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/persist/disk"
	"github.com/btcboost/copernicus/util"
)

func Check(b * block.Block) error {
	return nil
}

func GetBlock(hash *util.Hash) (* block.Block, error) {
	return nil,nil
}


func WriteBlockToDisk(bi *blockindex.BlockIndex, bl *block.Block)(*block.DiskBlockPos,error) {
	
	height := bi.Height
	pos := block.NewDiskBlockPos(0, 0)
	flag := disk.FindBlockPos(pos, bl.SerializeSize(), height, uint64(bl.GetBlockHeader().Time), false)
	if !flag {
		log.Error("WriteBlockToDisk():FindBlockPos failed")
		return nil, errcode.ProjectError{Code: 2000}
	}
	
	flag = disk.WriteBlockToDisk(bl, pos)
	if !flag {
		log.Error("WriteBlockToDisk():WriteBlockToDisk failed")
		return nil, errcode.ProjectError{Code: 2001}
	}
	return pos, nil
}

func WriteToFile(bi *blockindex.BlockIndex, b *block.Block) error {
	return nil
}
