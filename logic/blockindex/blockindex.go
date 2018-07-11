package blockindex

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/persist/global"
	"github.com/copernet/copernicus/util"
	"gopkg.in/fatih/set.v0"
)

//on main init call it
func LoadBlockIndexDB() bool {
	gChain := chain.GetInstance()
	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)

	// branchMap := make(map[util.Hash]*blockindex.BlockIndex)
	if !blkdb.GetInstance().LoadBlockIndexGuts(GlobalBlockIndexMap, gChain.GetParams()) {
		return false
	}

	gPersist := global.GetInstance()
	sortedByHeight := make([]*blockindex.BlockIndex, 0, len(GlobalBlockIndexMap))
	for _, index := range GlobalBlockIndexMap {
		sortedByHeight = append(sortedByHeight, index)
	}
	// for idx, bi := range sortedByHeight{
	// 	if bi.TxCount == 0{
	// 		log.Error("idx",idx)
	// 		h:= bi.GetBlockHash()
	// 		_, ok := GlobalBlockIndexMap[*h]
	// 		if ok{
	//
	// 		}
	// 	}
	// }
	//sort by decrease
	sort.SliceStable(sortedByHeight, func(i, j int) bool {
		return sortedByHeight[i].Height < sortedByHeight[j].Height
	})
	for _, index := range sortedByHeight {
		timeMax := index.Header.Time
		if index.Prev != nil {
			sum := big.NewInt(0)
			sum.Add(&index.Prev.ChainWork, pow.GetBlockProof(index))
			index.ChainWork = *sum
			if index.Header.Time < index.Prev.Header.Time {
				timeMax = index.Prev.Header.Time
			}
		} else {
			index.ChainWork = *pow.GetBlockProof(index)
		}
		index.TimeMax = timeMax

		// We can link the chain of blocks for which we've received transactions
		// at some point. Pruned nodes may have deleted the block.
		if index.TxCount > 0 {
			if index.Prev != nil {
				if index.Prev.ChainTxCount != 0 {
					index.ChainTxCount = index.Prev.ChainTxCount + index.TxCount
					branch = append(branch, index)
				} else {
					index.ChainTxCount = 0
					gChain.AddToOrphan(index)
				}
			} else {
				branch = append(branch, index)

				index.ChainTxCount = index.TxCount
			}
		} else {
			log.Error("index's Txcount is 0 ")
			panic("index's Txcount is 0 ")
		}
		if index.IsValid(blockindex.StatusAllValid) &&
			(index.ChainTxCount != 0 || index.Prev == nil) {
			// gChain.AddToBranch(index)
		}
		if index.Prev != nil {
			index.BuildSkip()
		}
	}

	// Load block file info
	btd := blkdb.GetInstance()
	var err error
	var bfi *block.BlockFileInfo

	gPersist.GlobalLastBlockFile, err = btd.ReadLastBlockFile()
	if err != nil {
		log.Error("btd.ReadLastBlockFile() err:%#v", err)
	}
	GlobalBlockFileInfo := make(global.BlockFileInfoList, gPersist.GlobalLastBlockFile+1)
	if err != nil {
		log.Error("Error: GetLastBlockFile err %#v", err)
	}
	logs.Debug("LoadBlockIndexDB(): last block file = %d", gPersist.GlobalLastBlockFile)
	for nFile := int32(0); nFile <= gPersist.GlobalLastBlockFile; nFile++ {
		bfi, err = btd.ReadBlockFileInfo(nFile)
		if err == nil {
			if bfi == nil {
				bfi = block.NewBlockFileInfo()
			}
			GlobalBlockFileInfo[nFile] = bfi
		} else {
			log.Error("btd.ReadBlockFileInfo(%#v) err:%#v", nFile, err)
			panic("btd.ReadBlockFileInfo(nFile) err")
		}
	}
	logs.Debug("LoadBlockIndexDB(): last block file info: %s\n",
		GlobalBlockFileInfo[gPersist.GlobalLastBlockFile].String())
	for nFile := gPersist.GlobalLastBlockFile + 1; true; nFile++ {
		bfi, err = btd.ReadBlockFileInfo(nFile)
		if bfi != nil && err == nil {
			GlobalBlockFileInfo = append(GlobalBlockFileInfo, bfi)
		} else {
			break
		}
	}
	gPersist.GlobalBlockFileInfo = GlobalBlockFileInfo
	// Check presence of blk files
	setBlkDataFiles := set.New()
	logs.Debug("Checking all blk files are present...")
	for _, item := range GlobalBlockIndexMap {
		index := item
		if index.Status&blockindex.BlockHaveData != 0 {
			setBlkDataFiles.Add(index.File)
		}
	}

	l := setBlkDataFiles.List()

	for _, item := range l {
		pos := &block.DiskBlockPos{
			File: item.(int32),
			Pos:  0,
		}
		file := disk.OpenBlockFile(pos, true)
		if file == nil {
			return false
		}
		file.Close()
	}

	gChain.InitLoad(GlobalBlockIndexMap, branch)

	// Load pointer to end of best chain todo: coinDB must init!!!
	bestHash, err := utxo.GetUtxoCacheInstance().GetBestBlock()
	log.Debug("find bestblock hash:%v and err:%v from utxo", bestHash, err)
	if err == nil {
		tip, ok := GlobalBlockIndexMap[bestHash]
		indexMapLen := len(GlobalBlockIndexMap)
		fmt.Println("indexMapLen====", indexMapLen)
		if !ok {
			//shoud reindex from db
			log.Debug("can't find tip from blockindex db, please delete blocks and chainstate and run again")
			panic("can't find tip from blockindex db")
			//return true
		}
		// init active chain by tip[load from db]
		gChain.SetTip(tip)
		log.Debug("LoadBlockIndexDB(): hashBestChain=%s height=%d date=%s progress=%f\n",
			gChain.Tip().GetBlockHash().ToString(), gChain.Height(),
			time.Unix(int64(gChain.Tip().GetBlockTime()), 0).Format("2006-01-02 15:04:05"),
			gChain.Tip())
	}

	return true
}
func CheckIndexAgainstCheckpoint(preIndex *blockindex.BlockIndex) bool {
	gChain := chain.GetInstance()
	if preIndex.IsGenesis(gChain.GetParams()) {
		return true
	}
	nHeight := preIndex.Height + 1
	// Don't accept any forks from the main chain prior to last checkpoint
	params := gChain.GetParams()
	checkPoints := params.Checkpoints
	var checkPoint *model.Checkpoint
	for i := len(checkPoints) - 1; i >= 0; i-- {
		checkPointIndex := gChain.FindBlockIndex(*checkPoints[i].Hash)
		if checkPointIndex != nil {
			checkPoint = checkPoints[i]
			break
		}
	}
	if checkPoint != nil && nHeight < checkPoint.Height {
		return false
	}
	return true
}
