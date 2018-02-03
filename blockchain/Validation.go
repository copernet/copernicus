package blockchain

import (
	"bytes"

	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/msg"
)

const (
	/*DEFAULT_PERMIT_BAREMULTISIG  Default for -permitbaremultisig */
	DEFAULT_PERMIT_BAREMULTISIG bool = true
	DEFAULT_CHECKPOINTS_ENABLED bool = true
	DEFAULT_TXINDEX             bool = false
	DEFAULT_BANSCORE_THRESHOLD  uint = 100
)

//IsUAHFenabled Check is UAHF has activated.
func IsUAHFenabled(params *msg.BitcoinParams, height int) bool {
	return height >= params.UAHFHeight
}

func IsCashHFEnabled(params *msg.BitcoinParams, medianTimePast int64) bool {
	return params.CashHardForkActivationTime <= medianTimePast
}

func ContextualCheckTransaction(params *msg.BitcoinParams, tx *model.Tx, state *model.ValidationState, height int, lockTimeCutoff int64) bool {

	if !tx.IsFinalTx(height, lockTimeCutoff) {
		return state.Dos(10, false, model.REJECT_INVALID, "bad-txns-nonfinal", false, "non-final transaction")
	}

	if IsUAHFenabled(params, height) && height <= params.AntiReplayOpReturnSunsetHeight {
		for _, txo := range tx.Outs {
			if txo.Script.IsCommitment(params.AntiReplayOpReturnCommitment) {
				return state.Dos(10, false, model.REJECT_INVALID, "bad-txn-replay", false, "non playable transaction")
			}
		}
	}

	return true
}

func ContextualCheckBlock(params *msg.BitcoinParams, block *model.Block, state *model.ValidationState, pindexPrev *BlockIndex) bool {
	nHeight := pindexPrev.Height + 1
	if pindexPrev == nil {
		nHeight = 0
	}

	//！todo : Add VersionBitsState() test in here
	nLockTimeFlags := 0
	//if version

	medianTimePast := pindexPrev.GetMedianTimePast()
	if pindexPrev == nil {
		medianTimePast = 0
	}

	lockTimeCutoff := int64(block.BlockHeader.GetBlockTime())
	if nLockTimeFlags&consensus.LocktimeMedianTimePast != 0 {
		lockTimeCutoff = medianTimePast
	}

	// Check that all transactions are finalized
	for _, tx := range block.Transactions {
		if !ContextualCheckTransaction(params, tx, state, nHeight, lockTimeCutoff) {
			return false
		}
	}

	// Enforce rule that the coinbase starts with serialized block height
	expect := model.Script{}
	if nHeight >= params.BIP34Height {
		expect.PushInt64(int64(nHeight))
		if block.Transactions[0].Ins[0].Script.Size() < expect.Size() ||
			bytes.Equal(expect.GetScriptByte(), block.Transactions[0].Ins[0].Script.GetScriptByte()[:len(expect.GetScriptByte())]) {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-cb-height", false, "block height mismatch in coinbase")
		}
	}

	return true
}

func CheckBlockHeader(block *model.Block, state *model.ValidationState, params *msg.BitcoinParams, fCheckPOW bool) bool {
	// Check proof of work matches claimed amount

	return true
}

func CheckBlock(params *msg.BitcoinParams, block *model.Block, state *model.ValidationState, fCheckPOW, fCheckMerkleRoot bool) bool {
	// These are checks that are independent of context.
	//if block.FChecked {
	//	return true
	//}

	// Check that the header is valid (particularly PoW).  This is mostly
	// redundant with the call in AcceptBlockHeader.
	//CheckBlockHeader()

	return true
}
