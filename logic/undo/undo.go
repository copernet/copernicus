package undo

import (
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/utxo"
	"fmt"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/undo"
	"copernicus/utils"
	"github.com/astaxie/beego/logs"
	"copernicus/log"
	"github.com/btcboost/copernicus/util"
	"bytes"
	"copernicus/container"
	"github.com/btcboost/copernicus/model/chain"

	"github.com/btcboost/copernicus/model/consensus"
	"sync/atomic"
	"time"
	"copernicus/core"
	"copernicus/net/msg"
	"gopkg.in/fatih/set.v0"
	"copernicus/policy"
	"math"
	"github.com/btcboost/copernicus/persist/disk"
)



var GRequestShutdown   = new(atomic.Value)
func StartShutdown() {
	GRequestShutdown.Store(true)
}

func AbortNodes(reason, userMessage string) bool {
	logs.Info("*** %s\n", reason)

	//todo:
	if len(userMessage) == 0 {
		panic("Error: A fatal internal error occurred, see debug.log for details")
	} else {

	}
	StartShutdown()
	return false
}
func AbortNode(state *block.ValidationState, reason, userMessage string) bool {
	AbortNodes(reason, userMessage)
	return state.Error(reason)
}


func FlushStateToDisk(state *block.ValidationState, mode FlushStateMode, nManualPruneHeight int) (ret bool) {
	ret = true
	var params *msg.BitcoinParams

	mempoolUsage := GMemPool.GetCacheUsage()

	// TODO: LOCK2(cs_main, cs_LastBlockFile);
	// var sc sync.RWMutex
	// sc.Lock()
	// defer sc.Unlock()

	var setFilesToPrune *set.Set
	fFlushForPrune := false

	defer func() {
		if r := recover(); r != nil {
			ret = AbortNode(state, "System error while flushing:", "")
		}
	}()
	if GPruneMode && (GCheckForPruning || nManualPruneHeight > 0) && !GfReindex {
		FindFilesToPruneManual(setFilesToPrune, nManualPruneHeight)
	} else {
		FindFilesToPrune(setFilesToPrune, uint64(params.PruneAfterHeight))
		GCheckForPruning = false
	}
	if !setFilesToPrune.IsEmpty() {
		fFlushForPrune = true
		if !GHavePruned {
			// TODO: pblocktree.WriteFlag("prunedblockfiles", true)
			GHavePruned = true
		}
	}
	nNow := utils.GetMockTimeInMicros()
	// Avoid writing/flushing immediately after startup.
	if gLastWrite == 0 {
		gLastWrite = int(nNow)
	}
	if gLastFlush == 0 {
		gLastFlush = int(nNow)
	}
	if gLastSetChain == 0 {
		gLastSetChain = int(nNow)
	}
	nMempoolSizeMax := utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize)) * 1000000
	cacheSize := GCoinsTip.DynamicMemoryUsage() * DBPeakUsageFactor
	nTotalSpace := float64(GnCoinCacheUsage) + math.Max(float64(nMempoolSizeMax-mempoolUsage), 0)
	// The cache is large and we're within 10% and 200 MiB or 50% and 50MiB
	// of the limit, but we have time now (not in the middle of a block processing).
	x := math.Max(nTotalSpace/2, nTotalSpace-MinBlockCoinsDBUsage*1024*1024)
	y := math.Max(9*nTotalSpace/10, nTotalSpace-MaxBlockCoinsDBUsage*1024*1024)
	fCacheLarge := mode == FlushStatePeriodic && float64(cacheSize) > math.Min(x, y)
	// The cache is over the limit, we have to write now.
	fCacheCritical := mode == FlushStateIfNeeded && float64(cacheSize) > nTotalSpace
	// It's been a while since we wrote the block index to disk. Do this
	// frequently, so we don't need to redownLoad after a crash.
	fPeriodicWrite := mode == FlushStatePeriodic && int(nNow) > gLastWrite+DataBaseWriteInterval*1000000
	// It's been very long since we flushed the cache. Do this infrequently,
	// to optimize cache usage.
	fPeriodicFlush := mode == FlushStatePeriodic && int(nNow) > gLastFlush+DataBaseFlushInterval*1000000
	// Combine all conditions that result in a full cache flush.
	fDoFullFlush := mode == FlushStateAlways || fCacheLarge || fCacheCritical || fPeriodicFlush || fFlushForPrune
	// Write blocks and block index to disk.
	if fDoFullFlush || fPeriodicWrite {
		// Depend on nMinDiskSpace to ensure we can write block index
		if !CheckDiskSpace(0) {
			ret = state.Error("out of disk space")
		}
		// First make sure all block and undo data is flushed to disk.
		FlushBlockFile(false)
		// Then update all block file information (which may refer to block and undo files).

		type Files struct {
			key   []int
			value []*BlockFileInfo
		}

		files := Files{
			key:   make([]int, 0),
			value: make([]*BlockFileInfo, 0),
		}

		lists := gSetDirtyFileInfo.List()
		for _, value := range lists {
			v := value.(int)
			files.key = append(files.key, v)
			files.value = append(files.value, gInfoBlockFile[v])
			gSetDirtyFileInfo.RemoveItem(v)
		}

		var blocks = make([]*core.BlockIndex, 0)
		list := gSetDirtyBlockIndex.List()
		for _, value := range list {
			v := value.(*core.BlockIndex)
			blocks = append(blocks, v)
			gSetDirtyBlockIndex.RemoveItem(value)
		}

		//err := GBlockTree.WriteBatchSync(files, gLastBlockFile, blocks)
		//if err != nil {
		//	ret = AbortNode(state, "Failed to write to block index database", "")
		//}

		// Finally remove any pruned files
		if fFlushForPrune {
			UnlinkPrunedFiles(setFilesToPrune)
		}
		gLastWrite = int(nNow)

	}

	// Flush best chain related state. This can only be done if the blocks /
	// block index write was also done.
	if fDoFullFlush {
		// Typical Coin structures on disk are around 48 bytes in size.
		// Pushing a new one to the database can cause it to be written
		// twice (once in the log, and once in the tables). This is already
		// an overestimation, as most will delete an existing entry or
		// overwrite one. Still, use a conservative safety factor of 2.
		if !CheckDiskSpace(uint32(48 * 2 * 2 * GCoinsTip.GetCacheSize())) {
			ret = state.Error("out of disk space")
		}
		// Flush the chainState (which may refer to block index entries).
		if !GCoinsTip.Flush() {
			ret = AbortNode(state, "Failed to write to coin database", "")
		}
		gLastFlush = int(nNow)
	}
	if fDoFullFlush || ((mode == FlushStateAlways || mode == FlushStatePeriodic) &&
		int(nNow) > gLastSetChain+DataBaseWriteInterval*1000000) {
		// Update best block in wallet (so we can detect restored wallets).
		// TODO:GetMainSignals().SetBestChain(chainActive.GetLocator())
		gLastSetChain = int(nNow)
	}

	return
}

