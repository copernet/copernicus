package chain

import (
"bytes"
"fmt"
"strings"
"time"





"github.com/btcboost/copernicus/errcode"
lmp "github.com/btcboost/copernicus/logic/mempool"
"github.com/btcboost/copernicus/model/block"
"github.com/btcboost/copernicus/model/blockindex"
mchain "github.com/btcboost/copernicus/model/chain"
"github.com/btcboost/copernicus/model/chainparams"
"github.com/btcboost/copernicus/model/consensus"
"github.com/btcboost/copernicus/model/mempool"
"github.com/btcboost/copernicus/model/script"
"github.com/btcboost/copernicus/model/versionbits"
"github.com/btcboost/copernicus/persist/global"
"github.com/btcboost/copernicus/util"
"github.com/btcboost/copernicus/util/amount"

	

"github.com/btcboost/copernicus/log"
ltx "github.com/btcboost/copernicus/logic/tx"
"github.com/btcboost/copernicus/logic/undo"



mUndo "github.com/btcboost/copernicus/model/undo"
"github.com/btcboost/copernicus/model/utxo"
"github.com/btcboost/copernicus/persist/disk"



lblock "github.com/btcboost/copernicus/logic/block"
"github.com/btcboost/copernicus/model/pow"

)

const MinBlocksToKeep = int32(288)

func AcceptBlock(params *chainparams.BitcoinParams, pblock *block.Block, state *block.ValidationState,
	fRequested bool, fNewBlock *bool) (bIndex *blockindex.BlockIndex, dbp *block.DiskBlockPos, err error) {
	if pblock != nil {
		*fNewBlock = false
	}
	bIndex, err = AcceptBlockHeader(&pblock.Header, params)
	if err != nil {
		return
	}
	log.Info(bIndex)

	if bIndex.Accepted() {
		err = errcode.ProjectError{Code: 3009}

		return
	}
	if !fRequested {
		tip := mchain.GetInstance().Tip()
		tipWork := tip.ChainWork
		fHasMoreWork := false
		if tip == nil {
			fHasMoreWork = true
		} else if bIndex.ChainWork.Cmp(&tipWork) == 1 {
			fHasMoreWork = true
		}
		if !fHasMoreWork {
			err = errcode.ProjectError{Code: 3008}

			return
		}
		fTooFarAhead := bIndex.Height > tip.Height+MinBlocksToKeep
		if fTooFarAhead {
			err = errcode.ProjectError{Code: 3007}

			return
		}
	}
	if bIndex.AllValid() == false {
		suc := lblock.CheckBlock(params, pblock, state, true, true)
		if !suc {
			return
		}

		bIndex.AddStatus(blockindex.StatusAllValid)
	}
	gPersist := global.GetInstance()
	if !lblock.CheckBlock(params, pblock, state, true, true) {
		bIndex.AddStatus(blockindex.StatusFailed)
		gPersist.AddDirtyBlockIndex(pblock.GetHash(), bIndex)
		err = errcode.ProjectError{Code: 3005}
		return
	}
	if !lblock.ContextualCheckBlock(params, pblock, state, bIndex.Prev) {
		bIndex.AddStatus(blockindex.StatusFailed)
		gPersist.AddDirtyBlockIndex(pblock.GetHash(), bIndex)
		err = errcode.ProjectError{Code: 3005}
		return
	}
	*fNewBlock = true

	dbp, err = lblock.WriteBlockToDisk(bIndex, pblock)
	if err != nil {
		bIndex.AddStatus(blockindex.StatusFailed)
		gPersist.GlobalDirtyBlockIndex[pblock.GetHash()] = bIndex
		err = errcode.ProjectError{Code: 3006}
		return
	}
	ReceivedBlockTransactions(pblock, bIndex, dbp)
	bIndex.SubStatus(blockindex.StatusWaitingData)
	bIndex.AddStatus(blockindex.StatusDataStored)
	gPersist.AddDirtyBlockIndex(pblock.GetHash(), bIndex)
	return
}

