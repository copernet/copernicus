package chain

import (
	"fmt"
	"time"
	
	"github.com/btcboost/copernicus/errcode"
	// lblock "github.com/btcboost/copernicus/logic/block"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/mempool"
	lmp "github.com/btcboost/copernicus/logic/mempool"
	"github.com/btcboost/copernicus/util"
	
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/logic/undo"
	mUndo "github.com/btcboost/copernicus/model/undo"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/persist/disk"
	
	"bytes"
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/model/pow"
)

func AcceptBlock(b *block.Block) (*blockindex.BlockIndex, error) {

	var bIndex, err = AcceptBlockHeader(&b.Header)
	if err != nil {
		return nil, err
	}
	log.Info(bIndex)

	return nil, nil
}

func AcceptBlockHeader(bh *block.BlockHeader) (*blockindex.BlockIndex, error) {
	var c = chain.GetInstance()

	bIndex := c.FindBlockIndex(bh.GetHash())
	if bIndex != nil {
		if bIndex.HeaderValid() == false {
			return nil, errcode.New(errcode.ErrorBlockHeaderNoValid)
		}

		return bIndex, nil
	}

	bIndex = blockindex.NewBlockIndex(bh)
	bIndex.Prev = c.FindBlockIndex(bh.HashPrevBlock)
	if bIndex.Prev == nil {
		return nil, errcode.New(errcode.ErrorBlockHeaderNoParent)
	}

	bIndex.Height = bIndex.Prev.Height + 1
	bIndex.TimeMax = util.MaxU32(bIndex.Prev.TimeMax,bIndex.Header.GetBlockTime())
	work := pow.GetBlockProof(bIndex)
	bIndex.ChainWork = *bIndex.Prev.ChainWork.Add(&bIndex.Prev.ChainWork,work)
	c.AddToIndexMap(bIndex)

	return bIndex, nil
}

