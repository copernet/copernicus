package blockchain

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"strconv"

	"os"

	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/msg"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
	"gopkg.in/fatih/set.v0"
)

const (
	// DEFAULT_PERMIT_BAREMULTISIG  Default for -permitbaremultisig
	DEFAULT_PERMIT_BAREMULTISIG bool = true
	DEFAULT_CHECKPOINTS_ENABLED bool = true
	DEFAULT_TXINDEX             bool = false
	DEFAULT_BANSCORE_THRESHOLD  uint = 100
	MIN_BLOCKS_TO_KEEP          int  = 288
	//BLOCKFILE_CHUNK_SIZE pre-allocation chunk size for blk?????.dat files (since 0.8)
	BLOCKFILE_CHUNK_SIZE int = 0x1000000 //16MiB
	//UNDOFILE_CHUNK_SIZE pre-allocation chunk size for rev?????.dat files (since 0.8)
	UNDOFILE_CHUNK_SIZE int = 0x100000 // 1 MiB
)

var (
	gsetDirtyBlockIndex *algorithm.Set
	//HashAssumeValid is Block hash whose ancestors we will assume to have valid scripts without checking them.
	HashAssumeValid utils.Hash
	MapBlockIndex   BlockMap
	vinfoBlockFile  = make([]BlockFileInfo, 0)
	lastBlockFile   int
)

type FlushStateMode int

const (
	FLUSH_STATE_NONE FlushStateMode = iota
	FLUSH_STATE_IF_NEEDED
	FLUSH_STATE_PERIODIC
	FLUSH_STATE_ALWAYS
)

func init() {
	gsetDirtyBlockIndex = algorithm.NewSet()
}

