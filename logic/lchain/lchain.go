package lchain

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"

	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/logic/lundo"

	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/disk"

	"github.com/copernet/copernicus/logic/lblock"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/pow"
)

// IsInitialBlockDownload Check whether we are doing an initial block download
// (synchronizing from disk or network)
func IsInitialBlockDownload() bool {
	return persist.Reindex || !chain.GetInstance().IsAlmostSynced()
}

func ConnectBlock(pblock *block.Block, pindex *blockindex.BlockIndex, view *utxo.CoinsMap, fJustCheck bool) error {
	gChain := chain.GetInstance()
	tip := gChain.Tip()
	start := time.Now()
	params := gChain.GetParams()
	// Check it again in case a previous version let a bad lblock in
	if err := lblock.CheckBlock(pblock, true, true); err != nil {
		return err
	}

	// Verify that the view's current state corresponds to the previous lblock
	var hashPrevBlock *util.Hash
	if pindex.Prev == nil {
		hashPrevBlock = &util.Hash{}
	} else {
		hashPrevBlock = pindex.Prev.GetBlockHash()
	}
	gUtxo := utxo.GetUtxoCacheInstance()
	bestHash, _ := gUtxo.GetBestBlock()
	log.Debug("bestHash = %s, hashPrevBloc = %s", bestHash.String(), hashPrevBlock)
	if !hashPrevBlock.IsEqual(&bestHash) {
		log.Debug("will panic in ConnectBlock()")
		panic("error: hashPrevBlock not equal view.GetBestBlock()")
	}

	// Special case for the genesis lblock, skipping connection of its
	// transactions (its coinbase is unspendable)
	blockHash := pblock.GetHash()
	if blockHash.IsEqual(params.GenesisHash) {
		if !fJustCheck {
			//view.SetBestBlock(*pindex.GetBlockHash())
		}
		return nil
	}

	fScriptChecks := true
	if chain.HashAssumeValid != util.HashZero {
		// We've been configured with the hash of a block which has been
		// externally verified to have a valid history. A suitable default value
		// is included with the software and updated from time to time. Because
		// validity relative to a piece of software is an objective fact these
		// defaults can be easily reviewed. This setting doesn't force the
		// selection of any particular chain but makes validating some faster by
		// effectively caching the result of part of the verification.
		if bi := gChain.FindBlockIndex(chain.HashAssumeValid); bi != nil {
			if bi.GetAncestor(pindex.Height) == pindex && tip.GetAncestor(pindex.Height) == pindex &&
				tip.ChainWork.Cmp(pow.HashToBig(&params.MinimumChainWork)) > 0 {
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

	time1 := time.Now()
	gPersist := persist.GetInstance()
	gPersist.GlobalTimeCheck += time1.Sub(start)
	log.Print("bench", "debug", " - Sanity checks: current %v [total %v]",
		time1.Sub(start), gPersist.GlobalTimeCheck)

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
	//zHash := util.HashZero
	//fEnforceBIP30 := (!blockHash.IsEqual(&zHash)) ||
	//	!((pindex.Height == 91842 &&
	//		blockHash.IsEqual(util.HashFromString("0x00000000000a4d0a398161ffc163c503763b1f4360639393e0e4c8e300e0caec"))) ||
	//		(pindex.Height == 91880 &&
	//			blockHash.IsEqual(util.HashFromString("0x00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721"))))
	bip30Enable := !((pindex.Height == 91842 && blockHash.IsEqual(util.HashFromString("00000000000a4d0a398161ffc163c503763b1f4360639393e0e4c8e300e0caec"))) ||
		(pindex.Height == 91880 && blockHash.IsEqual(util.HashFromString("00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721"))))

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
	bip34Enable := pindexBIP34height != nil && pindexBIP34height.GetBlockHash().IsEqual(&params.BIP34Hash)
	bip30Enable = bip30Enable && !bip34Enable

	flags := lblock.GetBlockScriptFlags(pindex.Prev)
	blockSubSidy := lblock.GetBlockSubsidy(pindex.Height, params)
	time2 := time.Now()
	gPersist.GlobalTimeForks += time2.Sub(time1)
	log.Print("bench", "debug", " - Fork checks: current %v [total %v]",
		time2.Sub(time1), gPersist.GlobalTimeForks)

	var coinsMap, blockUndo, err = ltx.ApplyBlockTransactions(pblock.Txs, bip30Enable, flags,
		fScriptChecks, blockSubSidy, pindex.Height, consensus.GetMaxBlockSigOpsCount(uint64(pblock.EncodeSize())))
	if err != nil {
		return err
	}
	// Write undo information to disk
	UndoPos := pindex.GetUndoPos()
	if UndoPos.IsNull() || !pindex.IsValid(blockindex.BlockValidScripts) {
		if UndoPos.IsNull() {
			pos := block.NewDiskBlockPos(pindex.File, 0)
			//blockUndo size + hash size + 4bytes len
			if err := disk.FindUndoPos(pindex.File, pos, blockUndo.SerializeSize()+36); err != nil {
				return err
			}
			if err := disk.UndoWriteToDisk(blockUndo, pos, *pindex.Prev.GetBlockHash(), params.BitcoinNet); err != nil {
				return err
			}

			// update nUndoPos in block index
			pindex.UndoPos = pos.Pos
			pindex.AddStatus(blockindex.BlockHaveUndo)
		}
		pindex.RaiseValidity(blockindex.BlockValidScripts)
		gPersist.AddDirtyBlockIndex(pindex)
	}
	// add this block to the view's block chain
	*view = *coinsMap

	//if (pindex.IsReplayProtectionEnabled(params) &&
	//	!pindex.Prev.IsReplayProtectionEnabled(params)) {
	//	lmempool.clear();
	//}

	log.Debug("Connect block heigh:%d, hash:%s", pindex.Height, blockHash.String())
	return nil
}

//InvalidBlockFound the found block is invalid
func InvalidBlockFound(pindex *blockindex.BlockIndex) {
	pindex.AddStatus(blockindex.BlockFailed)
	gChain := chain.GetInstance()
	gChain.RemoveFromBranch(pindex)
	gPersist := persist.GetInstance()
	gPersist.AddDirtyBlockIndex(pindex)
}

type connectTrace map[*blockindex.BlockIndex]*block.Block

// ConnectTip Connect a new block to chainActive. block is either nullptr or a pointer to
// a CBlock corresponding to indexNew, to bypass loading it again from disk.
// The block is always added to connectTrace (either after loading from disk or
// by copying block) - if that is not intended, care must be taken to remove
// the last entry in blocksConnected in case of failure.
func ConnectTip(pIndexNew *blockindex.BlockIndex,
	block *block.Block, connTrace connectTrace) error {
	gChain := chain.GetInstance()
	tip := gChain.Tip()

	if pIndexNew.Prev != tip {
		log.Error("error: try to connect to inactive chain!!!")
		panic("error: try to connect to inactive chain!!!")
	}
	// Read block from disk.
	nTime1 := time.Now().UnixNano()
	if block == nil {
		blockNew, err := disk.ReadBlockFromDisk(pIndexNew, gChain.GetParams())
		if !err || blockNew == nil {
			log.Error("error: FailedToReadBlock: %v", err)
			return errcode.New(errcode.FailedToReadBlock)
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
	gPersist := persist.GetInstance()
	gPersist.GlobalTimeReadFromDisk += nTime2 - nTime1
	log.Info("Load block from disk: %#v ms total: [%#v s]\n", (nTime2-nTime1)/1000, gPersist.GlobalTimeReadFromDisk/1000000)

	view := utxo.NewEmptyCoinsMap()
	err := ConnectBlock(blockConnecting, pIndexNew, view, false)
	if err != nil {
		InvalidBlockFound(pIndexNew)
		log.Error("ConnectTip(): ConnectBlock %s failed, err:%v", indexHash.String(), err)
		return err
	}

	nTime3 := util.GetMicrosTime()
	gPersist.GlobalTimeConnectTotal += nTime3 - nTime2
	log.Debug("Connect total: %.2fms [%.2fs]\n",
		float64(nTime3-nTime2)*0.001, float64(gPersist.GlobalTimeConnectTotal)*0.000001)
	//flushed := view.Flush(indexHash)
	err = utxo.GetUtxoCacheInstance().UpdateCoins(view, &indexHash)
	if err != nil {
		panic("here should be true when view flush state")
	}
	nTime4 := util.GetMicrosTime()
	gPersist.GlobalTimeFlush += nTime4 - nTime3
	log.Print("bench", "debug", " - Flush: %.2fms [%.2fs]\n",
		float64(nTime4-nTime3)*0.001, float64(gPersist.GlobalTimeFlush)*0.000001)
	// Write the chain state to disk, if necessary.
	if err := disk.FlushStateToDisk(disk.FlushStateAlways, 0); err != nil {
		return err
	}

	if pIndexNew.Height >= conf.Cfg.Chain.StartLogHeight {
		var stat stat
		if err := GetUTXOStats(utxo.GetUtxoCacheInstance().(*utxo.CoinsLruCache).GetCoinsDB(), &stat); err != nil {
			log.Debug("GetUTXOStats() failed with : %s", err)
			return err
		}
		f, err := os.OpenFile(filepath.Join(conf.DataDir, "utxo.log"), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0640)
		if err != nil {
			log.Debug("os.OpenFile() failed with : %s", err)
			return err
		}
		defer f.Close()
		if _, err := f.WriteString(stat.String()); err != nil {
			log.Debug("f.WriteString() failed with : %s", err)
			return err
		}
	}

	nTime5 := util.GetMicrosTime()
	gPersist.GlobalTimeChainState += nTime5 - nTime4
	log.Print("bench", "debug", " - Writing chainstate: %.2fms [%.2fs]\n",
		float64(nTime5-nTime4)*0.001, float64(gPersist.GlobalTimeChainState)*0.000001)
	// Remove conflicting transactions from the mempool.;
	mempool.GetInstance().RemoveTxSelf(blockConnecting.Txs)
	// Update chainActive & related variables.
	UpdateTip(pIndexNew)
	nTime6 := util.GetMicrosTime()
	gPersist.GlobalTimePostConnect += nTime6 - nTime5
	gPersist.GlobalTimeTotal += nTime6 - nTime1
	log.Print("bench", "debug", " - Connect postprocess: %.2fms [%.2fs]\n",
		float64(nTime6-nTime5)*0.001, float64(gPersist.GlobalTimePostConnect)*0.000001)
	log.Print("bench", "debug", " - Connect block: %.2fms [%.2fs]\n",
		float64(nTime6-nTime1)*0.001, float64(gPersist.GlobalTimeTotal)*0.000001)

	return nil
}

// DisconnectTip Disconnect chainActive's tip. You probably want to call
// mempool.removeForReorg and manually re-limit mempool size after this, with
// cs_main held.
func DisconnectTip(fBare bool) error {
	gChain := chain.GetInstance()
	tip := gChain.Tip()
	if tip == nil {
		panic("the chain tip element should not equal nil")
	}
	// Read block from disk.
	blk, ret := disk.ReadBlockFromDisk(tip, gChain.GetParams())
	if !ret {
		log.Debug("FailedToReadBlock")
		return errcode.New(errcode.FailedToReadBlock)
	}

	// Apply the block atomically to the chain state.
	nStart := time.Now().UnixNano()
	{
		view := utxo.NewEmptyCoinsMap()

		if DisconnectBlock(blk, tip, view) != undo.DisconnectOk {
			hash := tip.GetBlockHash()
			log.Error(fmt.Sprintf("DisconnectTip(): DisconnectBlock %s failed ", hash.String()))
			return errcode.New(errcode.DisconnectTipUndoFailed)
		}
		//flushed := view.Flush(blk.Header.HashPrevBlock)
		err := utxo.GetUtxoCacheInstance().UpdateCoins(view, &blk.Header.HashPrevBlock)
		if err != nil {
			panic("view flush error !!!")
		}

	}
	// replace implement with log.Print(in C++).
	log.Info("bench-debug - Disconnect block : %.2fms\n",
		float64(time.Now().UnixNano()-nStart)*0.001)

	// Write the chain state to disk, if necessary.
	if err := disk.FlushStateToDisk(disk.FlushStateIfNeeded, 0); err != nil {
		return err
	}

	if !fBare {
		// Resurrect mempool transactions from the disconnected block.
		for _, tx := range blk.Txs {
			// ignore validation errors in resurrected transactions
			if tx.IsCoinBase() {
				mempool.GetInstance().RemoveTxRecursive(tx, mempool.REORG)
			} else {
				e := lmempool.AcceptTxToMemPool(tx)
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
	// If this block was deactivating the replay protection, then we need to
	// remove transactions that are replay protected from the mempool. There is
	// no easy way to do this so we'll just discard the whole mempool and then
	// add the transaction of the block we just disconnected back.
	//
	// Samewise, if this block enabled the monolith opcodes, then we need to
	// clear the mempool of any transaction using them.
	//if ((IsReplayProtectionEnabled(config, pindexDelete) &&
	//	!IsReplayProtectionEnabled(config, pindexDelete->pprev)) ||
	//	(IsMonolithEnabled(config, pindexDelete) &&
	//		!IsMonolithEnabled(config, pindexDelete->pprev))) {
	//	lmempool.clear();
	//	// While not strictly necessary, clearing the disconnect pool is also
	//	// beneficial so we don't try to reuse its content at the end of the
	//	// reorg, which we know will fail.
	//	if (disconnectpool) {
	//		disconnectpool->clear();
	//	}
	//}
	// Update chainActive and related variables.
	UpdateTip(tip.Prev)
	// Let wallets know transactions went from 1-confirmed to
	// 0-confirmed or conflicted:

	return nil
}

// UpdateTip Update chainActive and related internal data structures.
func UpdateTip(pindexNew *blockindex.BlockIndex) {
	gChain := chain.GetInstance()
	gChain.SetTip(pindexNew)
	param := gChain.GetParams()
	//	TODO !!! notify mempool update tx
	warningMessages := make([]string, 0)
	// if !lundo.IsInitialBlockDownload() {
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
	// }
	txdata := param.TxData()
	tip := chain.GetInstance().Tip()
	utxoTip := utxo.GetUtxoCacheInstance()
	tipHash := tip.GetBlockHash()
	log.Info("new best=%s height=%d version=0x%08x work=%s tx=%d "+
		"date='%s' progress=%f memory=%d(cache=%d)", tipHash.String(),
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
func GuessVerificationProgress(data *model.ChainTxData, index *blockindex.BlockIndex) float64 {
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

func DisconnectBlock(pblock *block.Block, pindex *blockindex.BlockIndex, view *utxo.CoinsMap) undo.DisconnectResult {

	hashA := pindex.GetBlockHash()
	hashB, _ := utxo.GetUtxoCacheInstance().GetBestBlock()
	if !bytes.Equal(hashA[:], hashB[:]) {
		panic("the two hash should be equal ...")
	}
	pos := pindex.GetUndoPos()
	if pos.IsNull() {
		log.Error("DisconnectBlock(): no undo data available.")
		return undo.DisconnectFailed
	}
	blockUndo, ret := disk.UndoReadFromDisk(&pos, *pindex.Prev.GetBlockHash())
	if !ret {
		log.Error("DisconnectBlock(): failure reading undo data")
		return undo.DisconnectFailed
	}

	return lundo.ApplyBlockUndo(blockUndo, pblock, view)
}

func InitGenesisChain() error {
	gChain := chain.GetInstance()
	if gChain.Genesis() != nil {
		return nil
	}

	// Write genesisblock to disk
	bl := gChain.GetParams().GenesisBlock
	pos := block.NewDiskBlockPos(0, 0)
	flag := disk.FindBlockPos(pos, uint32(bl.SerializeSize()+4), 0, uint64(bl.GetBlockHeader().Time), false)
	if !flag {
		log.Error("InitChain.WriteBlockToDisk():FindBlockPos failed")
		return errcode.ProjectError{Code: 2000}
	}
	flag = disk.WriteBlockToDisk(bl, pos)
	if !flag {
		log.Error("InitChain.WriteBlockToDisk():WriteBlockToDisk failed")
		return errcode.ProjectError{Code: 2001}
	}

	// Add genesis block index to DB and make the Chain
	bIndex := blockindex.NewBlockIndex(&bl.Header)
	bIndex.Height = 0
	err := gChain.AddToIndexMap(bIndex)
	if err != nil {
		return err
	}
	lblock.ReceivedBlockTransactions(bl, bIndex, pos)
	gChain.SetTip(bIndex)

	// Set bestblockhash to DB
	coinsMap := utxo.NewEmptyCoinsMap()
	//coinsMap, _, _ := ltx.ApplyGeniusBlockTransactions(bl.Txs)
	bestHash := bIndex.GetBlockHash()
	utxo.GetUtxoCacheInstance().UpdateCoins(coinsMap, bestHash)

	err = disk.FlushStateToDisk(disk.FlushStateAlways, 0)

	return err
}
