package block

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	lbi "github.com/btcboost/copernicus/logic/blockindex"
	"github.com/btcboost/copernicus/logic/merkleroot"
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/versionbits"
	"github.com/btcboost/copernicus/persist/global"
	"github.com/btcboost/copernicus/util/amount"

	"github.com/btcboost/copernicus/persist/disk"
	"github.com/btcboost/copernicus/util"
)

const MinBlocksToKeep = int32(288)

func GetBlock(hash *util.Hash) (*block.Block, error) {
	return nil, nil
}

func WriteBlockToDisk(bi *blockindex.BlockIndex, bl *block.Block) (*block.DiskBlockPos, error) {

	height := bi.Height
	pos := block.NewDiskBlockPos(0, 0)
	flag := disk.FindBlockPos(pos, uint32(bl.SerializeSize()), height, uint64(bl.GetBlockHeader().Time), false)
	if !flag {
		log.Error("WriteBlockToDisk():FindBlockPos failed")
		return nil, errcode.ProjectError{Code: 2000}
	}

	flag = disk.WriteBlockToDisk(bl, pos)
	if !flag {
		log.Error("WriteBlockToDisk():WriteBlockToDisk failed")
		return nil, errcode.ProjectError{Code: 2001}
	}
	return pos, nil
}

func getLockTime(block *block.Block, indexPrev *blockindex.BlockIndex) int64 {

	params := chain.GetInstance().GetParams()
	lockTimeFlags := 0
	if versionbits.VersionBitsState(indexPrev, params, consensus.DeploymentCSV, versionbits.VBCache) == versionbits.ThresholdActive {
		lockTimeFlags |= consensus.LocktimeMedianTimePast
	}

	medianTimePast := indexPrev.GetMedianTimePast()
	if indexPrev == nil {
		medianTimePast = 0
	}
	bh := block.Header
	lockTimeCutoff := int64(bh.GetBlockTime())
	if lockTimeFlags&consensus.LocktimeMedianTimePast != 0 {
		lockTimeCutoff = medianTimePast
	}

	return lockTimeCutoff
}

func CheckBlock(pblock *block.Block) error {
	// These are checks that are independent of context.
	if pblock.Checked {
		return nil
	}
	blockSize := pblock.EncodeSize()
	nMaxBlockSigOps := consensus.GetMaxBlockSigOpsCount(uint64(blockSize))
	bh := pblock.Header
	// Check that the header is valid (particularly PoW).  This is mostly
	// redundant with the call in AcceptBlockHeader.
	if err := CheckBlockHeader(&bh); err != nil {
		return err
	}

	// Check the merkle root.
	mutated := false
	hashMerkleRoot2 := merkleroot.BlockMerkleRoot(pblock, &mutated)
	if !bh.MerkleRoot.IsEqual(&hashMerkleRoot2) {
		return errcode.New(errcode.ErrorBadTxnMrklRoot)
	}

	// Check for merkle tree malleability (CVE-2012-2459): repeating
	// sequences of transactions in a block without affecting the merkle
	// root of a block, while still invalidating it.
	if mutated {
		return errcode.New(errcode.ErrorbadTxnsDuplicate)
	}

	// All potential-corruption validation must be done before we do any
	// transaction validation, as otherwise we may mark the header as invalid
	// because we receive the wrong transactions for it.

	// First transaction must be coinBase.
	if len(pblock.Txs) == 0 {
		return errcode.New(errcode.ErrorBadCoinBaseMissing)
	}

	// size limits
	nMaxBlockSize := consensus.DefaultMaxBlockSize

	// Bail early if there is no way this block is of reasonable size.
	minTransactionSize := tx.NewEmptyTx().EncodeSize()
	if len(pblock.Txs)*int(minTransactionSize) > int(nMaxBlockSize) {
		return errcode.New(errcode.ErrorBadBlkLength)
	}

	currentBlockSize := pblock.EncodeSize()
	if currentBlockSize > int(nMaxBlockSize) {
		return errcode.New(errcode.ErrorBadBlkTxSize)
	}

	err := ltx.CheckBlockTransactions(pblock.Txs, nMaxBlockSigOps)
	if err != nil {
		return errcode.New(errcode.ErrorBadBlkTx)
	}
	pblock.Checked = true
	return nil
}

func ContextualCheckBlock(b *block.Block, indexPrev *blockindex.BlockIndex) error {

	var height int32
	if indexPrev != nil {
		height = indexPrev.Height + 1
	}
	lockTimeCutoff := getLockTime(b, indexPrev)

	// Check that all transactions are finalized
	// Enforce rule that the coinBase starts with serialized block height
	if err := ltx.CheckBlockContextureTransactions(b.Txs, height, lockTimeCutoff); err != nil {
		return err
	}
	return nil
}