func AcceptBlockHeader(bh *block.BlockHeader, params *chainparams.BitcoinParams) (*blockindex.BlockIndex, error) {
	var c = mchain.GetInstance()

	bIndex := c.FindBlockIndex(bh.GetHash())
	if bIndex != nil {
		return bIndex, nil
	}

	//this is a new blockheader
	err := lblock.CheckBlockHeader(bh, params, true)
	if err != nil {
		return nil, err
	}

	bIndex = blockindex.NewBlockIndex(bh)
	bIndex.Prev = c.FindBlockIndex(bh.HashPrevBlock)
	if bIndex.Prev == nil {
		return nil, errcode.New(errcode.ErrorBlockHeaderNoParent)
	}
	bIndex.Height = bIndex.Prev.Height + 1
	bIndex.TimeMax = util.MaxU32(bIndex.Prev.TimeMax, bIndex.Header.GetBlockTime())
	work := pow.GetBlockProof(bIndex)
	bIndex.ChainWork = *bIndex.Prev.ChainWork.Add(&bIndex.Prev.ChainWork, work)
	bIndex.AddStatus(blockindex.StatusWaitingData)

	err = c.AddToIndexMap(bIndex)
	if err != nil {
		return nil, err
	}
	
	return bIndex, nil
}

// GetBlockScriptFlags Returns the script flags which should be checked for a given block
func GetBlockScriptFlags(pindex *blockindex.BlockIndex, param *chainparams.BitcoinParams) uint32 {
	// TODO: AssertLockHeld(cs_main);
	// var sc sync.RWMutex
	// sc.Lock()
	// defer sc.Unlock()

	// BIP16 didn't become active until Apr 1 2012
	nBIP16SwitchTime := 1333238400
	fStrictPayToScriptHash := int(pindex.GetBlockTime()) >= nBIP16SwitchTime

	var flags uint32

	if fStrictPayToScriptHash {
		flags = script.ScriptVerifyP2SH
	} else {
		flags = script.ScriptVerifyNone
	}

	// Start enforcing the DERSIG (BIP66) rule
	if pindex.Height >= param.BIP66Height {
		flags |= script.ScriptVerifyDersig
	}

	// Start enforcing CHECKLOCKTIMEVERIFY (BIP65) rule
	if pindex.Height >= param.BIP65Height {
		flags |= script.ScriptVerifyCheckLockTimeVerify
	}

	// Start enforcing BIP112 (CHECKSEQUENCEVERIFY) using versionbits logic.
	if versionbits.VersionBitsState(pindex.Prev, param, consensus.DeploymentCSV, versionbits.VBCache) == versionbits.ThresholdActive {
		flags |= script.ScriptVerifyCheckSequenceVerify
	}
	// If the UAHF is enabled, we start accepting replay protected txns
	if chainparams.IsUAHFEnabled(pindex.Height) {
		flags |= script.ScriptVerifyStrictEnc
		flags |= script.ScriptEnableSigHashForkId
	}

	// If the Cash HF is enabled, we start rejecting transaction that use a high
	// s in their signature. We also make sure that signature that are supposed
	// to fail (for instance in multisig or other forms of smart contracts) are
	// null.
	if IsCashHFEnabled(param, pindex.GetMedianTimePast()) {
		flags |= script.ScriptVerifyLowS
		flags |= script.ScriptVerifyNullFail
	}

	return flags
}

func IsCashHFEnabled(params *chainparams.BitcoinParams, medianTimePast int64) bool {
	return params.CashHardForkActivationTime <= medianTimePast
}

var HashAssumeValid util.Hash

