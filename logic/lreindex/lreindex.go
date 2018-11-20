package lreindex

import (
	"container/list"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblock"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/util"
	"io"
	"os"
	"time"
)

func Reindex() (err error) {
	blkFiles, err := disk.GetBlkFiles()
	if err != nil {
		log.Error("reindex: get blk files failed, err:%s", err)
		return err
	}

	log.Info("Start reindexing")
	blkdb.GetInstance().WriteReindexing(true)
	for index, filePath := range blkFiles {
		dbp := block.NewDiskBlockPos(int32(index), uint32(0))
		_, err := loadExternalBlockFile(filePath, dbp)
		if err != nil {
			if err.Error() == io.EOF.Error() {
				log.Info("have read full data from file<%s>, next", filePath)
			} else {
				log.Error("When reindex, load block file <%s> failed!", filePath)
			}
		}
	}

	err = lchain.ActivateBestChain(nil)
	if err != nil {
		log.Error("ActivateBestChain failed after reindex")
		return err
	}

	blkdb.GetInstance().WriteReindexing(false)
	conf.Cfg.Reindex = false
	log.Info("Reindexing finished")

	return err
}

func loadExternalBlockFile(filePath string, dbp *block.DiskBlockPos) (nLoaded int, err error) {
	mapBlocksUnknownParent := make(map[util.Hash]*list.List)
	log.Info("file: %s", filePath)
	mchain := chain.GetInstance()
	params := mchain.GetParams()
	nLoaded = 0

	nStartTime := time.Now()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Error("loadExternalBlockFile: %s", err)
		return
	}

	fileSize := uint32(fileInfo.Size())
	for dbp.Pos < fileSize+1 {
		log.Info("pos: %d", dbp.Pos)
		var blk *block.Block
		blk, err = disk.ReadBlockFromDiskByPos(*dbp, params)
		if err != nil {
			if err.Error() == io.EOF.Error() {
				log.Debug("read to the end of file")
				break
			}
			log.Error("fail to read block from pos<%d, %d>", dbp.File, dbp.Pos)
			return nLoaded, errcode.New(errcode.FailedToReadBlock)
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
				mapBlocksUnknownParent[blkPreHash].PushBack(*dbp)
			}
			dbp.Pos = dbp.Pos + uint32(blk.EncodeSize()) + 4
			continue

		}

		if blkIndex := mchain.FindBlockIndex(blkHash); blkIndex == nil || !blkIndex.HasData() {
			persist.CsMain.Lock()
			fNewBlock := false
			_, _, err = lblock.AcceptBlock(blk, true, dbp, &fNewBlock)
			if err != nil {
				log.Error("accept block error: %s", err)
				break
			}

			nLoaded++
			persist.CsMain.Unlock()
			log.Debug("already accept block: %s", blk.Header.Hash.String())
			//		} else if blkIndex := mchain.FindBlockIndex(blkHash); blkHash != *params.GenesisHash && blkIndex.Height%1000 == 0 {
		} else {
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
				diskBlkPos, _ := val.(block.DiskBlockPos)

				blk, err := disk.ReadBlockFromDiskByPos(diskBlkPos, params)
				if err != nil {
					log.Error("when process successors, fail to read block from pos<%d, %d>", diskBlkPos.File, diskBlkPos.Pos)
					continue
				}
				hash := blk.GetHash()
				//hex.EncodeToString(blk.GetHash()[:])
				log.Info("Processing out of order child %s of %s", hash.String(), head.String())
				persist.CsMain.Lock()
				fNewBlock := false
				_, _, err = lblock.AcceptBlock(blk, true, &diskBlkPos, &fNewBlock)
				if err == nil {
					nLoaded++
					queue.PushBack(blk.GetHash())
				} else {
					log.Error("Error accept out of order block: %s", hash.String())

				}
				persist.CsMain.Unlock()
			}
		}

		dbp.Pos = dbp.Pos + uint32(blk.EncodeSize()) + 4
	}

	log.Info("end-pos: %d", dbp.Pos)
	log.Info("file size: %d", fileInfo.Size())
	nEndTime := time.Now()
	if nLoaded > 0 {
		log.Info("Loaded %d blocks from external file in %f seconds", nLoaded, nEndTime.Sub(nStartTime).Seconds())
	}

	return nLoaded, err
}

func UnloadBlockIndex() {
	persist.CsMain.Lock()
	defer persist.CsMain.Unlock()

	persistGlobal := persist.GetInstance()
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
	globalChain.SetIndexBestHeader(nil)

	//TODO clear mempool
}