/*
func ConnectBlock(param *consensus.BitcoinParams, pblock *block.Block, state *block.ValidationState,
	pindex *blockindex.BlockIndex, view *utxo.CoinsMap, fJustCheck bool) bool {

	// TODO: AssertLockHeld(cs_main);
	// var sc sync.RWMutex
	// sc.Lock()
	// defer sc.Unlock()

	nTimeStart := utils.GetMicrosTime()

	// Check it again in case a previous version let a bad block in
	if !CheckBlock(param, pblock, state, !fJustCheck, !fJustCheck) {
		logs.Error(fmt.Sprintf("CheckBlock: %s", FormatStateMessage(state)))
		return false
	}

	// Verify that the view's current state corresponds to the previous block
	hashPrevBlock := *pindex.Prev.GetBlockHash()

	if hashPrevBlock != view.GetBestBlock() {
		panic("error: hashPrevBlock not equal view.GetBestBlock()")
	}

	// Special case for the genesis block, skipping connection of its
	// transactions (its coinbase is unspendable)
	if pblock.Hash.IsEqual(param.GenesisHash) {
		if !fJustCheck {
			view.SetBestBlock(*pindex.GetBlockHash())
		}
		return true
	}

	fScriptChecks := true
	if HashAssumeValid != utils.HashZero {
		// We've been configured with the hash of a block which has been
		// externally verified to have a valid history. A suitable default value
		// is included with the software and updated from time to time. Because
		// validity relative to a piece of software is an objective fact these
		// defaults can be easily reviewed. This setting doesn't force the
		// selection of any particular chain but makes validating some faster by
		// effectively caching the result of part of the verification.
		if it, ok := MapBlockIndex.Data[HashAssumeValid]; ok {
			if it.GetAncestor(pindex.Height) == pindex && gIndexBestHeader.GetAncestor(pindex.Height) == pindex &&
				gIndexBestHeader.ChainWork.Cmp(&param.MinimumChainWork) > 0 {
				// This block is a member of the assumed verified chain and an
				// ancestor of the best header. The equivalent time check
				// discourages hashpower from extorting the network via DOS
				// attack into accepting an invalid block through telling users
				// they must manually set assumevalid. Requiring a software
				// change or burying the invalid block, regardless of the
				// setting, makes it hard to hide the implication of the demand.
				// This also avoids having release candidates that are hardly
				// doing any signature verification at all in testing without
				// having to artificially set the default assumed verified block
				// further back. The test against nMinimumChainWork prevents the
				// skipping when denied access to any chain at least as good as
				// the expected chain.
				fScriptChecks = (GetBlockProofEquivalentTime(gIndexBestHeader, pindex, gIndexBestHeader, param)) <= 60*60*24*7*2
			}
		}
	}

	nTime1 := utils.GetMicrosTime()
	gTimeCheck += nTime1 - nTimeStart
	log.Print("bench", "debug", " - Sanity checks: %.2fms [%.2fs]\n",
		0.001*float64(nTime1-nTimeStart), float64(gTimeCheck)*0.000001)

	// Do not allow blocks that contain transactions which 'overwrite' older
	// transactions, unless those are already completely spent. If such
	// overwrites are allowed, coinbases and transactions depending upon those
	// can be duplicated to remove the ability to spend the first instance --
	// even after being sent to another address. See BIP30 and
	// http://r6.ca/blog/20120206T005236Z.html for more information. This logic
	// is not necessary for memory pool transactions, as AcceptToMemoryPool
	// already refuses previously-known transaction ids entirely. This rule was
	// originally applied to all blocks with a timestamp after March 15, 2012,
	// 0:00 UTC. Now that the whole chain is irreversibly beyond that time it is
	// applied to all blocks except the two in the chain that violate it. This
	// prevents exploiting the issue against nodes during their initial block
	// download.
	fEnforceBIP30 := (pindex.BlockHash != utils.HashZero) || !(pindex.Height == 91842 &&
		*pindex.GetBlockHash() == *utils.HashFromString("0x00000000000a4d0a398161ffc163c503763b1f4360639393e0e4c8e300e0caec")) ||
		*pindex.GetBlockHash() == *utils.HashFromString("0x00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721")

	// Once BIP34 activated it was not possible to create new duplicate
	// coinBases and thus other than starting with the 2 existing duplicate
	// coinBase pairs, not possible to create overwriting txs. But by the time
	// BIP34 activated, in each of the existing pairs the duplicate coinBase had
	// overwritten the first before the first had been spent. Since those
	// coinBases are sufficiently buried its no longer possible to create
	// further duplicate transactions descending from the known pairs either. If
	// we're on the known chain at height greater than where BIP34 activated, we
	// can save the db accesses needed for the BIP30 check.
	pindexBIP34height := pindex.Prev.GetAncestor(param.BIP34Height)
	// Only continue to enforce if we're below BIP34 activation height or the
	// block hash at that height doesn't correspond.
	fEnforceBIP30 = fEnforceBIP30 && (&pindexBIP34height == nil ||
		!(*pindexBIP34height.GetBlockHash() == param.BIP34Hash))

	if fEnforceBIP30 {
		for _, tx := range pblock.Txs {
			for o := 0; o < len(tx.Outs); o++ {
				outPoint := &core.OutPoint{
					Hash:  Tx.GetHash(),
					Index: uint32(o),
				}
				if view.HaveCoin(outPoint) {
					return state.Dos(100, false, core.RejectInvalid, "bad-txns-BIP30",
						false, "")
				}
			}
		}
	}

	// Start enforcing BIP68 (sequence locks) using versionBits logic.
	nLockTimeFlags := 0
	if VersionBitsState(pindex.Prev, param, consensus.DeploymentCSV, VBCache) == ThresholdActive {
		nLockTimeFlags |= consensus.LocktimeVerifySequence
	}

	flags := GetBlockScriptFlags(pindex, param)
	nTime2 := utils.GetMicrosTime()
	gTimeForks += nTime2 - nTime1
	log.Print("bench", "debug", " - Fork checks: %.2fms [%.2fs]\n",
		0.001*float64(nTime2-nTime1), float64(gTimeForks)*0.000001)

	var blockundo *BlockUndo
	// TODO:not finish
	// CCheckQueueControl<CScriptCheck> control(fScriptChecks ? &scriptCheckQueue : nullptr);

	prevheights := make([]int, 0)
	var nFees utils.Amount
	nInputs := 0

	// SigOps counting. We need to do it again because of P2SH.
	nSigOpsCount := 0
	currentBlockSize := pblock.SerializeSize()
	nMaxSigOpsCount := consensus.GetMaxBlockSigOpsCount(uint64(currentBlockSize))

	tmpBlockPos := pindex.GetBlockPos()
	txPos := &core.DiskTxPos{
		BlockIn:    &tmpBlockPos,
		TxOffsetIn: len(pblock.Txs),
	}

	var vPos map[utils.Hash]core.DiskTxPos
	for i := 0; i < len(pblock.Txs); i++ {
		tx := pblock.Txs[i]
		nInputs += len(tx.Ins)
		if !tx.IsCoinBase() {
			if !view.HaveInputs(tx) {
				logs.Error("ConnectBlock(): inputs missing/spent")
				return state.Dos(100, false, core.RejectInvalid,
					"bad-txns-inputs-missIngorSpent", false, "")
			}

			// Check that transaction is BIP68 final BIP68 lock checks (as
			// opposed to nLockTime checks) must be in ConnectBlock because they
			// require the UTXO set.
			for j := 0; j < len(tx.Ins); j++ {
				prevheights[j] = int(view.AccessCoin(tx.Ins[j].PreviousOutPoint).GetHeight())
			}

			if !SequenceLocks(tx, nLockTimeFlags, prevheights, pindex) {
				logs.Error("ConnectBlock(): inputs missing/spent")
				return state.Dos(100, false, core.RejectInvalid, "bad-txns-nonFinal",
					false, "")
			}
		}
		// GetTransactionSigOpCount counts 2 types of sigOps:
		// * legacy (always)
		// * p2sh (when P2SH enabled in flags and excludes coinBase)
		txSigOpsCount := GetTransactionSigOpCount(tx, view, uint(flags))
		if txSigOpsCount > int(policy.MaxTxSigOpsCount) {
			return state.Dos(100, false, core.RejectInvalid, "bad-txn-sigOps",
				false, "")
		}

		nSigOpsCount += txSigOpsCount
		if nSigOpsCount > int(nMaxSigOpsCount) {
			logs.Error("ConnectBlock(): too many sigOps")
			return state.Dos(100, false, core.RejectInvalid,
				"bad-blk-sigops", false, "")
		}

		if !tx.IsCoinBase() {
			fee := view.GetValueIn(tx) - utils.Amount(tx.GetValueOut())
			nFees += fee
			// Don't cache results if we're actually connecting blocks (still consult the cache, though).
			fCacheResults := fJustCheck
			vChecks := make([]*ScriptCheck, 0)
			if !CheckInputs(tx, state, view, fScriptChecks, flags, fCacheResults, fCacheResults,
				core.NewPrecomputedTransactionData(tx), vChecks) {
				logs.Error(fmt.Sprintf("ConnectBlock(): CheckInputs on %s failed with %s",
					tx.GetHash(), FormatStateMessage(state)))
				return false
			}

			// todo:control.add(vChecks)
		}

		var undoDummy TxUndo
		if i > 0 {
			blockundo.txundo = append(blockundo.txundo, newTxUndo())
		}
		if i == 0 {
			undoDummy.PrevOut = view.UpdateCoins(tx, pindex.Height)
		} else {
			blockundo.txundo[len(blockundo.txundo)-1].PrevOut = view.UpdateCoins(tx, pindex.Height)
		}
		_ = undoDummy

		vPos[Tx.GetHash()] = *txPos
		txPos.TxOffsetIn += tx.SerializeSize()
	}

	nTime3 := utils.GetMicrosTime()
	gTimeConnect += nTime3 - nTime2
	if nInputs <= 1 {
		log.Print("bench", "debug", " - Connect %u transactions: %.2fms (%.3fms/tx, %.3fms/txin) [%.2fs]\n",
			len(pblock.Txs), 0.001*float64(nTime3-nTime2), 0.001*float64(nTime3-nTime2)/float64(len(pblock.Txs)), 0, float64(gTimeConnect)*0.000001)
	} else {
		log.Print("bench", "debug", " - Connect %u transactions: %.2fms (%.3fms/tx, %.3fms/txin) [%.2fs]\n",
			len(pblock.Txs), 0.001*float64(nTime3-nTime2), 0.001*float64(nTime3-nTime2)/float64(len(pblock.Txs)),
			0.001*float64(nTime3-nTime2)/float64(nInputs-1), float64(gTimeConnect)*0.000001)
	}

	blockReward := nFees + GetBlockSubsidy(pindex.Height, param)

	if pblock.Txs[0].GetValueOut() > int64(blockReward) {
		logs.Error("ConnectBlock(): coinbase pays too much ")
		return state.Dos(100, false,
			core.RejectInvalid, "bad-cb-amount", false, "")
	}

	// todo:control

	nTime4 := utils.GetMicrosTime()
	gTimeVerify += nTime4 - nTime2

	if nInputs <= 1 {
		log.Print("bench", "debug", " - Verify %u txIns: %.2fms (%.3fms/txIn) [%.2fs]\n",
			nInputs-1, 0.001*float64(nTime4-nTime2), 0, float64(gTimeVerify)*0.000001)
	} else {
		log.Print("bench", "debug", " - Verify %u txIns: %.2fms (%.3fms/txIn) [%.2fs]\n",
			nInputs-1, 0.001*float64(nTime4-nTime2), 0.001*float64(nTime4-nTime2)/float64(nInputs-1),
			float64(gTimeVerify)*0.000001)
	}

	if fJustCheck {
		return true
	}

	// Write undo information to disk
	tmpUndoPos := pindex.GetUndoPos()
	if tmpUndoPos.IsNull() || !pindex.IsValid(core.BlockValidScripts) {
		if tmpUndoPos.IsNull() {
			var pos core.DiskBlockPos
			// todo：SerializeSize
			// if !FindUndoPos(state, pindex.File, pos, len(blockundo.)) {
			// 	logger.ErrorLog("ConnectBlock(): FindUndoPos failed")
			// }
			if !UndoWriteToDisk(blockundo, &pos, *pindex.Prev.GetBlockHash(), param.BitcoinNet) {
				return AbortNode(state, "Failed to write undo data", "")
			}

			// update nUndoPos in block index
			pindex.UndoPos = pos.Pos
			pindex.Status |= core.BlockHaveUndo
		}

		pindex.RaiseValidity(core.BlockValidScripts)
		gSetDirtyBlockIndex.AddItem(pindex)
	}

	if GTxIndex { // todo:
		return AbortNode(state, "Failed to write transaction index", "")
	}

	// add this block to the view's block chain
	view.SetBestBlock(*pindex.GetBlockHash())

	nTime5 := utils.GetMicrosTime()
	gTimeIndex += nTime5 - nTime4
	log.Print("bench", "debug", " - Index writing: %.2fms [%.2fs]\n",
		0.001*float64(nTime5-nTime4), float64(gTimeIndex)*0.000001)

	// Watch for changes to the previous coinbase transaction.
	// todo:GetMainSignals().UpdatedTransaction(hashPrevBestCoinBase);
	gHashPrevBestCoinBase = pblock.Txs[0].Hash

	nTime6 := utils.GetMicrosTime()
	gTimeCallbacks += nTime6 - nTime5
	log.Print("bench", "debug", " - Callbacks: %.2fms [%.2fs]\n",
		0.001*float64(nTime6-nTime5), float64(gTimeCallbacks)*0.000001)
	return true
}
*/