func ConnectBlock(params *chainparams.BitcoinParams, pblock *block.Block, state *block.ValidationState,
	pindex *blockindex.BlockIndex, view *utxo.CoinsMap, fJustCheck bool) bool {
	gChain := mchain.GetInstance()
	tip := gChain.Tip()
	nTimeStart := util.GetMicrosTime()

	// Check it again in case a previous version let a bad block in
	if lblock.CheckBlock(params, pblock, state, true,
		true) {
		return false
	}
	// if err := ltx.CheckBlockTransactions(pblock.Txs, pindex.Height, lockTime, blockReward, maxBlockSigOps); err != nil{
	// 	log.Error(fmt.Sprintf("CheckBlock: %#v", state))
	// 	return false
	// }

	// Verify that the view's current state corresponds to the previous block
	hashPrevBlock := *pindex.Prev.GetBlockHash()
	gUtxo := utxo.GetUtxoCacheInstance()
	bestHash := gUtxo.GetBestBlock()
	if hashPrevBlock.IsEqual(&bestHash) {
		panic("error: hashPrevBlock not equal view.GetBestBlock()")
	}

	// Special case for the genesis block, skipping connection of its
	// transactions (its coinbase is unspendable)
	blockHash := pblock.GetHash()
	if blockHash.IsEqual(params.GenesisHash) {
		if !fJustCheck {
			view.SetBestBlock(*pindex.GetBlockHash())
		}
		return true
	}

	fScriptChecks := true
	if HashAssumeValid != util.HashZero {
		// We've been configured with the hash of a block which has been
		// externally verified to have a valid history. A suitable default value
		// is included with the software and updated from time to time. Because
		// validity relative to a piece of software is an objective fact these
		// defaults can be easily reviewed. This setting doesn't force the
		// selection of any particular chain but makes validating some faster by
		// effectively caching the result of part of the verification.
		if bi := gChain.FindBlockIndex(HashAssumeValid); bi != nil {
			if bi.GetAncestor(pindex.Height) == pindex && tip.GetAncestor(pindex.Height) == pindex &&
				tip.ChainWork.Cmp(&params.MinimumChainWork) > 0 {
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
				fScriptChecks = (pow.GetBlockProofEquivalentTime(tip, pindex, tip, params)) <= 60*60*24*7*2
			}
		}
	}

	nTime1 := util.GetMicrosTime()
	gPersist := global.GetInstance()
	gPersist.GlobalTimeCheck += nTime1 - nTimeStart
	log.Print("bench", "debug", " - Sanity checks: %.2fms [%.2fs]\n",
		0.001*float64(nTime1-nTimeStart), float64(gPersist.GlobalTimeCheck)*0.000001)

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
	zHash := util.HashZero
	fEnforceBIP30 := (!blockHash.IsEqual(&zHash)) ||
		!(pindex.Height == 91842 &&
			blockHash.IsEqual(util.HashFromString("0x00000000000a4d0a398161ffc163c503763b1f4360639393e0e4c8e300e0caec")) ||
			blockHash.IsEqual(util.HashFromString("0x00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721")))

	// Once BIP34 activated it was not possible to create new duplicate
	// coinBases and thus other than starting with the 2 existing duplicate
	// coinBase pairs, not possible to create overwriting txs. But by the time
	// BIP34 activated, in each of the existing pairs the duplicate coinBase had
	// overwritten the first before the first had been spent. Since those
	// coinBases are sufficiently buried its no longer possible to create
	// further duplicate transactions descending from the known pairs either. If
	// we're on the known chain at height greater than where BIP34 activated, we
	// can save the db accesses needed for the BIP30 check.
	pindexBIP34height := pindex.Prev.GetAncestor(params.BIP34Height)
	// Only continue to enforce if we're below BIP34 activation height or the
	// block hash at that height doesn't correspond.
	hash := pindexBIP34height.GetBlockHash()
	BIP34Hash := params.BIP34Hash
	fEnforceBIP30 = fEnforceBIP30 && (&pindexBIP34height == nil ||
		!(hash.IsEqual(&BIP34Hash)))

	flags := GetBlockScriptFlags(pindex, params)
	blockSubSidy := GetBlockSubsidy(pindex.Height, params)
	nTime2 := util.GetMicrosTime()
	gPersist.GlobalTimeForks += nTime2 - nTime1
	log.Print("bench", "debug", " - Fork checks: %.2fms [%.2fs]\n",
		0.001*float64(nTime2-nTime1), float64(gPersist.GlobalTimeForks)*0.000001)

	var coinsMap, blockUndo, err = ltx.ApplyBlockTransactions(pblock.Txs, fEnforceBIP30, flags, fScriptChecks, blockSubSidy, pindex.Height)
	if err != nil {
		return false
	}
	// Write undo information to disk
	UndoPos := pindex.GetUndoPos()
	if UndoPos.IsNull() || !pindex.IsValid(blockindex.BlockValidScripts) {
		if UndoPos.IsNull() {
			var pos *block.DiskBlockPos = block.NewDiskBlockPos(pindex.File, 0)

			if err := disk.FindUndoPos(state, pindex.File, pos, blockUndo.SerializeSize()); err != nil {
				return disk.AbortNode(state, "Failed to FindUndoPos", "")
			}
			if !disk.UndoWriteToDisk(blockUndo, pos, *pindex.Prev.GetBlockHash(), params.BitcoinNet) {
				return disk.AbortNode(state, "Failed to write undo data", "")
			}

			// update nUndoPos in block index
			pindex.UndoPos = pos.Pos
			pindex.Status |= blockindex.BlockHaveUndo
		}
		pindex.RaiseValidity(blockindex.BlockValidScripts)
		gPersist.GlobalDirtyBlockIndex[*hash] = pindex
	}
	// add this block to the view's block chain
	coinsMap.SetBestBlock(blockHash)
	*view = *coinsMap
	return true
}

