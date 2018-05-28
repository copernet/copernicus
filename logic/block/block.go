package block

import (
	
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/logic/merkleroot"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/tx"
	ltx "github.com/btcboost/copernicus/logic/tx"
	
	"github.com/btcboost/copernicus/persist/disk"
	"github.com/btcboost/copernicus/util"
)

func Check(b *block.Block) error {
	return nil
}

func GetBlock(hash *util.Hash) (* block.Block, error) {
	return nil,nil
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

func WriteToFile(bi *blockindex.BlockIndex, b *block.Block) error {
	return nil
}



func CheckBlock(params *chainparams.BitcoinParams, pblock *block.Block, state *block.ValidationState,
	checkPOW, checkMerkleRoot bool, nHeight int32, nLocktime int64, nMaxBlockSigOps uint64) bool {

	// These are checks that are independent of context.
	if pblock.Checked {
		return true
	}
	bh := pblock.Header
	// Check that the header is valid (particularly PoW).  This is mostly
	// redundant with the call in AcceptBlockHeader.
	if err := CheckBlockHeader(&bh, params, checkPOW); err!=nil {
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
	err := ltx.CheckBlockTransactions(pblock.Txs, nHeight, nLocktime, nMaxBlockSigOps)
	if err != nil{
		return state.Dos(100, false, block.RejectInvalid, "bad-blk-tx",
			false, "CheckBlockTransactions failed")
	}
	if checkPOW && checkMerkleRoot {
		pblock.Checked = true
	}

	return true
}
