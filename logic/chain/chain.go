package chain

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/chain"
	lblock "github.com/btcboost/copernicus/logic/block"
	"fmt"

	"github.com/btcboost/copernicus/log"
	"time"
	"strings"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/persist/disk"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/logic/undo"
	mUndo "github.com/btcboost/copernicus/model/undo"

	"github.com/btcboost/copernicus/model/mempool"
	lmp "github.com/btcboost/copernicus/logic/mempool"

	"bytes"
	"copernicus/net/msg"
	"github.com/astaxie/beego/logs"
)

func AcceptBlock(b * block.Block) (*blockindex.BlockIndex,error) {


	var bIndex,err = AcceptBlockHeader(&b.Header)
	if err != nil {
		return nil,err
	}
	log.Info(bIndex)

	return nil,nil
}

func  AcceptBlockHeader(bh * block.BlockHeader) (*blockindex.BlockIndex,error) {
	var c = chain.GetInstance()

	bIndex := c.FindBlockIndex(bh.GetHash())
	if bIndex != nil {
		if bIndex.HeaderValid() == false {
			return nil,errcode.New(errcode.ErrorBlockHeaderNoValid)
		}
	} else {
		err := lblock.CheckBlockHeader(&bIndex.Header)
		if err != nil {
			return nil,err
		}

		bIndex = blockindex.NewBlockIndex(bh)
		bIndex.Prev = c.FindBlockIndex(bh.HashPrevBlock)
		if bIndex.Prev == nil {
			return nil,errcode.New(errcode.ErrorBlockHeaderNoParent)
		}
	}


	return nil,nil
}
// DisconnectTip Disconnect chainActive's tip. You probably want to call
// mempool.removeForReorg and manually re-limit mempool size after this, with
// cs_main held.
func DisconnectTip(param *consensus.BitcoinParams, state *block.ValidationState, fBare bool) bool {

	tip := chain.GetInstance().Tip()
	if tip == nil {
		panic("the chain tip element should not equal nil")
	}
	// Read block from disk.
	blk, ret := disk.ReadBlockFromDisk(tip, param)
	if !ret{
		return disk.AbortNode(state, "Failed to read block", "")
	}

	// Apply the block atomically to the chain state.
	nStart := time.Now().UnixNano()
	{
		view := utxo.NewEmptyCoinsMap()

		if DisconnectBlock(blk, tip, view) != mUndo.DisconnectOk {
			hash := tip.GetBlockHash()
			log.Error(fmt.Sprintf("DisconnectTip(): DisconnectBlock %s failed ", hash.ToString()))
			return false
		}
		flushed := view.Flush(blk.Header.HashPrevBlock)
		if !flushed {
			panic("view flush error !!!")
		}

	}
	// replace implement with log.Print(in C++).
	log.Info("bench-debug - Disconnect block : %.2fms\n",
		float64(time.Now().UnixNano()-nStart)*0.001)

	// Write the chain state to disk, if necessary.
	if !disk.FlushStateToDisk(state, disk.FlushStateIfNeeded, 0) {
		return false
	}

	if !fBare {
		// Resurrect mempool transactions from the disconnected block.
		for _, tx := range blk.Txs {
			// ignore validation errors in resurrected transactions
			if tx.IsCoinBase() {
				mempool.Gpool.RemoveTxRecursive(tx,mempool.REORG)
			}else{
				e := lmp.AccpetTxToMemPool(tx, chain.GetInstance())
				if e != nil{
					mempool.Gpool.RemoveTxRecursive(tx,mempool.REORG)
				}
			}
		}
		// AcceptToMemoryPool/addUnchecked all assume that new memPool entries
		// have no in-memPool children, which is generally not true when adding
		// previously-confirmed transactions back to the memPool.
		// UpdateTransactionsFromBlock finds descendants of any transactions in
		// this block that were added back and cleans up the memPool state.
		//mempool.Gpool.UpdateTransactionsFromBlock(vHashUpdate)
	}

	// Update chainActive and related variables.
	UpdateTip(param, tip.Prev)
	// Let wallets know transactions went from 1-confirmed to
	// 0-confirmed or conflicted:

	// todo !!! add  GetMainSignals().SyncTransaction()
	return true
}