//todo
func InvalidBlockFound(pindex *blockindex.BlockIndex, state *block.ValidationState) {

}

func GetBlockSubsidy(height int32, params *chainparams.BitcoinParams) amount.Amount {
	halvings := height / params.SubsidyReductionInterval
	// Force block reward to zero when right shift is undefined.
	if halvings >= 64 {
		return 0
	}

	nSubsidy := amount.Amount(50 * util.COIN)
	// Subsidy is cut in half every 210,000 blocks which will occur
	// approximately every 4 years.
	return amount.Amount(uint(nSubsidy) >> uint(halvings))
}

type connectTrace map[*blockindex.BlockIndex]*block.Block

// ConnectTip Connect a new block to chainActive. block is either nullptr or a pointer to
// a CBlock corresponding to indexNew, to bypass loading it again from disk.
// The block is always added to connectTrace (either after loading from disk or
// by copying block) - if that is not intended, care must be taken to remove
// the last entry in blocksConnected in case of failure.
func ConnectTip(param *chainparams.BitcoinParams, state *block.ValidationState, pIndexNew *blockindex.BlockIndex,
	block *block.Block, connTrace connectTrace) bool {
	gChain := mchain.GetInstance()
	tip := gChain.Tip()

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
		connTrace[pIndexNew] = blockNew
		block = blockNew

	} else {
		connTrace[pIndexNew] = block
	}
	blockConnecting := block
	indexHash := blockConnecting.GetHash()
	// Apply the block atomically to the chain state.
	nTime2 := time.Now().UnixNano()
	gPersist := global.GetInstance()
	gPersist.GlobalTimeReadFromDisk += nTime2 - nTime1
	log.Info("  - Load block from disk: %#v ms total: [%#v s]\n", (nTime2-nTime1)/1000, gPersist.GlobalTimeReadFromDisk/1000000)

	view := utxo.NewEmptyCoinsMap()
	rv := ConnectBlock(param, blockConnecting, state, pIndexNew, view, false)
	if !rv {
		if state.IsInvalid() {
			InvalidBlockFound(pIndexNew, state)
		}
		log.Error(fmt.Sprintf("ConnectTip(): ConnectBlock %s failed", indexHash.String()))
		return false
	}
	nTime3 := util.GetMicrosTime()
	gPersist.GlobalTimeConnectTotal += nTime3 - nTime2
	log.Print("bench", "debug", " - Connect total: %.2fms [%.2fs]\n",
		float64(nTime3-nTime2)*0.001, float64(gPersist.GlobalTimeConnectTotal)*0.000001)
	flushed := view.Flush(indexHash)
	if !flushed {
		panic("here should be true when view flush state")
	}
	nTime4 := util.GetMicrosTime()
	gPersist.GlobalTimeFlush += nTime4 - nTime3
	log.Print("bench", "debug", " - Flush: %.2fms [%.2fs]\n",
		float64(nTime4-nTime3)*0.001, float64(gPersist.GlobalTimeFlush)*0.000001)
	// Write the chain state to disk, if necessary.
	if !disk.FlushStateToDisk(state, disk.FlushStateIfNeeded, 0) {
		return false
	}
	nTime5 := util.GetMicrosTime()
	gPersist.GlobalTimeChainState += nTime5 - nTime4
	log.Print("bench", "debug", " - Writing chainstate: %.2fms [%.2fs]\n",
		float64(nTime5-nTime4)*0.001, float64(gPersist.GlobalTimeChainState)*0.000001)
	// Remove conflicting transactions from the mempool.;
	mempool.GetInstance().RemoveTxSelf(blockConnecting.Txs)
	// Update chainActive & related variables.
	UpdateTip(param, pIndexNew)
	nTime6 := util.GetMicrosTime()
	gPersist.GlobalTimePostConnect += nTime6 - nTime1
	gPersist.GlobalTimeTotal += nTime6 - nTime1
	log.Print("bench", "debug", " - Connect postprocess: %.2fms [%.2fs]\n",
		float64(nTime6-nTime5)*0.001, float64(gPersist.GlobalTimePostConnect)*0.000001)
	log.Print("bench", "debug", " - Connect block: %.2fms [%.2fs]\n",
		float64(nTime6-nTime1)*0.001, float64(gPersist.GlobalTimeTotal)*0.000001)

	return true
}