// ReceivedBlockTransactions Mark a block as having its data received and checked (up to
// * BLOCK_VALID_TRANSACTIONS).
func ReceivedBlockTransactions(pblock *block.Block,
	pindexNew *blockindex.BlockIndex, pos *block.DiskBlockPos) bool {
	hash := pindexNew.GetBlockHash()
	pindexNew.TxCount = int32(len(pblock.Txs))
	pindexNew.ChainTxCount = 0
	pindexNew.File = pos.File
	pindexNew.DataPos = pos.Pos
	pindexNew.UndoPos = 0
	pindexNew.AddStatus(blockindex.StatusDataStored)
	gPersist := global.GetInstance()
	gPersist.AddDirtyBlockIndex(*hash, pindexNew)
	gChain := chain.GetInstance()
	if pindexNew.IsGenesis(gChain.GetParams()) || gChain.ParentInBranch(pindexNew) {
		// If indexNew is the genesis block or all parents are in branch
		gChain.AddToBranch(pindexNew)
	} else {
		if !pindexNew.IsGenesis(gChain.GetParams()) && pindexNew.Prev.IsValid(blockindex.BlockValidTree) {
			gChain.AddToOrphan(pindexNew)
		}
	}

	return true
}

// GetBlockScriptFlags Returns the script flags which should be checked for a given block
func GetBlockScriptFlags(pindex *blockindex.BlockIndex) uint32 {
	gChain := chain.GetInstance()
	return gChain.GetBlockScriptFlags(pindex)
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

func AcceptBlock(pblock *block.Block,
	fRequested bool, fNewBlock *bool) (bIndex *blockindex.BlockIndex, dbp *block.DiskBlockPos, err error) {
	if pblock != nil {
		*fNewBlock = false
	}
	bIndex, err = AcceptBlockHeader(&pblock.Header)
	if err != nil {
		return
	}
	log.Info(bIndex)

	if bIndex.Accepted() {
		err = errcode.ProjectError{Code: 3009}

		return
	}
	if !fRequested {
		tip := chain.GetInstance().Tip()
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
		err = CheckBlock(pblock)
		if err != nil {
			return
		}

		bIndex.AddStatus(blockindex.StatusAllValid)
	}
	gPersist := global.GetInstance()
	if err = CheckBlock(pblock); err != nil {
		bIndex.AddStatus(blockindex.StatusFailed)
		gPersist.AddDirtyBlockIndex(pblock.GetHash(), bIndex)
		return
	}
	if err = ContextualCheckBlock(pblock, bIndex.Prev); err != nil {
		bIndex.AddStatus(blockindex.StatusFailed)
		gPersist.AddDirtyBlockIndex(pblock.GetHash(), bIndex)
		return
	}
	*fNewBlock = true

	dbp, err = WriteBlockToDisk(bIndex, pblock)
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

func AcceptBlockHeader(bh *block.BlockHeader) (*blockindex.BlockIndex, error) {
	var c = chain.GetInstance()

	bIndex := c.FindBlockIndex(bh.GetHash())
	if bIndex != nil {
		return bIndex, nil
	}

	//this is a new blockheader
	err := CheckBlockHeader(bh)
	if err != nil {
		return nil, err
	}
	gChain := chain.GetInstance()

	bIndex = blockindex.NewBlockIndex(bh)
	if !bIndex.IsGenesis(gChain.GetParams()) {

		bIndex.Prev = c.FindBlockIndex(bh.HashPrevBlock)
		if bIndex.Prev == nil {
			return nil, errcode.New(errcode.ErrorBlockHeaderNoParent)
		}
		if !lbi.CheckIndexAgainstCheckpoint(bIndex.Prev) {
			return nil, errcode.ProjectError{Code: 3100}
		}
		if !ContextualCheckBlockHeader(bh, bIndex.Prev, util.GetAdjustedTime()) {
			return nil, errcode.ProjectError{Code: 3101}
		}
	}

	bIndex.Height = bIndex.Prev.Height + 1
	bIndex.TimeMax = util.MaxU32(bIndex.Prev.TimeMax, bIndex.Header.GetBlockTime())
	bIndex.AddStatus(blockindex.StatusWaitingData)

	err = c.AddToIndexMap(bIndex)
	if err != nil {
		return nil, err
	}
	return bIndex, nil
}
