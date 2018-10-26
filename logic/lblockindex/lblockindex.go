package lblockindex

import (
	"math/big"
	"sort"
	"time"

	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/util"
	"gopkg.in/fatih/set.v0"
)

//LoadBlockIndexDB on main init call it
func LoadBlockIndexDB() bool {
	gChain := chain.GetInstance()
	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)

	// Load blockindex from DB
	if !blkdb.GetInstance().LoadBlockIndexGuts(GlobalBlockIndexMap, gChain.GetParams()) {
		return false
	}
	gPersist := persist.GetInstance()
	sortedByHeight := make([]*blockindex.BlockIndex, 0, len(GlobalBlockIndexMap))
	for _, index := range GlobalBlockIndexMap {
		sortedByHeight = append(sortedByHeight, index)
	}
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
		if index.TxCount <= 0 {
			log.Error("index's Txcount is 0 ")
			panic("index's Txcount is 0 ")
		}
		if index.Prev != nil {
			if index.Prev.ChainTxCount != 0 {
				index.ChainTxCount = index.Prev.ChainTxCount + index.TxCount
				branch = append(branch, index)
			} else {
				index.ChainTxCount = 0
				gChain.AddToOrphan(index)
			}
		} else {
			// genesis block
			index.ChainTxCount = index.TxCount
			branch = append(branch, index)
		}
	}
	log.Debug("LoadBlockIndexDB, BlockIndexMap len:%d, Branch len:%d, Orphan len:%d",
		len(GlobalBlockIndexMap), len(branch), gChain.ChainOrphanLen())

	// Load block file info
	btd := blkdb.GetInstance()
	var err error
	var bfi *block.BlockFileInfo

	globalLastBlockFile, err := btd.ReadLastBlockFile()
	globalBlockFileInfo := make([]*block.BlockFileInfo, 0, gPersist.GlobalLastBlockFile+1)
	if err != nil {
		log.Debug("ReadLastBlockFile() from DB err:%#v", err)
	} else {
		var nFile int32
		for ; nFile <= globalLastBlockFile; nFile++ {
			bfi, err = btd.ReadBlockFileInfo(nFile)
			if err != nil || bfi == nil {
				log.Error("ReadBlockFileInfo(%d) from DB err:%#v", nFile, err)
				panic("ReadBlockFileInfo err")
			}
			globalBlockFileInfo = append(globalBlockFileInfo, bfi)
		}
		for nFile = globalLastBlockFile + 1; true; nFile++ {
			bfi, err = btd.ReadBlockFileInfo(nFile)
			if bfi != nil && err == nil {
				log.Debug("LoadBlockIndexDB: the last block file info: %d is less than real block file info: %d",
					globalLastBlockFile, nFile)
				globalBlockFileInfo = append(globalBlockFileInfo, bfi)
				globalLastBlockFile = nFile
			} else {
				break
			}
		}
	}
	gPersist.GlobalBlockFileInfo = globalBlockFileInfo
	gPersist.GlobalLastBlockFile = globalLastBlockFile
	log.Debug("LoadBlockIndexDB: Read last block file info: %d, block file info len:%d",
		globalLastBlockFile, len(globalBlockFileInfo))

	// Check presence of block index files
	setBlkDataFiles := set.New()
	log.Debug("Checking all blk files are present...")
	for _, item := range GlobalBlockIndexMap {
		index := item
		if index.HasData() {
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
			log.Debug("Check block file: %d error, please delete blocks and chainstate and run again", pos.File)
			panic("LoadBlockIndexDB: check block file err")
		}
		file.Close()
	}

	// Build chain's active
	gChain.InitLoad(GlobalBlockIndexMap, branch)
	bestHash, err := utxo.GetUtxoCacheInstance().GetBestBlock()
	log.Debug("find bestblock hash:%s and err:%v from utxo", bestHash, err)
	if err == nil {
		tip, ok := GlobalBlockIndexMap[bestHash]
		if !ok {
			//shoud reindex from db
			log.Debug("can't find beskblock from blockindex db, please delete blocks and chainstate and run again")
			panic("can't find tip from blockindex db")
		}
		// init active chain by tip[load from db]
		gChain.SetTip(tip)
		log.Debug("LoadBlockIndexDB(): hashBestChain=%s height=%d date=%s, tiphash:%s\n",
			gChain.Tip().GetBlockHash(), gChain.Height(),
			time.Unix(int64(gChain.Tip().GetBlockTime()), 0).Format("2006-01-02 15:04:05"),
			gChain.Tip().GetBlockHash())
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