// DisconnectTip Disconnect chainActive's tip. You probably want to call
// mempool.removeForReorg and manually re-limit mempool size after this, with
// cs_main held.
func DisconnectTip(param *chainparams.BitcoinParams, state *block.ValidationState, fBare bool) bool {
	gChain := mchain.GetInstance()
	tip := gChain.Tip()
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
				mempool.GetInstance().RemoveTxRecursive(tx, mempool.REORG)
			} else {
				e := lmp.AcceptTxToMemPool(tx)
				if e != nil {
					mempool.GetInstance().RemoveTxRecursive(tx, mempool.REORG)
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

	return true
}

// UpdateTip Update chainActive and related internal data structures.
func UpdateTip(param *chainparams.BitcoinParams, pindexNew *blockindex.BlockIndex) {
	gChain := mchain.GetInstance()
	gChain.SetTip(pindexNew)

	//	TODO !!! notify mempool update tx
	warningMessages := make([]string, 0)
	if !undo.IsInitialBlockDownload() {
		// todo check block version and warn it
		// nUpgraded := 0
		// pindex := gChain.Tip()
		// for bit := 0; bit < versionbits.VersionBitsNumBits; bit++ {
		// 	checker := versionbits.NewWarningBitsConChecker(bit)
		// 	state := versionbits.GetStateFor(checker, pindex, param, GWarningCache[bit])
		// 	if state == versionbits.ThresholdActive || state == versionbits.ThresholdLockedIn {
		// 		if state == versionbits.ThresholdActive {
		// 			strWaring := fmt.Sprintf("Warning: unknown new rules activated (versionbit %d)", bit)
		// 			msg.SetMiscWarning(strWaring)
		// 			if !gWarned {
		// 				AlertNotify(strWaring)
		// 				gWarned = true
		// 			}
		// 		} else {
		// 			warningMessages = append(warningMessages,
		// 				fmt.Sprintf("unknown new rules are about to activate (versionbit %d)", bit))
		// 		}
		// 	}
		// }
		// // Check the version of the last 100 blocks to see if we need to
		// // upgrade:
		// for i := 0; i < 100 && index != nil; i++ {
		// 	nExpectedVersion := ComputeBlockVersion(index.Prev, param, VBCache)
		// 	if index.Header.Version > VersionBitsLastOldBlockVersion &&
		// 		(int(index.Header.Version)&(^nExpectedVersion) != 0) {
		// 		nUpgraded++
		// 		index = index.Prev
		// 	}
		// }
		// if nUpgraded > 0 {
		// 	warningMessages = append(warningMessages,
		// 		fmt.Sprintf("%d of last 100 blocks have unexpected version", nUpgraded))
		// }
		// if nUpgraded > 100/2 {
		// 	strWarning := fmt.Sprintf("Warning: Unknown block versions being mined!" +
		// 		" It's possible unknown rules are in effect")
		// 	// notify GetWarnings(), called by Qt and the JSON-RPC code to warn
		// 	// the user:
		// 	msg.SetMiscWarning(strWarning)
		// 	if !gWarned {
		// 		AlertNotify(strWarning)
		//
		// 		gWarned = true
		// 	}
		// }
	}
	txdata := param.TxData()
	tip := mchain.GetInstance().Tip()
	utxoTip := utxo.GetUtxoCacheInstance()
	tipHash := tip.GetBlockHash()
	log.Info("new best=%s height=%d version=0x%08x work=%.8g tx=%lu "+
		"date='%s' progress=%f cache=%.1f(%utxo)", tipHash.String(),
		tip.Height, tip.Header.Version,
		tip.ChainWork.String(), tip.ChainTxCount,
		time.Unix(int64(tip.Header.Time), 0).String(),
		GuessVerificationProgress(txdata, tip),
		utxoTip.DynamicMemoryUsage(), utxoTip.GetCacheSize())
	if len(warningMessages) != 0 {
		log.Info("waring= %s", strings.Join(warningMessages, ","))
	}
}

// GuessVerificationProgress Guess how far we are in the verification process at the given block index
func GuessVerificationProgress(data *chainparams.ChainTxData, index *blockindex.BlockIndex) float64 {
	if index == nil {
		return float64(0)
	}

	now := time.Now()

	var txTotal float64
	// todo confirm time precise
	if int64(index.ChainTxCount) <= data.TxCount {
		txTotal = float64(data.TxCount) + (now.Sub(data.Time).Seconds())*data.TxRate
	} else {
		txTotal = float64(index.ChainTxCount) + float64(now.Second()-int(index.GetBlockTime()))*data.TxRate
	}
	return float64(index.ChainTxCount) / txTotal
}

func DisconnectBlock(pblock *block.Block, pindex *blockindex.BlockIndex, view *utxo.CoinsMap) mUndo.DisconnectResult {

	hashA := pindex.GetBlockHash()
	hashB := utxo.GetUtxoCacheInstance().GetBestBlock()
	if !bytes.Equal(hashA[:], hashB[:]) {
		panic("the two hash should be equal ...")
	}
	pos := pindex.GetUndoPos()
	if pos.IsNull() {
		log.Error("DisconnectBlock(): no undo data available")
		return mUndo.DisconnectFailed
	}
	blockUndo, ret := disk.UndoReadFromDisk(&pos, *pindex.Prev.GetBlockHash())
	if !ret {
		log.Error("DisconnectBlock(): failure reading undo data")
		return mUndo.DisconnectFailed
	}

	return undo.ApplyBlockUndo(blockUndo, pblock, view)
}


// ReceivedBlockTransactions Mark a block as having its data received and checked (up to
// * BLOCK_VALID_TRANSACTIONS).
func ReceivedBlockTransactions(pblock *block.Block,
	pindexNew *blockindex.BlockIndex, pos *block.DiskBlockPos) bool {
	hash := pindexNew.GetBlockHash()
	pindexNew.TxCount = len(pblock.Txs)
	pindexNew.ChainTxCount = 0
	pindexNew.File = pos.File
	pindexNew.DataPos = pos.Pos
	pindexNew.UndoPos = 0
	pindexNew.AddStatus(blockindex.StatusDataStored)
	gPersist := global.GetInstance()
	gPersist.AddDirtyBlockIndex(*hash, pindexNew)
	gChain := mchain.GetInstance()
	if pindexNew.IsGenesis() || gChain.ParentInBranch(pindexNew) {
		// If indexNew is the genesis block or all parents are in branch
		gChain.AddToBranch(pindexNew)
	} else {
		if !pindexNew.IsGenesis() && pindexNew.Prev.IsValid(blockindex.BlockValidTree) {
			gChain.AddToOrphan(pindexNew)
		}
	}

	return true
}
