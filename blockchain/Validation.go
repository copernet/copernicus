package blockchain

import (
	"bytes"
	"fmt"

	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/msg"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

const (
	// DEFAULT_PERMIT_BAREMULTISIG  Default for -permitbaremultisig
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

	nLockTimeFlags := 0
	if VersionBitsState(pindexPrev, params, msg.DEPLOYMENT_CSV, &versionBitsCache) == THRESHOLD_ACTIVE {
		nLockTimeFlags |= consensus.LocktimeMedianTimePast
	}

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

func CheckBlock(params *msg.BitcoinParams, pblock *model.Block, state *model.ValidationState, fCheckPOW, fCheckMerkleRoot bool) bool {
	//These are checks that are independent of context.
	if pblock.FChecked {
		return true
	}

	//Check that the header is valid (particularly PoW).  This is mostly
	// redundant with the call in AcceptBlockHeader.
	if !CheckBlockHeader(pblock, state, params, fCheckPOW) {
		return false
	}

	// Check the merkle root.
	if fCheckMerkleRoot {
		mutated := false
		hashMerkleRoot2 := consensus.BlockMerkleRoot(pblock, &mutated)
		if !pblock.BlockHeader.HashMerkleRoot.IsEqual(&hashMerkleRoot2) {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-txnmrklroot", true, "hashMerkleRoot mismatch")
		}

		// Check for merkle tree malleability (CVE-2012-2459): repeating
		// sequences of transactions in a block without affecting the merkle
		// root of a block, while still invalidating it.
		if mutated {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-txns-duplicate", true, "duplicate transaction")
		}
	}

	// All potential-corruption validation must be done before we do any
	// transaction validation, as otherwise we may mark the header as invalid
	// because we receive the wrong transactions for it.

	// First transaction must be coinbase.
	if len(pblock.Transactions) == 0 {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-cb-missing", false, "first tx is not coinbase")
	}

	//size limits
	nMaxBlockSize := policy.DEFAULT_BLOCK_MIN_TX_FEE

	// Bail early if there is no way this block is of reasonable size.
	minTransactionSize := model.NewTx().SerializeSize()
	if len(pblock.Transactions)*minTransactionSize > int(nMaxBlockSize) {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-blk-length", false, "size limits failed")
	}

	currentBlockSize := pblock.SerializeSize()
	if currentBlockSize > int(nMaxBlockSize) {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-blk-length", false, "size limits failed")
	}

	// And a valid coinbase.
	if !CheckCoinbase(pblock.Transactions[0], state, false) {
		hs := pblock.Transactions[0].TxHash()
		return state.Invalid(false, state.GetRejectCode(), state.GetRejectReason(),
			fmt.Sprintf("Coinbase check failed (txid %s) %s", hs.ToString(), state.GetDebugMessage()))
	}

	// Keep track of the sigops count.
	nSigOps := 0
	nMaxSigOpsCount := consensus.GetMaxBlockSigOpsCount(uint64(currentBlockSize))

	// Check transactions
	txCount := len(pblock.Transactions)
	tx := pblock.Transactions[0]

	i := 0
	for {
		// Count the sigops for the current transaction. If the total sigops
		// count is too high, the the block is invalid.
		nSigOps += tx.GetSigOpCountWithoutP2SH()
		if uint64(nSigOps) > nMaxSigOpsCount {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-blk-sigops",
				false, "out-of-bounds SigOpCount")
		}

		// Go to the next transaction.
		i++

		// We reached the end of the block, success.
		if i >= txCount {
			break
		}

		// Check that the transaction is valid. because this check differs for
		// the coinbase, the loos is arranged such as this only runs after at
		// least one increment.
		tx := pblock.Transactions[i]
		if !CheckRegularTransaction(tx, state, false) {
			hs := tx.TxHash()
			return state.Invalid(false, state.GetRejectCode(), state.GetRejectReason(),
				fmt.Sprintf("Transaction check failed (txid %s) %s", hs.ToString(), state.GetDebugMessage()))
		}
	}

	if fCheckPOW && fCheckMerkleRoot {
		pblock.FChecked = true
	}

	return true
}

// todo !!! will remove the Global status，And reconstruction package structure.

type BlockMap map[utils.Hash]*BlockIndex

var mapBlockIndex BlockMap

// AcceptBlock Store block on disk. If dbp is non-null, the file is known
// to already reside on disk.
func AcceptBlock(param *msg.BitcoinParams, pblock *model.Block, state *model.ValidationState, ppindex **BlockIndex, fRequested bool, dbp *DiskBlockPos, fNewBlock *bool) bool {

	if fNewBlock != nil {
		*fNewBlock = false
	}

	var pindex *BlockIndex
	if ppindex != nil {
		pindex = *ppindex
	}

	if !AcceptBlockHeader(param, &pblock.BlockHeader, state, &pindex) {
		return false
	}

	return true
}

func CheckBlockIndex(param *msg.BitcoinParams) {

}

func ActivateBestChain(param *msg.BitcoinParams, state *model.ValidationState, pblock *model.Block) bool {

	return false
}

func AcceptBlockHeader(param *msg.BitcoinParams, pblock *model.BlockHeader, state *model.ValidationState, ppindex **BlockIndex) bool {

	// Check for duplicate

	return true
}

func ProcessNewBlock(param *msg.BitcoinParams, pblock *model.Block, fForceProcessing bool, fNewBlock *bool) (bool, error) {

	if fNewBlock != nil {
		*fNewBlock = false
	}
	state := model.ValidationState{}
	// Ensure that CheckBlock() passes before calling AcceptBlock, as
	// belt-and-suspenders.
	ret := CheckBlock(param, pblock, &state, true, true)

	var pindex *BlockIndex
	if ret {
		ret = AcceptBlock(param, pblock, &state, &pindex, fForceProcessing, nil, fNewBlock)
	}

	CheckBlockIndex(param)
	if !ret {
		//	!todo add asynchronous notification
		return false, errors.Errorf(" AcceptBlock FAILED ")
	}

	notifyHeaderTip()

	// Only used to report errors, not invalidity - ignore it
	if !ActivateBestChain(param, &state, pblock) {
		return false, errors.Errorf("ActivateBestChain failed")
	}

	return true, nil
}

func ComputeBlockVersion(indexPrev *BlockIndex, params *msg.BitcoinParams, t *VersionBitsCache) int {
	version := VERSIONBITS_TOP_BITS

	for i := 0; i < int(msg.MAX_VERSION_BITS_DEPLOYMENTS); i++ {
		state := func() ThresholdState {
			t.Lock()
			defer t.Unlock()
			v := VersionBitsState(indexPrev, params, msg.DeploymentPos(i), t)
			return v
		}()

		if state == THRESHOLD_LOCKED_IN || state == THRESHOLD_STARTED {
			version |= int(VersionBitsMask(params, msg.DeploymentPos(i)))
		}
	}

	return version
}

func CheckCoinbase(tx *model.Tx, state *model.ValidationState, fCheckDuplicateInputs bool) bool {

	if !tx.IsCoinBase() {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-cb-missing", false, "first tx is not coinbase")
	}

	if !CheckTransactionCommon(tx, state, fCheckDuplicateInputs) {
		return false
	}

	if tx.Ins[0].Script.Size() < 2 || tx.Ins[0].Script.Size() > 100 {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-cb-length", false, "")
	}

	return true
}

//CheckRegularTransaction Context-independent validity checks for coinbase and
// non-coinbase transactions
func CheckRegularTransaction(tx *model.Tx, state *model.ValidationState, fCheckDuplicateInputs bool) bool {

	if tx.IsCoinBase() {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-tx-coinbase", false, "")
	}

	if !CheckTransactionCommon(tx, state, fCheckDuplicateInputs) {
		// CheckTransactionCommon fill in the state.
		return false
	}

	for _, txin := range tx.Ins {
		if txin.PreviousOutPoint.IsNull() {
			return state.Dos(10, false, model.REJECT_INVALID, "bad-txns-prevout-null",
				false, "")
		}
	}

	return true
}

func CheckTransactionCommon(tx *model.Tx, state *model.ValidationState, fCheckDuplicateInputs bool) bool {
	// Basic checks that don't depend on any context
	if len(tx.Ins) == 0 {
		return state.Dos(10, false, model.REJECT_INVALID, "bad-txns-vin-empty", false, "")
	}

	if len(tx.Outs) == 0 {
		return state.Dos(10, false, model.REJECT_INVALID, "bad-txns-vout-empty", false, "")
	}

	// Size limit
	if tx.SerializeSize() > model.MAX_TX_SIZE {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-txns-oversize", false, "")
	}

	// Check for negative or overflow output values
	nValueOut := int64(0)
	for _, txout := range tx.Outs {
		if txout.Value < 0 {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-txns-vout-negative", false, "")
		}

		if txout.Value > model.MAX_MONEY {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-txns-vout-toolarge", false, "")
		}

		nValueOut += txout.Value
		if !MoneyRange(nValueOut) {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-txns-txouttotal-toolarge", false, "")
		}
	}

	if tx.GetSigOpCountWithoutP2SH() > model.MAX_TX_SIGOPS_COUNT {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-txn-sigops", false, "")
	}

	// Check for duplicate inputs - note that this check is slow so we skip it
	// in CheckBlock
	if fCheckDuplicateInputs {
		vInOutPoints := make(map[model.OutPoint]struct{})
		for _, txIn := range tx.Ins {
			if _, ok := vInOutPoints[*txIn.PreviousOutPoint]; !ok {
				vInOutPoints[*txIn.PreviousOutPoint] = struct{}{}
			} else {
				return state.Dos(100, false, model.REJECT_INVALID, "bad-txns-inputs-duplicate", false, "")
			}
		}
	}

	return true
}

func MoneyRange(money int64) bool {
	return money <= 0 && money <= model.MAX_MONEY
}

func notifyHeaderTip() {

}

func init() {
	mapBlockIndex = make(BlockMap)
}