/*
// ConnectTip Connect a new block to chainActive. block is either nullptr or a pointer to
// a CBlock corresponding to indexNew, to bypass loading it again from disk.
// The block is always added to connectTrace (either after loading from disk or
// by copying block) - if that is not intended, care must be taken to remove
// the last entry in blocksConnected in case of failure.
func ConnectTip(param *consensus.BitcoinParams, state *block.ValidationState, pIndexNew *blockindex.BlockIndex,
	block *block.Block, connectTrace map[*blockindex.BlockIndex]*block.Block) bool {
	tip := chain.GetInstance().Tip()
	if pIndexNew.Prev != tip {
		log.Error("error: try to connect to inactive chain!!!")
		panic("error: try to connect to inactive chain!!!")
	}
	// Read block from disk.
	nTime1 := time.Now().UnixNano()
	if block == nil {
		blockNew, err := disk.ReadBlockFromDisk(pIndexNew, param)
		if !err || blockNew == nil {
			return disk.AbortNode(state, "Failed to read block", "")
		}
		connectTrace[pIndexNew] = blockNew
		block = blockNew

	} else {
		connectTrace[pIndexNew] = block
	}
	blockConnecting := block
	// Apply the block atomically to the chain state.
	nTime2 := time.Now().UnixNano()
	disk.GlobalTimeReadFromDisk += nTime2 - nTime1
	log.Info("  - Load block from disk: %#v ms total: [%#v s]\n", (nTime2-nTime1)/1000, disk.GlobalTimeReadFromDisk/1000000)

	view := utxo.NewEmptyCoinsMap()
	rv := ConnectBlock(param, blockConnecting, state, pIndexNew, view, false)
	// todo etMainSignals().BlockChecked(blockConnecting, state)
	if !rv {
		if state.IsInvalid() {
			InvalidBlockFound(indexNew, state)
		}
		hash := indexNew.GetBlockHash()
		logs.Error(fmt.Sprintf("ConnectTip(): ConnectBlock %s failed", hash.ToString()))
		return false
	}
	nTime3 := utils.GetMicrosTime()
	gTimeConnectTotal += nTime3 - nTime2
	log.Print("bench", "debug", " - Connect total: %.2fms [%.2fs]\n",
		float64(nTime3-nTime2)*0.001, float64(gTimeConnectTotal)*0.000001)
	flushed := view.Flush()
	if !flushed {
		panic("here should be true when view flush state")
	}
	nTime4 := utils.GetMicrosTime()
	gTimeFlush += nTime4 - nTime3
	log.Print("bench", "debug", " - Flush: %.2fms [%.2fs]\n",
		float64(nTime4-nTime3)*0.001, float64(gTimeFlush)*0.000001)
	// Write the chain state to disk, if necessary.
	if !FlushStateToDisk(state, FlushStateIfNeeded, 0) {
		return false
	}
	nTime5 := utils.GetMicrosTime()
	gTimeChainState += nTime5 - nTime4
	log.Print("bench", "debug", " - Writing chainstate: %.2fms [%.2fs]\n",
		float64(nTime5-nTime4)*0.001, float64(gTimeChainState)*0.000001)
	// Remove conflicting transactions from the mempool.;
	GMemPool.RemoveTxSelf(blockConnecting.Txs)
	// Update chainActive & related variables.
	UpdateTip(param, indexNew)
	nTime6 := utils.GetMicrosTime()
	gTimePostConnect += nTime6 - nTime1
	gTimeTotal += nTime6 - nTime1
	log.Print("bench", "debug", " - Connect postprocess: %.2fms [%.2fs]\n",
		float64(nTime6-nTime5)*0.001, float64(gTimePostConnect)*0.000001)
	log.Print("bench", "debug", " - Connect block: %.2fms [%.2fs]\n",
		float64(nTime6-nTime1)*0.001, float64(gTimeTotal)*0.000001)

	return true
}
*/