type FlushStateMode int

const (
	FlushStateNone FlushStateMode = iota
	FlushStateIfNeeded
	FlushStatePeriodic
	FlushStateAlways
)

// DisconnectTip Disconnect chainActive's tip. You probably want to call
// mempool.removeForReorg and manually re-limit mempool size after this, with
// cs_main held.
func DisconnectTip(param *consensus.BitcoinParams, state *block.ValidationState, fBare bool) bool {

	indexDelete := chain.GetInstance().Tip()
	if indexDelete == nil {
		panic("the chain tip element should not equal nil")
	}
	// Read block from disk.
	blk, ret := disk.ReadBlockFromDisk(indexDelete, param)
	if !ret{
		return AbortNode(state, "Failed to read block", "")
	}

	// Apply the block atomically to the chain state.
	nStart := time.Now().UnixNano()
	{
		view := utxo.NewEmptyCoinsMap()

		if DisconnectBlock(blk, indexDelete, view) != undo.DisconnectOk {
			hash := indexDelete.GetBlockHash()
			logs.Error(fmt.Sprintf("DisconnectTip(): DisconnectBlock %s failed ", hash.ToString()))
			return false
		}
		flushed := view.Flush(blk.Header.HashPrevBlock)
		if !flushed {
			panic("view flush error !!!")
		}

	}
	// replace implement with log.Print(in C++).
	log.Print("bench", "debug", " - Disconnect block : %.2fms\n",
		float64(time.Now().UnixNano()-nStart)*0.001)

	// Write the chain state to disk, if necessary.
	if !FlushStateToDisk(state, FlushStateIfNeeded, 0) {
		return false
	}

	if !fBare {
		// Resurrect mempool transactions from the disconnected block.
		vHashUpdate := container.Vector{}
		for _, tx := range block.Txs {
			// ignore validation errors in resurrected transactions
			var stateDummy block.ValidationState
			if tx.IsCoinBase() || !AcceptToMemoryPool(param, GMemPool, &stateDummy, tx,
				false, nil, nil, true, 0) {
				GMemPool.Lock()
				GMemPool.RemoveTxRecursive(tx, mempool.REORG)
				GMemPool.Unlock()
			} else if GMemPool.Exists(tx.Hash) {
				vHashUpdate.PushBack(tx.Hash)
			}
		}
		// AcceptToMemoryPool/addUnchecked all assume that new memPool entries
		// have no in-memPool children, which is generally not true when adding
		// previously-confirmed transactions back to the memPool.
		// UpdateTransactionsFromBlock finds descendants of any transactions in
		// this block that were added back and cleans up the memPool state.
		GMemPool.UpdateTransactionsFromBlock(vHashUpdate)
	}

	// Update chainActive and related variables.
	UpdateTip(param, indexDelete.Prev)
	// Let wallets know transactions went from 1-confirmed to
	// 0-confirmed or conflicted:
	for _, tx := range block.Txs {
		// todo !!! add  GetMainSignals().SyncTransaction()
		_ = tx
	}
	return true
}


