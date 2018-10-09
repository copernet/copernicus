package lreindex

import (
	"container/list"
	"github.com/copernet/copernicus.bak/persist/global"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblock"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/util"
	"os"
	"time"
)

func Reindex() (err error) {
	blkFiles, err := disk.GetBlkFiles()
	if err != nil {
		log.Error("reindex: get blk files failed, err:%s", err)
	}

	log.Info("Start reindexing")

	for index, filePath := range blkFiles {
		if index == 1 {
			break
		}
		dbp := block.NewDiskBlockPos(int32(index), uint32(0))
		err = loadExternalBlockFile(filePath, dbp)
	}
	blkdb.GetInstance().WriteReindexing(false)
	conf.Cfg.Reindex = false
	log.Info("Reindexing finished")

	return err
}

func loadExternalBlockFile(filePath string, dbp *block.DiskBlockPos) (err error) {
	mapBlocksUnknownParent := make(map[util.Hash]*list.List)
	log.Info("file: %s", filePath)
	mchain := chain.GetInstance()
	params := mchain.GetParams()
	nLoaded := 0

	nStartTime := time.Now()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Error("loadExternalBlockFile: %s", err)
		return
	}

	fileSize := uint32(fileInfo.Size())
	for dbp.Pos < fileSize {
		log.Info("pos: %d", dbp.Pos)
		blk, ok := disk.ReadBlockFromDiskByPos(*dbp, params)
		if ok != true {
			log.Error("fail to read block from pos<%d, %d>", dbp.File, dbp.Pos)
			return errcode.New(errcode.FailedToReadBlock)
		}

		blkHash := blk.GetHash()
		blkPreHash := blk.Header.HashPrevBlock

		if blkHash != *params.GenesisHash && nil == mchain.FindBlockIndex(blkPreHash) {
			log.Info("Out of order block %s, parent %s not known",
				blkHash.String(),
				blkPreHash.String())
			if dbp != nil {
				_, exist := mapBlocksUnknownParent[blkPreHash]
				if !exist {
					mapBlocksUnknownParent[blkPreHash] = list.New()
				}
				mapBlocksUnknownParent[blkPreHash].PushBack(dbp)
			}
			continue

		}

		if blkIndex := mchain.FindBlockIndex(blkHash); blkIndex == nil || !blkIndex.HasData() {
			global.CsMain.Lock()
			fNewBlock := false
			_, _, err = lblock.AcceptBlock(blk, true, dbp, &fNewBlock)
			if err != nil {
				break
			}

			nLoaded++
			global.CsMain.Unlock()
		} else if blkIndex := mchain.FindBlockIndex(blkHash); blkHash != *params.GenesisHash && blkIndex.Height%1000 == 0 {
			log.Info("already had block %s at height %d", blkHash.String(), blkIndex.Height)
		}

		// Activate the genesis block so normal node progress can continue
		if blkHash == *params.GenesisHash {
			err = lchain.ActivateBestChain(blk)
			if err != nil {
				log.Error("Activate the genesis block failed")
				break
			}
		}

		// Recursively process earlier encountered successors of this block

		queue := list.New()
		queue.PushBack(blkHash)
		for queue.Len() > 0 {
			val := queue.Remove(queue.Front())
			head, _ := val.(util.Hash)
			itemList, ok := mapBlocksUnknownParent[head]
			if !ok {
				continue
			}
			for itemList.Len() > 0 {
				val := itemList.Remove(itemList.Front())
				diskBlkPos, _ := val.(*block.DiskBlockPos)

				blk, ok := disk.ReadBlockFromDiskByPos(*diskBlkPos, params)
				if !ok {
					log.Error("when process successors, fail to read block from pos<%d, %d>", diskBlkPos.File, diskBlkPos.Pos)
					continue
				}
				hash := blk.GetHash()
				//hex.EncodeToString(blk.GetHash()[:])
				log.Info("Processing out of order child %s of %s", hash.String(), blkHash.String())
				global.CsMain.Lock()
				fNewBlock := false
				_, _, err = lblock.AcceptBlock(blk, true, diskBlkPos, &fNewBlock)
				if err == nil {
					nLoaded++
					queue.PushBack(blk.GetHash())
				} else {
					log.Error("Error accept out of order block: %s", hash.String())

				}
				global.CsMain.Unlock()
			}
		}

		dbp.Pos = dbp.Pos + uint32(blk.EncodeSize()) + 4
		nLoaded++
	}
	log.Info("end-pos: %d", dbp.Pos)
	log.Info("file size: %d", fileInfo.Size())

	nEndTime := time.Now()
	if nLoaded > 0 {
		log.Info("Loaded %d blocks from external file in %f seconds", nLoaded, nEndTime.Sub(nStartTime).Seconds())
	}

	return
}

func UnloadBlockIndex() {
	global.CsMain.Lock()
	defer global.CsMain.Unlock()

	persistGlobal := global.GetInstance()
	persistGlobal.GlobalBlockFileInfo = make([]*block.BlockFileInfo, 0, 1000)
	persistGlobal.GlobalDirtyFileInfo = make(map[int32]bool)
	persistGlobal.GlobalDirtyBlockIndex = make(map[util.Hash]*blockindex.BlockIndex)
	persistGlobal.GlobalMapBlocksUnlinked = make(map[*blockindex.BlockIndex][]*blockindex.BlockIndex)
	persistGlobal.GlobalLastBlockFile = 0
	persistGlobal.GlobalBlockSequenceID = 1

	indexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)
	globalChain := chain.GetInstance()
	globalChain.InitLoad(indexMap, branch)
	globalChain.ClearActive()

	//TODO clear mempool
}
