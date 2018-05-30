package block

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
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
	
	"github.com/btcboost/copernicus/persist/disk"
	"github.com/btcboost/copernicus/util"
)


func GetBlock(hash *util.Hash) (* block.Block, error) {
	return nil,nil
}
func Check(b *block.Block) error{
	return nil
}

func WriteBlockToDisk(bi *blockindex.BlockIndex, bl *block.Block)(*block.DiskBlockPos,error) {
	
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



func getLockTime(block *block.Block, indexPrev *blockindex.BlockIndex) (int64) {
	
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

func CheckBlock( pblock *block.Block, state *block.ValidationState,
	checkPOW, checkMerkleRoot bool) bool {
	// These are checks that are independent of context.
	if pblock.Checked {
		return true
	}
	blockSize := pblock.EncodeSize()
	nMaxBlockSigOps := consensus.GetMaxBlockSigOpsCount(uint64(blockSize))
	bh := pblock.Header
	// Check that the header is valid (particularly PoW).  This is mostly
	// redundant with the call in AcceptBlockHeader.
	if err := CheckBlockHeader(&bh, checkPOW); err!=nil {
		return false
	}

	// Check the merkle root.
	if checkMerkleRoot {
		mutated := false
		hashMerkleRoot2 := merkleroot.BlockMerkleRoot(pblock, &mutated)
		if !bh.MerkleRoot.IsEqual(&hashMerkleRoot2) {
			return state.Dos(100, false, block.RejectInvalid, "bad-txnMrklRoot",
				true, "hashMerkleRoot mismatch")
		}

		// Check for merkle tree malleability (CVE-2012-2459): repeating
		// sequences of transactions in a block without affecting the merkle
		// root of a block, while still invalidating it.
		if mutated {
			return state.Dos(100, false, block.RejectInvalid, "bad-txns-duplicate",
				true, "duplicate transaction")
		}
	}

	// All potential-corruption validation must be done before we do any
	// transaction validation, as otherwise we may mark the header as invalid
	// because we receive the wrong transactions for it.

	// First transaction must be coinBase.
	if len(pblock.Txs) == 0 {
		return state.Dos(100, false, block.RejectInvalid, "bad-cb-missing",
			false, "first tx is not coinBase")
	}

	// size limits
	nMaxBlockSize := consensus.DefaultMaxBlockSize

	// Bail early if there is no way this block is of reasonable size.
	minTransactionSize := tx.NewEmptyTx().EncodeSize()
	if len(pblock.Txs) * int(minTransactionSize) > int(nMaxBlockSize) {
		return state.Dos(100, false, block.RejectInvalid, "bad-blk-length",
			false, "size limits failed")
	}

	currentBlockSize := pblock.EncodeSize()
	if currentBlockSize > int(nMaxBlockSize) {
		return state.Dos(100, false, block.RejectInvalid, "bad-blk-length",
			false, "size limits failed")
	}

	err := ltx.CheckBlockTransactions(pblock.Txs, nMaxBlockSigOps)
	if err != nil{
		return state.Dos(100, false, block.RejectInvalid, "bad-blk-tx",
			false, "CheckBlockTransactions failed")
	}
	if checkPOW && checkMerkleRoot {
		pblock.Checked = true
	}

	return true
}

func ContextualCheckBlock( b *block.Block, state *block.ValidationState,
	indexPrev *blockindex.BlockIndex) bool {

	var height int32
	if indexPrev != nil {
		height = indexPrev.Height + 1
	}
	lockTimeCutoff := getLockTime(b, indexPrev)

	// Check that all transactions are finalized
	// Enforce rule that the coinBase starts with serialized block height
	if err := ltx.CheckBlockContextureTransactions(b.Txs, height, lockTimeCutoff);err!=nil {
		return false
	}
	return true
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
	gChain := chain.GetInstance()
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


// GetBlockScriptFlags Returns the script flags which should be checked for a given block
func GetBlockScriptFlags(pindex *blockindex.BlockIndex) uint32 {
	gChain := chain.GetInstance()
	return gChain.GetBlockScriptFlags(pindex)
}

func IsCashHFEnabled(params *chainparams.BitcoinParams, medianTimePast int64) bool {
	return params.CashHardForkActivationTime <= medianTimePast
}