func FormatStateMessage(state *model.ValidationState) string {
	if state.GetDebugMessage() == "" {
		return fmt.Sprintf("%s%s (code %c)", state.GetRejectReason(), "", state.GetRejectCode())
	}
	return fmt.Sprintf("%s%s (code %c)", state.GetRejectReason(), state.GetDebugMessage(), state.GetRejectCode())
}

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
	if VersionBitsState(pindexPrev, params, msg.DEPLOYMENT_CSV, &Gversionbitscache) == THRESHOLD_ACTIVE {
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

func CheckBlockHeader(blockHeader *model.BlockHeader, state *model.ValidationState, params *msg.BitcoinParams, fCheckPOW bool) bool {
	// Check proof of work matches claimed amount
	mpow := Pow{}
	blkHash, _ := blockHeader.GetHash()
	if fCheckPOW && !mpow.CheckProofOfWork(&blkHash, blockHeader.Bits, params) {
		return state.Dos(50, false, model.REJECT_INVALID, "high-hash", false, "proof of work failed")
	}

	return true
}

func CheckBlock(params *msg.BitcoinParams, pblock *model.Block, state *model.ValidationState, fCheckPOW, fCheckMerkleRoot bool) bool {
	//These are checks that are independent of context.
	if pblock.FChecked {
		return true
	}

	//Check that the header is valid (particularly PoW).  This is mostly
	// redundant with the call in AcceptBlockHeader.
	if !CheckBlockHeader(&pblock.BlockHeader, state, params, fCheckPOW) {
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

func TraceLog() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	return fmt.Sprintf("%s:%d %s\n", file, line, f.Name())
}

func (c *ChainState) CheckBlockIndex(param *msg.BitcoinParams) {

	if !GfCheckBlockIndex {
		return
	}

	//todo !! consider mutex here
	// During a reindex, we read the genesis block and call CheckBlockIndex
	// before ActivateBestChain, so we have the genesis block in mapBlockIndex
	// but no active chain. (A few of the tests when iterating the block tree
	// require that chainActive has been initialized.)
	if GChainState.ChainAcTive.Height() < 0 {
		if len(GChainState.MapBlockIndex.Data) > 1 {
			panic("because the activeChain height less 0, so the global status should have less 1 element")
		}
		return
	}

	// Build forward-pointing map of the entire block tree.
	forward := make(map[*BlockIndex][]*BlockIndex)
	for _, v := range GChainState.MapBlockIndex.Data {
		forward[v.PPrev] = append(forward[v.PPrev], v)
	}
	if len(forward) != len(GChainState.MapBlockIndex.Data) {
		panic("the two map size should be equal")
	}

	rangeGenesis := forward[nil]
	pindex := rangeGenesis[0]
	// There is only one index entry with parent nullptr.
	if len(rangeGenesis) != 1 {
		panic("There is only one index entry with parent nullptr.")
	}

	// Iterate over the entire block tree, using depth-first search.
	// Along the way, remember whether there are blocks on the path from genesis
	// block being explored which are the first to have certain properties.
	nNode := 0
	nHeight := 0
	// Oldest ancestor of pindex which is invalid.
	var pindexFirstInvalid *BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_HAVE_DATA.
	var pindexFirstMissing *BlockIndex
	// Oldest ancestor of pindex for which nTx == 0.
	var pindexFirstNeverProcessed *BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_TREE
	// (regardless of being valid or not).
	var pindexFirstNotTreeValid *BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_TRANSACTIONS
	// (regardless of being valid or not).
	var pindexFirstNotTransactionsValid *BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_CHAIN
	// (regardless of being valid or not).
	var pindexFirstNotChainValid *BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_SCRIPTS
	// (regardless of being valid or not).
	var pindexFirstNotScriptsValid *BlockIndex
	for pindex != nil {
		nNode++
		if pindexFirstInvalid == nil && pindex.Status&BLOCK_FAILED_VALID != 0 {
			pindexFirstInvalid = pindex
		}
		if pindexFirstMissing == nil && !(pindex.Status&BLOCK_HAVE_DATA != 0) {
			pindexFirstMissing = pindex
		}
		if pindexFirstNeverProcessed == nil && pindex.Txs == 0 {
			pindexFirstNeverProcessed = pindex
		}
		if pindex.PPrev != nil && pindexFirstNotTreeValid == nil &&
			(pindex.Status&BLOCK_VALID_MASK) < BLOCK_VALID_TREE {
			pindexFirstNotTreeValid = pindex
		}
		if pindex.PPrev != nil && pindexFirstNotTransactionsValid == nil &&
			(pindex.Status&BLOCK_VALID_MASK) < BLOCK_VALID_TRANSACTIONS {
			pindexFirstNotTransactionsValid = pindex
		}
		if pindex.PPrev != nil && pindexFirstNotChainValid == nil &&
			(pindex.Status&BLOCK_VALID_MASK) < BLOCK_VALID_CHAIN {
			pindexFirstNotChainValid = pindex
		}
		if pindex.PPrev != nil && pindexFirstNotScriptsValid == nil &&
			(pindex.Status&BLOCK_VALID_MASK) < BLOCK_VALID_SCRIPTS {
			pindexFirstNotScriptsValid = pindex
		}

		// Begin: actual consistency checks.
		if pindex.PPrev == nil {
			// Genesis block checks.
			// Genesis block's hash must match.
			if pindex.PHashBlock.Cmp(param.GenesisHash) != 0 {
				panic("the genesis block's hash incorrect")
			}
			// The current active chain's genesis block must be this block.
			if pindex != GChainState.ChainAcTive.Genesis() {
				panic("The current active chain's genesis block must be this block.")
			}
		}
		if pindex.ChainTx == 0 {
			// nSequenceId can't be set positive for blocks that aren't linked
			// (negative is used for preciousblock)
			if pindex.SequenceID > 0 {
				panic("nSequenceId can't be set positive for blocks that aren't linked")
			}
		}
		// VALID_TRANSACTIONS is equivalent to nTx > 0 for all nodes (whether or
		// not pruning has occurred). HAVE_DATA is only equivalent to nTx > 0
		// (or VALID_TRANSACTIONS) if no pruning has occurred.
		if !GfHavePruned {
			// If we've never pruned, then HAVE_DATA should be equivalent to nTx
			// > 0
			if !(pindex.Status&BLOCK_HAVE_DATA == BLOCK_HAVE_DATA) !=
				(pindex.Txs == 0) {
				panic("never pruned, then HAVE_DATA should be equivalent to nTx > 0")
			}
			if pindexFirstMissing != pindexFirstNeverProcessed {
				panic("never pruned, then HAVE_DATA should be equivalent to nTx > 0")
			}
		} else {
			// If we have pruned, then we can only say that HAVE_DATA implies
			// nTx > 0
			if pindex.Status&BLOCK_HAVE_DATA != 0 {
				if pindex.Txs <= 0 {
					panic("block status is BLOCK_HAVE_DATA, so the nTx > 0")
				}
			}
		}
		if pindex.Status&BLOCK_HAVE_UNDO != 0 {
			if pindex.Status&BLOCK_HAVE_DATA == 0 {
				panic("the block data should be had store the blk*dat file, so the " +
					"blkindex' status & BLOCK_HAVE_DATA should != 0")
			}
		}
		// This is pruning-independent.
		if (pindex.Status&BLOCK_VALID_MASK >= BLOCK_VALID_TRANSACTIONS) !=
			(pindex.Txs > 0) {
			panic("the blockindex TRANSACTIONS status should equivalent Txs > 0 ")
		}
		// All parents having had data (at some point) is equivalent to all
		// parents being VALID_TRANSACTIONS, which is equivalent to nChainTx
		// being set.
		// nChainTx != 0 is used to signal that all parent blocks have been
		// processed (but may have been pruned).
		if (pindexFirstNeverProcessed != nil) !=
			(pindex.ChainTx == 0) {
			panic("the block status is not equivalent ChainTx")
		}
		if pindexFirstNotTransactionsValid != nil !=
			(pindex.ChainTx == 0) {
			panic("the block status is not equivalent ChainTx")
		}
		// nHeight must be consistent.
		if pindex.Height != nHeight {
			panic("the blockIndex height is incorrect")
		}
		// For every block except the genesis block, the chainwork must be
		// larger than the parent's.
		if pindex.PPrev != nil && pindex.ChainWork.Cmp(&pindex.PPrev.ChainWork) < 0 {
			panic("For every block except the genesis block, the chainwork must be " +
				"larger than the parent's.")
		}
		// The pskip pointer must point back for all but the first 2 blocks.
		if pindex.Height >= 2 && (pindex.PSkip == nil || pindex.PSkip.Height >= nHeight) {
			panic(" The pskip pointer must point back for all but the first 2 blocks.")
		}
		// All mapBlockIndex entries must at least be TREE valid
		if pindexFirstNotTreeValid != nil {
			panic("All mapBlockIndex entries must at least be TREE valid")
		}
		if pindex.Status&BLOCK_VALID_MASK >= BLOCK_VALID_TREE {
			// TREE valid implies all parents are TREE valid
			if pindexFirstNotTreeValid != nil {
				panic("status TREE valid implies all parents are TREE valid")
			}
		}
		if pindex.Status&BLOCK_VALID_MASK >= BLOCK_VALID_CHAIN {
			// CHAIN valid implies all parents are CHAIN valid
			if pindexFirstNotChainValid != nil {
				panic("status CHAIN valid implies all parents are CHAIN valid")
			}
		}
		if pindex.Status&BLOCK_VALID_MASK >= BLOCK_VALID_SCRIPTS {
			// SCRIPTS valid implies all parents are SCRIPTS valid
			if pindexFirstNotScriptsValid != nil {
				panic("status SCRIPTS valid implies all parents are SCRIPTS valid")
			}
		}
		if pindexFirstInvalid == nil {
			// Checks for not-invalid blocks.
			// The failed mask cannot be set for blocks without invalid parents.
			if pindex.Status&BLOCK_FAILED_MASK != 0 {
				panic("The failed mask cannot be set for blocks without invalid parents.")
			}
		}
		if !blockIndexWorkComparator(pindex, GChainState.ChainAcTive.Tip()) &&
			pindexFirstNeverProcessed == nil {
			if pindexFirstInvalid == nil {
				// If this block sorts at least as good as the current tip and
				// is valid and we have all data for its parents, it must be in
				// setBlockIndexCandidates. chainActive.Tip() must also be there
				// even if some data has been pruned.
				if pindexFirstMissing == nil || pindex == GChainState.ChainAcTive.Tip() {
					if !c.setBlockIndexCandidates.HasItem(pindex) {
						panic("the setBlockIndexCandidates should have the pindex ")
					}
				}
				// If some parent is missing, then it could be that this block
				// was in setBlockIndexCandidates but had to be removed because
				// of the missing data. In this case it must be in
				// mapBlocksUnlinked -- see test below.
			}
		} else {
			// If this block sorts worse than the current tip or some ancestor's
			// block has never been seen, it cannot be in
			// setBlockIndexCandidates.
			if c.setBlockIndexCandidates.HasItem(pindex) {
				panic("the blockindex should not be in setBlockIndexCandidates")
			}
		}
		// Check whether this block is in mapBlocksUnlinked.
		foundInUnlinked := false
		if rangeUnlinked, ok := GChainState.MapBlocksUnlinked[pindex.PPrev]; ok {
			for i := 0; i < len(rangeUnlinked); i++ {
				if rangeUnlinked[i] == pindex {
					foundInUnlinked = true
					break
				}
			}
		}
		if pindex.PPrev != nil && (pindex.Status&BLOCK_HAVE_DATA != 0) &&
			pindexFirstNeverProcessed != nil && pindexFirstInvalid == nil {
			// If this block has block data available, some parent was never
			// received, and has no invalid parents, it must be in
			// mapBlocksUnlinked.
			if !foundInUnlinked {
				panic("the block must be in mapBlocksUnlinked")
			}
		}

		if !(pindex.Status&BLOCK_HAVE_DATA != 0) {
			// Can't be in mapBlocksUnlinked if we don't HAVE_DATA
			if foundInUnlinked {
				panic("the block can't be in mapBlocksUnlinked")
			}
		}
		if pindexFirstMissing == nil {
			// We aren't missing data for any parent -- cannot be in
			// mapBlocksUnlinked.
			if foundInUnlinked {
				panic("the block can't be in mapBlocksUnlinked")
			}
		}
		if pindex.PPrev != nil && (pindex.Status&BLOCK_HAVE_DATA != 0) &&
			pindexFirstNeverProcessed == nil && pindexFirstMissing != nil {
			// We HAVE_DATA for this block, have received data for all parents
			// at some point, but we're currently missing data for some parent.
			// We must have pruned.
			if !GfHavePruned {
				panic("We must have pruned.")
			}
			// This block may have entered mapBlocksUnlinked if:
			//  - it has a descendant that at some point had more work than the
			//    tip, and
			//  - we tried switching to that descendant but were missing
			//    data for some intermediate block between chainActive and the
			//    tip.
			// So if this block is itself better than chainActive.Tip() and it
			// wasn't in
			// setBlockIndexCandidates, then it must be in mapBlocksUnlinked.
			if blockIndexWorkComparator(pindex, GChainState.ChainAcTive.Tip()) &&
				!GChainState.setBlockIndexCandidates.HasItem(pindex) {
				if pindexFirstInvalid == nil {
					if !foundInUnlinked {
						panic("the block must be in mapBlocksUnlinked")
					}
				}
			}
		}

		// Try descending into the first subnode.
		if ran, ok := forward[pindex]; ok {
			// A subnode was found.
			pindex = ran[0]
			nHeight++
			continue
		}
		// This is a leaf node. Move upwards until we reach a node of which we
		// have not yet visited the last child.
		for pindex != nil {
			// We are going to either move to a parent or a sibling of pindex.
			// If pindex was the first with a certain property, unset the
			// corresponding variable.
			if pindex == pindexFirstInvalid {
				pindexFirstInvalid = nil
			}
			if pindex == pindexFirstMissing {
				pindexFirstMissing = nil
			}
			if pindex == pindexFirstNeverProcessed {
				pindexFirstNeverProcessed = nil
			}
			if pindex == pindexFirstNotTreeValid {
				pindexFirstNotTreeValid = nil
			}
			if pindex == pindexFirstNotTransactionsValid {
				pindexFirstNotTransactionsValid = nil
			}
			if pindex == pindexFirstNotChainValid {
				pindexFirstNotChainValid = nil
			}
			if pindex == pindexFirstNotScriptsValid {
				pindexFirstNotScriptsValid = nil
			}
			// Find our parent.
			pindexPar := pindex.PPrev
			// Find which child we just visited.
			if rangePar, ok := forward[pindexPar]; ok {
				tmp := rangePar[0]
				for pindex != tmp {
					// Our parent must have at least the node we're coming from as
					// child.
					if len(rangePar) == 0 {
						panic("")
					}
					rangePar = rangePar[1:]
					tmp = rangePar[0]
				}
				// Proceed to the next one.
				rangePar = rangePar[1:]
				if len(rangePar) > 0 {
					// Move to the sibling.
					pindex = rangePar[0]
					break
				} else {
					// Move up further.
					pindex = pindexPar
					nHeight--
					continue
				}

			}
		}
	}

	// Check that we actually traversed the entire map.
	if nNode != len(forward) {
		panic("the node number should equivalent forward element")
	}
}

func BlockIndexWorkComparator(pa, pb interface{}) bool {
	a := pa.(*BlockIndex)
	b := pb.(*BlockIndex)
	return blockIndexWorkComparator(a, b)
}

func blockIndexWorkComparator(pa, pb *BlockIndex) bool {
	// First sort by most total work, ...
	if pa.ChainWork.Cmp(&pb.ChainWork) > 0 {
		return false
	}
	if pa.ChainWork.Cmp(&pb.ChainWork) < 0 {
		return true
	}

	// ... then by earliest time received, ...
	if pa.SequenceID < pb.SequenceID {
		return false
	}
	if pa.SequenceID > pb.SequenceID {
		return true
	}

	// Use pointer address as tie breaker (should only happen with blocks
	// loaded from disk, as those all have id 0).
	a, err := strconv.ParseUint(fmt.Sprintf("%x", pa), 16, 0)
	if err != nil {
		panic("convert hex string to uint failed")
	}
	b, err := strconv.ParseUint(fmt.Sprintf("%x", pb), 16, 0)
	if err != nil {
		panic("convert hex string to uint failed")
	}
	if a < b {
		return false
	}
	if a > b {
		return true
	}

	// Identical blocks.
	return false
}

func ActivateBestChain(param *msg.BitcoinParams, state *model.ValidationState, pblock *model.Block) bool {

	return false
}

func AcceptBlockHeader(param *msg.BitcoinParams, pblkHeader *model.BlockHeader, state *model.ValidationState, ppindex **BlockIndex) bool {

	// Check for duplicate
	var pindex *BlockIndex
	hash, err := pblkHeader.GetHash()
	if err != nil {
		return false
	}
	if !hash.IsEqual(param.GenesisHash) {
		if pindex, ok := GChainState.MapBlockIndex.Data[hash]; ok {
			// Block header is already known.
			if ppindex != nil {
				*ppindex = pindex
			}
			if pindex.Status&BLOCK_FAILED_MASK != 0 {
				return state.Invalid(state.Error(fmt.Sprintf("block %s is marked invalid", hash.ToString())), 0, "duplicate", "")
			}
			return true
		}

		// todo !! Add log, when return false
		if !CheckBlockHeader(pblkHeader, state, param, true) {
			return false
		}

		// Get prev block index
		var pindexPrev *BlockIndex
		v, ok := GChainState.MapBlockIndex.Data[pblkHeader.HashPrevBlock]
		if !ok {
			return state.Dos(10, false, 0, "bad-prevblk", false, "")
		}
		pindexPrev = v

		if pindexPrev.Status&BLOCK_FAILED_MASK == BLOCK_FAILED_MASK {
			return state.Dos(100, false, model.REJECT_INVALID, "bad-prevblk", false, "")
		}

		if pindexPrev == nil {
			panic("the pindexPrev should not be nil")
		}

		if GfCheckpointsEnabled && !checkIndexAgainstCheckpoint(pindexPrev, state, param, &hash) {
			return false
		}

		// todo !! Add time param in the function
		if !ContextualCheckBlockHeader(pblkHeader, state, param, pindexPrev, 0) {
			return false
		}
	}

	if pindex == nil {
		pindex = AddToBlockIndex(pblkHeader)
	}

	if ppindex != nil {
		*ppindex = pindex
	}

	GChainState.CheckBlockIndex(param)
	return true
}

func AddToBlockIndex(pblkHeader *model.BlockHeader) *BlockIndex {
	// Check for duplicate
	hash, _ := pblkHeader.GetHash()
	if v, ok := GChainState.MapBlockIndex.Data[hash]; ok {
		return v
	}

	// Construct new block index object
	pindexNew := NewBlockIndex(pblkHeader)
	if pindexNew == nil {
		panic("the pindexNew should not equal nil")
	}

	// We assign the sequence id to blocks only when the full data is available,
	// to avoid miners withholding blocks but broadcasting headers, to get a
	// competitive advantage.
	pindexNew.SequenceID = 0
	GChainState.MapBlockIndex.Data[hash] = pindexNew
	pindexNew.PHashBlock = hash

	if miPrev, ok := GChainState.MapBlockIndex.Data[pblkHeader.HashPrevBlock]; ok {
		pindexNew.PPrev = miPrev
		pindexNew.Height = pindexNew.PPrev.Height + 1
		pindexNew.BuildSkip()
	}

	if pindexNew.PPrev != nil {
		pindexNew.TimeMax = uint32(math.Max(float64(pindexNew.PPrev.TimeMax), float64(pindexNew.Time)))
		pindexNew.ChainWork = pindexNew.PPrev.ChainWork
	} else {
		pindexNew.TimeMax = pindexNew.Time
		pindexNew.ChainWork = *big.NewInt(0)
	}

	pindexNew.RaiseValidity(BLOCK_VALID_TREE)
	if GindexBestHeader == nil || GindexBestHeader.ChainWork.Cmp(&pindexNew.ChainWork) < 0 {
		GindexBestHeader = pindexNew
	}

	gsetDirtyBlockIndex.AddItem(pindexNew)
	return pindexNew
}

func ContextualCheckBlockHeader(pblkHead *model.BlockHeader, state *model.ValidationState, param *msg.BitcoinParams, pindexPrev *BlockIndex, adjustedTime int) bool {
	nHeight := 0
	if pindexPrev != nil {
		nHeight = pindexPrev.Height + 1
	}

	pow := Pow{}
	// Check proof of work
	if pblkHead.Bits != pow.GetNextWorkRequired(pindexPrev, pblkHead, param) {
		return state.Dos(100, false, model.REJECT_INVALID, "bad-diffbits", false, "incorrect proof of work")
	}

	// Check timestamp against prev
	if int64(pblkHead.GetBlockTime()) <= pindexPrev.GetMedianTimePast() {
		return state.Invalid(false, model.REJECT_INVALID, "time-too-old",
			"block's timestamp is too early")
	}

	// Check timestamp
	if int(pblkHead.GetBlockTime()) >= adjustedTime+2*60*60 {
		return state.Invalid(false, model.REJECT_INVALID, "time-too-new",
			"block's timestamp is too far in the future")
	}

	// Reject outdated version blocks when 95% (75% on testnet) of the network
	// has upgraded:
	// check for version 2, 3 and 4 upgrades
	if pblkHead.Version < 2 && nHeight >= param.BIP34Height ||
		pblkHead.Version < 3 && nHeight >= param.BIP66Height ||
		pblkHead.Version < 4 && nHeight >= param.BIP65Height {
		return state.Invalid(false, model.REJECT_INVALID, fmt.Sprintf("bad-version(0x%08x)", pblkHead.Version),
			fmt.Sprintf("rejected nVersion=0x%08x block", pblkHead.Version))
	}

	return true
}

func checkIndexAgainstCheckpoint(pindexPrev *BlockIndex, state *model.ValidationState, param *msg.BitcoinParams, hash *utils.Hash) bool {
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

	GChainState.CheckBlockIndex(param)
	if !ret {
		//todo !!! add asynchronous notification
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

/**
 * BeginTime:Threshold condition checker that triggers when unknown versionbits are seen
 * on the network.
 */

func BeginTime(params *msg.BitcoinParams) int64 {
	return 0
}

func EndTime(params *msg.BitcoinParams) int64 {
	return math.MaxInt64
}

func Period(params *msg.BitcoinParams) int {
	return int(params.MinerConfirmationWindow)
}

func Threshold(params *msg.BitcoinParams) int {
	return int(params.RuleChangeActivationThreshold)
}

func Condition(pindex *BlockIndex, params *msg.BitcoinParams, t *VersionBitsCache) bool {
	return (int(pindex.Version)&VERSIONBITS_TOP_MASK) == VERSIONBITS_TOP_BITS && (pindex.Version)&1 != 0 && (ComputeBlockVersion(pindex.PPrev, params, t)&1) == 0
}

var warningcache [VERSIONBITS_NUM_BITS]ThresholdConditionCache

// GetBlockScriptFlags Returns the script flags which should be checked for a given block
func GetBlockScriptFlags(pindex *BlockIndex, param *msg.BitcoinParams) uint32 {
	//TODO: AssertLockHeld(cs_main);
	//var sc sync.RWMutex
	//sc.Lock()
	//defer sc.Unlock()

	// BIP16 didn't become active until Apr 1 2012
	nBIP16SwitchTime := 1333238400
	fStrictPayToScriptHash := int(pindex.GetBlockTime()) >= nBIP16SwitchTime

	var flags uint32

	if fStrictPayToScriptHash {
		flags = core.SCRIPT_VERIFY_P2SH
	} else {
		flags = core.SCRIPT_VERIFY_NONE
	}

	// Start enforcing the DERSIG (BIP66) rule
	if pindex.Height >= param.BIP66Height {
		flags |= core.SCRIPT_VERIFY_DERSIG
	}

	// Start enforcing CHECKLOCKTIMEVERIFY (BIP65) rule
	if pindex.Height >= param.BIP65Height {
		flags |= core.SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY
	}

	// Start enforcing BIP112 (CHECKSEQUENCEVERIFY) using versionbits logic.
	if VersionBitsState(pindex.PPrev, param, msg.DEPLOYMENT_CSV, &versionBitsCache) == THRESHOLD_ACTIVE {
		flags |= core.SCRIPT_VERIFY_CHECKSEQUENCEVERIFY
	}

	// If the UAHF is enabled, we start accepting replay protected txns
	if IsUAHFenabled(param, pindex.Height) {
		flags |= core.SCRIPT_VERIFY_STRICTENC
		flags |= core.SCRIPT_ENABLE_SIGHASH_FORKID
	}

	// If the Cash HF is enabled, we start rejecting transaction that use a high
	// s in their signature. We also make sure that signature that are supposed
	// to fail (for instance in multisig or other forms of smart contracts) are
	// null.
	if IsCashHFEnabled(param, pindex.GetMedianTimePast()) {
		flags |= core.SCRIPT_VERIFY_LOW_S
		flags |= core.SCRIPT_VERIFY_NULLFAIL
	}

	return flags
}

/**
 * BLOCK PRUNING CODE
 */

//CalculateCurrentUsage Calculate the amount of disk space the block & undo files currently use
func CalculateCurrentUsage() uint64 {
	var retval uint64
	for _, file := range vinfoBlockFile {
		retval += uint64(file.Size + file.UndoSize)
	}
	return retval
}

//PruneOneBlockFile Prune a block file (modify associated database entries)
func PruneOneBlockFile(fileNumber int) {
	bm := &BlockMap{
		Data: make(map[utils.Hash]*BlockIndex),
	}
	for _, value := range bm.Data {
		pindex := value
		if pindex.File == fileNumber {
			pindex.Status &= ^BLOCK_HAVE_DATA
			pindex.Status &= ^BLOCK_HAVE_UNDO
			pindex.File = 0
			pindex.DataPosition = 0
			pindex.UndoPosition = 0
			gsetDirtyBlockIndex.AddItem(pindex)

			// Prune from mapBlocksUnlinked -- any block we prune would have
			// to be downloaded again in order to consider its chain, at which
			// point it would be considered as a candidate for
			// mapBlocksUnlinked or setBlockIndexCandidates.
			ranges := GChainState.MapBlocksUnlinked[pindex.PPrev]
			tmpRange := make([]*BlockIndex, len(ranges))
			copy(tmpRange, ranges)
			for len(tmpRange) > 0 {
				v := tmpRange[0]
				tmpRange = tmpRange[1:]
				if v == pindex {
					tmp := make([]*BlockIndex, len(ranges)-1)
					for _, val := range tmpRange {
						if val != v {
							tmp = append(tmp, val)
						}
					}
					GChainState.MapBlocksUnlinked[pindex.PPrev] = tmp
				}
			}
		}
	}

	vinfoBlockFile[fileNumber].SetNull()
	gsetDirtyBlockIndex.AddItem(fileNumber)
}

func GetBlockPosFilename(pos *DiskBlockPos, prefix string) string {
	path := conf.GetDataPath() + "/" + "blocks" + "/" + fmt.Sprintf("%s%d.dat", prefix, pos.File)
	return path
}

func UnlinkPrunedFiles(setFilesToPrune *set.Set) {
	lists := setFilesToPrune.List()
	for key, value := range lists {
		v := value.(int)
		pos := &DiskBlockPos{
			File: v,
			Pos:  0,
		}
		os.Remove(GetBlockPosFilename(pos, "blk"))
		os.Remove(GetBlockPosFilename(pos, "rev"))
		log.Info("Prune: %s deleted blk/rev (%05u)\n", key)
	}
}

func FindFilesToPruneManual(setFilesToPrune *set.Set, manualPruneHeight int) {
	if GfPruneMode && manualPruneHeight <= 0 {
		panic("the GfPruneMode is false and manualPruneHeight equal zero")
	}

	//TODO: LOCK2(cs_main, cs_LastBlockFile);
	//var sc sync.RWMutex
	//sc.Lock()
	//defer sc.Unlock()

	if GChainActive.Tip() == nil {
		return
	}

	// last block to prune is the lesser of (user-specified height, MIN_BLOCKS_TO_KEEP from the tip)
	lastBlockWeCanPrune := math.Min(float64(manualPruneHeight), float64(GChainActive.Tip().Height-MIN_BLOCKS_TO_KEEP))
	count := 0
	for fileNumber := 0; fileNumber < lastBlockFile; fileNumber++ {
		if vinfoBlockFile[fileNumber].Size == 0 || int(vinfoBlockFile[fileNumber].HeightLast) > lastBlockFile {
			continue
		}
		PruneOneBlockFile(fileNumber)
		setFilesToPrune.Add(fileNumber)
		count++
	}
	log.Info("Prune (Manual): prune_height=%d removed %d blk/rev pairs\n", lastBlockWeCanPrune, count)
}

// PruneBlockFilesManual is called from the RPC code for pruneblockchain */
func PruneBlockFilesManual(nManualPruneHeight int) {
	var state *model.ValidationState
	FlushStateToDisk(state, FLUSH_STATE_NONE, nManualPruneHeight)
}

//FindFilesToPrune calculate the block/rev files that should be deleted to remain under target*/
func FindFilesToPrune(setFilesToPrune *set.Set, nPruneAfterHeight uint64) {
	//TODO: LOCK2(cs_main, cs_LastBlockFile);
	//var sc sync.RWMutex
	//sc.Lock()
	//defer sc.Unlock()
	if GChainActive.Tip() == nil || GPruneTarget == 0 {
		return
	}

	if uint64(GChainActive.Tip().Height) <= nPruneAfterHeight {
		return
	}

	nLastBlockWeCanPrune := GChainActive.Tip().Height - MIN_BLOCKS_TO_KEEP
	nCurrentUsage := CalculateCurrentUsage()
	// We don't check to prune until after we've allocated new space for files,
	// so we should leave a buffer under our target to account for another
	// allocation before the next pruning.
	nBuffer := uint64(BLOCKFILE_CHUNK_SIZE + UNDOFILE_CHUNK_SIZE)
	count := 0
	if nCurrentUsage+nBuffer >= GPruneTarget {
		for fileNumber := 0; fileNumber < lastBlockFile; fileNumber++ {
			nBytesToPrune := uint64(vinfoBlockFile[fileNumber].Size + vinfoBlockFile[fileNumber].UndoSize)

			if vinfoBlockFile[fileNumber].Size == 0 {
				continue
			}

			// are we below our target?
			if nCurrentUsage+nBuffer < GPruneTarget {
				break
			}

			// don't prune files that could have a block within
			// MIN_BLOCKS_TO_KEEP of the main chain's tip but keep scanning
			if int(vinfoBlockFile[fileNumber].HeightLast) > nLastBlockWeCanPrune {
				continue
			}

			PruneOneBlockFile(fileNumber)
			// Queue up the files for removal
			setFilesToPrune.Add(fileNumber)
			nCurrentUsage -= nBytesToPrune
			count++
		}
	}

	log.Info("prune", "Prune: target=%dMiB actual=%dMiB diff=%dMiB max_prune_height=%d removed %d blk/rev pairs\n",
		GPruneTarget/1024/1024, nCurrentUsage/1024/1024, (GPruneTarget-nCurrentUsage)/1024/1024, nLastBlockWeCanPrune, count)
}

func FlushStateToDisk(state *model.ValidationState, mode FlushStateMode, nManualPruneHeight int) bool {
	//TODO
	return true
}

//**************************** CBlock and CBlockIndex ****************************//

var (
	nTimeCheck     int64
	nTimeForks     int64
	nTimeVerify    int64
	nTimeConnect   int64
	nTimeIndex     int64
	nTimeCallbacks int64
	nTimeTotal     int64
)
