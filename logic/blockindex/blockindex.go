package blockindex

import (
	"copernicus/core"
	"sort"
	"math/big"
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/model/chain/global"
	"gopkg.in/fatih/set.v0"
	"time"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/persist/blkdb"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/persist/disk"
	"github.com/btcboost/copernicus/model/pow"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/model/block"
)


func LoadBlockIndexDB(params *chainparams.BitcoinParams) bool {
	if !blkdb.GetBlockTreeDBInstance().LoadBlockIndexGuts() {
		return false
	}
	
	chainGlobal := global.GetChainGlobalInstance()
	sortedByHeight := make([]*blockindex.BlockIndex, 0, len(chainGlobal.GlobalBlockIndexMap))
	for _, index := range chainGlobal.GlobalBlockIndexMap {
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
			if index.Header.Time < index.Prev.Header.Time{
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
				} else {
					index.ChainTxCount = 0
					chainGlobal.GlobalBlocksUnlinkedMap[index.Prev] = index
				}
			} else {
				index.ChainTxCount = index.TxCount
			}
		}else{
			log.Error("index's Txcount is 0 ")
			panic("index's Txcount is 0 ")
		}

		if index.Prev != nil {
			index.BuildSkip()
		}
		// todo notunderstand
		//if index.IsValid(BlockValidTransactions) &&
		//	(index.ChainTxCount != 0 || index.Prev == nil) {
		//	gSetBlockIndexCandidates.AddItem(index)
		//}
		//
		//if index.Status&BlockFailedMask != 0 &&
		//	(index.ChainWork.Cmp(&gIndexBestInvalid.ChainWork) > 0) {
		//	gIndexBestInvalid = index
		//}
		//

		//
		//if index.IsValid(BlockValidTree) &&
		//	(GIndexBestHeader == nil || BlockIndexWorkComparator(GIndexBestHeader, index)) {
		//	GIndexBestHeader = index
		//}
	}

	// Load block file info
	var err error
	chainGlobal.GlobalLastBlockFile, err = blkdb.GetBlockTreeDBInstance().ReadLastBlockFile()
	if err != nil{
		log.Error("Error: GetLastBlockFile err %#v", err)
	}
	logs.Debug("LoadBlockIndexDB(): last block file = %d", chainGlobal.GlobalLastBlockFile)
	for file := 0; file <= chainGlobal.GlobalLastBlockFile; file++ {
		chainGlobal.GlobalBlockFileInfoMap[file],err = blkdb.GetBlockTreeDBInstance().ReadBlockFileInfo(file)
	}
	logs.Debug("LoadBlockIndexDB(): last block file info: %s\n",
		chainGlobal.GlobalBlockFileInfoMap[chainGlobal.GlobalLastBlockFile].String())
	var bli *block.BlockFileInfo
	for file := chainGlobal.GlobalLastBlockFile + 1; true; file++ {
		bli, err = blkdb.GetBlockTreeDBInstance().ReadBlockFileInfo(file)
		if  bli != nil && err == nil {
			chainGlobal.GlobalBlockFileInfoMap[file] = bli
		} else {
			break
		}
	}

	// Check presence of blk files
	setBlkDataFiles := set.New()
	logs.Debug("Checking all blk files are present...")
	for _, item := range chainGlobal.GlobalBlockIndexMap {
		index := item
		if index.Status&core.BlockHaveData != 0 {
			setBlkDataFiles.Add(index.File)
		}
	}

	l := setBlkDataFiles.List()
	for _, item := range l {
		pos := &core.DiskBlockPos{
			File: item.(int),
			Pos:  0,
		}

		file := disk.OpenBlockFile(pos, true)
		if file == nil {
			return false
		}
		file.Close()
	}

	// Check whether we have ever pruned block & undo files
	chainGlobal.GlobalHavePruned = blkdb.GetBlockTreeDBInstance().ReadFlag("prunedblockfiles")
	if chainGlobal.GlobalHavePruned  {
		logs.Debug("LoadBlockIndexDB(): Block files have previously been pruned")
	}

	// Check whether we need to continue reindexing
	reIndexing := false
	reIndexing = blkdb.GetBlockTreeDBInstance().ReadReindexing()
	if reIndexing {
		chainGlobal.GlobalReindex = true
	}

	// Check whether we have a transaction index
	chainGlobal.GlobalTxIndex = blkdb.GetBlockTreeDBInstance().ReadFlag("txindex")
	if chainGlobal.GlobalTxIndex {
		logs.Debug("LoadBlockIndexDB(): transaction index enabled")
	} else {
		logs.Debug("LoadBlockIndexDB(): transaction index disabled")
	}

	// Load pointer to end of best chain
	index, ok := chainGlobal.GlobalBlockIndexMap[utxo.GetUtxoCacheInstance().GetBestBlock()]
	if !ok {
		return true
	}

	chain.GetInstance().SetTip(index)
	PruneBlockIndexCandidates()

	log.Debug("LoadBlockIndexDB(): hashBestChain=%s height=%d date=%s progress=%f\n",
		chain.GetInstance().Tip().GetBlockHash().ToString(), chain.GetInstance().Height(),
		time.Unix(int64(chain.GetInstance().Tip().GetBlockTime()), 0).Format("2006-01-02 15:04:05"),
		GuessVerificationProgress(params.TxData(), chain.GetInstance().Tip()))

	return true
}
