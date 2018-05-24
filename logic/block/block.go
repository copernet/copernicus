package block

import (
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


func WriteBlockToDisk(bl *block.Block, bi *blockindex.BlockIndex, pos *block.DiskBlockPos) bool {
	
	if pos == nil{
		height := bi.Height
		pos = block.NewDiskBlockPos(0,0)
		flag := disk.FindBlockPos(pos, bl.SerializeSize(), height, uint64(bl.GetBlockHeader().Time), false)
		if !flag{
			log.Error("WriteBlockToDisk():FindBlockPos failed")
			return false
		}
	}
	err := disk.WriteBlockToDisk(bl, pos)
	if !err{
		log.Error("WriteBlockToDisk():WriteBlockToDisk failed")
		return false
	}
	return true
}