// DisconnectTip Disconnect chainActive's tip. You probably want to call
// mempool.removeForReorg and manually re-limit mempool size after this, with
// cs_main held.
func DisconnectTip(param *chainparams.BitcoinParams, state *block.ValidationState, fBare bool) bool {

	tip := chain.GetInstance().Tip()
	if tip == nil {
		panic("the chain tip element should not equal nil")
	}
	// Read block from disk.
	blk, ret := disk.ReadBlockFromDisk(tip, param)
	if !ret {
		return disk.AbortNode(state, "Failed to read block", "")
	}

	// Apply the block atomically to the chain state.
	nStart := time.Now().UnixNano()
	{
		view := utxo.NewEmptyCoinsMap()

		if DisconnectBlock(blk, tip, view) != mUndo.DisconnectOk {
			hash := tip.GetBlockHash()
			log.Error(fmt.Sprintf("DisconnectTip(): DisconnectBlock %s failed ", hash.String()))
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
				mempool.Gpool.RemoveTxRecursive(tx, mempool.REORG)
			} else {
				_, e := lmp.AccpetTxToMemPool(tx, chain.GetInstance())
				if e != nil {
					mempool.Gpool.RemoveTxRecursive(tx, mempool.REORG)
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
func UpdateTip(param *chainparams.BitcoinParams, pindexNew *blockindex.BlockIndex) {
	//chain.GetInstance().SetTip(pindexNew)
	//// New best block
	////GMemPool.AddTransactionsUpdated(1)
	//
	////	TODO !!! add Parallel Programming boost::condition_variable
	//warningMessages := make([]string, 0)
	//if !IsInitialBlockDownload() {
	//	nUpgraded := 0
	//	tip := pindexNew
	//	for bit := 0; bit < VersionBitsNumBits; bit++ {
	//		checker := NewWarningBitsConChecker(bit)
	//		state := GetStateFor(checker, index, param, GWarningCache[bit])
	//		if state == ThresholdActive || state == ThresholdLockedIn {
	//			if state == ThresholdActive {
	//				strWaring := fmt.Sprintf("Warning: unknown new rules activated (versionbit %d)", bit)
	//				msg.SetMiscWarning(strWaring)
	//				if !gWarned {
	//					AlertNotify(strWaring)
	//					gWarned = true
	//				}
	//			} else {
	//				warningMessages = append(warningMessages,
	//					fmt.Sprintf("unknown new rules are about to activate (versionbit %d)", bit))
	//			}
	//		}
	//	}
	//	// Check the version of the last 100 blocks to see if we need to
	//	// upgrade:
	//	for i := 0; i < 100 && index != nil; i++ {
	//		nExpectedVersion := ComputeBlockVersion(index.Prev, param, VBCache)
	//		if index.Header.Version > VersionBitsLastOldBlockVersion &&
	//			(int(index.Header.Version)&(^nExpectedVersion) != 0) {
	//			nUpgraded++
	//			index = index.Prev
	//		}
	//	}
	//	if nUpgraded > 0 {
	//		warningMessages = append(warningMessages,
	//			fmt.Sprintf("%d of last 100 blocks have unexpected version", nUpgraded))
	//	}
	//	if nUpgraded > 100/2 {
	//		strWarning := fmt.Sprintf("Warning: Unknown block versions being mined!" +
	//			" It's possible unknown rules are in effect")
	//		// notify GetWarnings(), called by Qt and the JSON-RPC code to warn
	//		// the user:
	//		msg.SetMiscWarning(strWarning)
	//		if !gWarned {
	//			AlertNotify(strWarning)
	//
	//			gWarned = true
	//		}
	//	}
	//}
	//txdata := param.TxData()
	//tip := chain.GetInstance().Tip()
	//utxoTip := utxo.GetUtxoCacheInstance()
	//logs.Info("%s: new best=%s height=%d version=0x%08x work=%.8g tx=%lu "+
	//	"date='%s' progress=%f cache=%.1f(%utxo)", log.TraceLog(), tip.BlockHash.ToString(),
	//	tip.Height, tip.Header.Version,
	//	tip.ChainWork.String(), tip.ChainTxCount,
	//	time.Unix(int64(tip.Header.Time), 0).String(),
	//	GuessVerificationProgress(txdata, tip),
	//	utxoTip.DynamicMemoryUsage(), utxoTip.GetCacheSize())
	//if len(warningMessages) != 0 {
	//	logs.Info("waring= %s", strings.Join(warningMessages, ","))
	//}
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
	blockUndo, ret := disk.UndoReadFromDisk(&pos, *pindex.Prev.GetBlockHash())
	if !ret {
		logs.Error("DisconnectBlock(): failure reading undo data")
		return mUndo.DisconnectFailed
	}

	return undo.ApplyBlockUndo(blockUndo, pblock, view)
}