func DisconnectBlock(pblock *block.Block, pindex *blockindex.BlockIndex, view *utxo.CoinsMap) undo.DisconnectResult {

	hashA := pindex.GetBlockHash()
	hashB := utxo.GetUtxoCacheInstance().GetBestBlock()
	if !bytes.Equal(hashA[:], hashB[:]) {
		panic("the two hash should be equal ...")
	}
	var blockUndo undo.BlockUndo
	pos := pindex.GetUndoPos()
	if pos.IsNull() {
		logs.Error("DisconnectBlock(): no undo data available")
		return undo.DisconnectFailed
	}

	if !UndoReadFromDisk(&blockUndo, &pos, *pindex.Prev.GetBlockHash()) {
		logs.Error("DisconnectBlock(): failure reading undo data")
		return undo.DisconnectFailed
	}

	return ApplyBlockUndo(&blockUndo, pblock, pindex, view)
}

func UndoReadFromDisk(blockundo *undo.BlockUndo, pos *block.DiskBlockPos, hashblock util.Hash) (ret bool) {
	ret = true
	defer func() {
		if err := recover(); err != nil {
			logs.Error(fmt.Sprintf("%s: Deserialize or I/O error - %v", log.TraceLog(), err))
			ret = false
		}
	}()
	file := disk.OpenUndoFile(*pos, true)
	if file == nil {
		logs.Error(fmt.Sprintf("%s: OpenUndoFile failed", log.TraceLog()))
		return false
	}

	// Read block
	var hashCheckSum utils.Hash
	ok := hashblock.Serialize(file)
	if !ok {
		return ok
	}
	blockundo, err := DeserializeBlockUndo(file)
	if err != nil {
		return false
	}
	ok = hashCheckSum.Deserialize(file)

	// Verify checksum
	// todo !!! add if bytes.Equal(hashCheckSum[:], )

	return ok
}


func ApplyBlockUndo(blockUndo *undo.BlockUndo, blk *block.Block, index *blockindex.BlockIndex,
	cache *utxo.CoinsMap) undo.DisconnectResult {
	clean := true
	txUndos := blockUndo.GetTxundo()
	if len(txUndos)+1 != len(blk.Txs) {
		fmt.Println("DisconnectBlock(): block and undo data inconsistent")
		return undo.DisconnectFailed
	}
	i := len(blk.Txs)
	// Undo transactions in reverse order.
	for  i >0 {
		i--
		tx := blk.Txs[i]
		txid := tx.Hash

		// Check that all outputs are available and match the outputs in the
		// block itself exactly.
		for j := 0; j < tx.GetOutsCount(); j++ {
			if tx.GetTxOut(j).IsSpendable() {
				continue
			}

			out := outpoint.NewOutPoint(txid, uint32(j))
			coin := cache.SpendCoin(out)
			coinOut := coin.GetTxOut()
			if coin!=nil || tx.GetTxOut(j).IsEqual(&coinOut)  {
				// transaction output mismatch
				clean = false
			}

			// Restore inputs
			if i < 1 {
				// Skip the coinbase
				break
			}

			txundo := txUndos[i-1]
			if len(txundo.PrevOut) != len(tx.GetIns()) {
				fmt.Println("DisconnectBlock(): transaction and undo data inconsistent")
				return undo.DisconnectFailed
			}
			ins := tx.GetIns()
			for k := len(ins); k > 0; {
				k--
				outpoint := ins[k].PreviousOutPoint
				c := txundo.PrevOut[k]
				res := UndoCoinSpend(c, cache, outpoint)
				if res == undo.DisconnectFailed {
					return undo.DisconnectFailed
				}
				clean = clean && (res != undo.DisconnectUnclean)
			}
		}
	}



	if clean {
		return undo.DisconnectOk
	}
	return undo.DisconnectUnclean
}


func UndoCoinSpend(coin *utxo.Coin, cm *utxo.CoinsMap, out *outpoint.OutPoint) undo.DisconnectResult {
	clean := true
	if cm.FetchCoin(out)!=nil {
		// Overwriting transaction output.
		clean = false
	}
	// delete this logic from core-abc
	//if coin.GetHeight() == 0 {
	//	// Missing undo metadata (height and coinbase). Older versions included
	//	// this information only in undo records for the last spend of a
	//	// transactions' outputs. This implies that it must be present for some
	//	// other output of the same tx.
	//	alternate := utxo.AccessByTxid(cache, &out.Hash)
	//	if alternate.IsSpent() {
	//		// Adding output for transaction without known metadata
	//		return DisconnectFailed
	//	}
	//
	//	// This is somewhat ugly, but hopefully utility is limited. This is only
	//	// useful when working from legacy on disck data. In any case, putting
	//	// the correct information in there doesn't hurt.
	//	coin = utxo.NewCoin(coin.GetTxOut(), alternate.GetHeight(), alternate.IsCoinBase())
	//}
	cm.AddCoin(out, *coin, coin.IsCoinBase())
	if clean {
		return undo.DisconnectOk
	}
	return undo.DisconnectUnclean
}