// UpdateTip Update chainActive and related internal data structures.
func UpdateTip(param *consensus.BitcoinParams, pindexNew *blockindex.BlockIndex) {
	chain.GetInstance().SetTip(pindexNew)
	// New best block
	//GMemPool.AddTransactionsUpdated(1)

	//	TODO !!! add Parallel Programming boost::condition_variable
	warningMessages := make([]string, 0)
	if !IsInitialBlockDownload() {
		nUpgraded := 0
		tip := pindexNew
		for bit := 0; bit < VersionBitsNumBits; bit++ {
			checker := NewWarningBitsConChecker(bit)
			state := GetStateFor(checker, index, param, GWarningCache[bit])
			if state == ThresholdActive || state == ThresholdLockedIn {
				if state == ThresholdActive {
					strWaring := fmt.Sprintf("Warning: unknown new rules activated (versionbit %d)", bit)
					msg.SetMiscWarning(strWaring)
					if !gWarned {
						AlertNotify(strWaring)
						gWarned = true
					}
				} else {
					warningMessages = append(warningMessages,
						fmt.Sprintf("unknown new rules are about to activate (versionbit %d)", bit))
				}
			}
		}
		// Check the version of the last 100 blocks to see if we need to
		// upgrade:
		for i := 0; i < 100 && index != nil; i++ {
			nExpectedVersion := ComputeBlockVersion(index.Prev, param, VBCache)
			if index.Header.Version > VersionBitsLastOldBlockVersion &&
				(int(index.Header.Version)&(^nExpectedVersion) != 0) {
				nUpgraded++
				index = index.Prev
			}
		}
		if nUpgraded > 0 {
			warningMessages = append(warningMessages,
				fmt.Sprintf("%d of last 100 blocks have unexpected version", nUpgraded))
		}
		if nUpgraded > 100/2 {
			strWarning := fmt.Sprintf("Warning: Unknown block versions being mined!" +
				" It's possible unknown rules are in effect")
			// notify GetWarnings(), called by Qt and the JSON-RPC code to warn
			// the user:
			msg.SetMiscWarning(strWarning)
			if !gWarned {
				AlertNotify(strWarning)

				gWarned = true
			}
		}
	}
	txdata := param.TxData()
	tip := chain.GetInstance().Tip()
	utxoTip := utxo.GetUtxoCacheInstance()
	logs.Info("%s: new best=%s height=%d version=0x%08x work=%.8g tx=%lu "+
		"date='%s' progress=%f cache=%.1f(%utxo)", log.TraceLog(), tip.BlockHash.ToString(),
		tip.Height, tip.Header.Version,
		tip.ChainWork.String(), tip.ChainTxCount,
		time.Unix(int64(tip.Header.Time), 0).String(),
		GuessVerificationProgress(txdata, tip),
		utxoTip.DynamicMemoryUsage(), utxoTip.GetCacheSize())
	if len(warningMessages) != 0 {
		logs.Info("waring= %s", strings.Join(warningMessages, ","))
	}
}

func DisconnectBlock(pblock *block.Block, pindex *blockindex.BlockIndex, view *utxo.CoinsMap) mUndo.DisconnectResult {

	hashA := pindex.GetBlockHash()
	hashB := utxo.GetUtxoCacheInstance().GetBestBlock()
	if !bytes.Equal(hashA[:], hashB[:]) {
		panic("the two hash should be equal ...")
	}
	pos := pindex.GetUndoPos()
	if pos.IsNull() {
		logs.Error("DisconnectBlock(): no undo data available")
		return mUndo.DisconnectFailed
	}
	blockUndo, ret := undo.UndoReadFromDisk(&pos, *pindex.Prev.GetBlockHash())
	if !ret {
		logs.Error("DisconnectBlock(): failure reading undo data")
		return mUndo.DisconnectFailed
	}

	return undo.ApplyBlockUndo(blockUndo, pblock, view)
}
