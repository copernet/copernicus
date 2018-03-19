package blockchain

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"sort"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/logger"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
	"gopkg.in/fatih/set.v0"
)

const (
	// DefaultPermitBareMultiSig  Default for -permitBareMultiSig
	DefaultPermitBareMultiSig      = true
	DefaultCheckPointsEnabled      = true
	DefaultTxIndex                 = false
	DefaultBanscoreThreshold  uint = 100
	// MinBlocksToKeep of chainActive.Tip() will not be pruned.
	MinBlocksToKeep      = 288
	DefaultMaxTipAge     = 24 * 60 * 60
	DefaultRelayPriority = true

	// DefaultMemPoolExpiry Default for -memPoolExpiry, expiration time
	// for memPool transactions in hours
	DefaultMemPoolExpiry       = 336
	MemPoolDumpVersion         = 1
	DefaultLimitFreeRelay      = 0
	DefaultAncestorLimit       = 25
	DefaultAncestorSizeLimit   = 101
	DefaultDescendantLimit     = 25
	DefaultDescendantSizeLimit = 101
	MaxFeeEstimationTipAge     = 3 * 60 * 60
	// gMinDiskSpace: Minimum disk space required - used in CheckDiskSpace()
	gMinDiskSpace = 52428800
)

// Reject codes greater or equal to this can be returned by AcceptToMemPool for
// transactions, to signal internal conditions. They cannot and should not be
// sent over the P2P network.
const (
	RejectInternal = 0x100
	// RejectHighFee too high fee. Can not be triggered by P2P transactions
	RejectHighFee = 0x100
	// RejectAlreadyKnown Transaction is already known (either in memPool or blockChain)
	RejectAlreadyKnown = 0x101
	// RejectConflict transaction conflicts with a transaction already known
	RejectConflict = 0x102
)

var (
	gSetDirtyBlockIndex      *container.Set
	gSetBlockIndexCandidates *container.Set
	//HashAssumeValid is Block hash whose ancestors we will assume to have valid scripts without checking them.
	HashAssumeValid       utils.Hash
	gHashPrevBestCoinBase utils.Hash
	MapBlockIndex         BlockMap
	MapBlocksUnlinked     = make(map[*core.BlockIndex][]*core.BlockIndex) // todo init
	gInfoBlockFile        = make([]*BlockFileInfo, 0)
	gLastBlockFile        int
	//setDirtyFileInfo  Dirty block file entries.
	gSetDirtyFileInfo *container.Set
	gLatchToFalse     atomic.Value
	//gBlockSequenceID blocks loaded from disk are assigned id 0, so start the counter at 1.
	gBlockSequenceID   int32
	gIndexHeaderOld    *core.BlockIndex
	gIndexBestHeader   *core.BlockIndex
	gIndexBestInvalid  *core.BlockIndex
	gIndexBestForkTip  *core.BlockIndex
	gIndexBestForkBase *core.BlockIndex
	gWarned            bool
	gTimeReadFromDisk  int64
	gTimeConnectTotal  int64
	gTimeFlush         int64
	gTimeChainState    int64
	gTimePostConnect   int64
	gTimeCheck         int64
	gTimeForks         int64
	gTimeVerify        int64
	gTimeConnect       int64
	gTimeIndex         int64
	gTimeCallbacks     int64
	gTimeTotal         int64
	gMinRelayTxFee     = utils.NewFeeRate(int64(DefaultMinRelayTxFee))
	GRequestShutdown   atomic.Value
	GDumpMemPoolLater  atomic.Value
	gLastFlush         int
	gLastSetChain      int
	gLastWrite         int

	gFreeCount float64
	gLastTime  int
	//chainWork for the last block that preciousBlock has been applied to.
	gLastPreciousChainWork big.Int
	//Decreasing counter (used by subsequent preciousBlock calls).
	gMapBlocksUnknownParent = make(map[utils.Hash][]*core.DiskBlockPos)
	gBlockReverseSequenceID = -1
)

// StartShutdown Thread management and startup/shutdown:
//
// The network-processing threads are all part of a thread group created by
// AppInit() or the Qt main() function.
//
// A clean exit happens when StartShutdown() or the SIGTERM signal handler sets
// fRequestShutdown, which triggers the DetectShutdownThread(), which interrupts
// the main thread group. DetectShutdownThread() then exits, which causes
// AppInit() to continue (it .joins the shutdown thread). Shutdown() is then
// called to clean up database connections, and stop other threads that should
// only be stopped after the main network-processing threads have exited.
//
// Note that if running -daemon the parent process returns from AppInit2 before
// adding any threads to the threadGroup, so .join_all() returns immediately and
// the parent exits from main().
//
// Shutdown for Qt is very similar, only it uses a QTimer to detect
// fRequestShutdown getting set, and then does the normal Qt shutdown thing.
//
func StartShutdown() {
	GRequestShutdown.Store(true)
}

func ShutdownRequested() bool {
	// Load() will return nil if Store() has not been called
	// if GRequestShutdown is nil, following will happens:
	// panic: interface conversion: interface {} is nil, not bool
	value, ok := GRequestShutdown.Load().(bool)
	if ok {
		return value
	}
	return false
}

type FlushStateMode int

const (
	FlushStateNone FlushStateMode = iota
	FlushStateIfNeeded
	FlushStatePeriodic
	FlushStateAlways
)

func init() {
	gSetDirtyBlockIndex = container.NewSet()
	gSetDirtyFileInfo = container.NewSet()
	gLatchToFalse = atomic.Value{}
	gBlockSequenceID = 1
}

func FindForkInGlobalIndex(chain *core.Chain, locator *BlockLocator) *core.BlockIndex {
	// Find the first block the caller has in the main chain
	for _, hash := range locator.vHave {
		mi, ok := MapBlockIndex.Data[hash]
		if ok {
			if chain.Contains(mi) {
				return mi
			}
			if mi.GetAncestor(chain.Height()) == chain.Tip() {
				return chain.Tip()
			}
		}
	}

	return chain.Genesis()
}

func FormatStateMessage(state *core.ValidationState) string {
	if state.GetDebugMessage() == "" {
		return fmt.Sprintf("%s%s (code %c)", state.GetRejectReason(), "", state.GetRejectCode())
	}
	return fmt.Sprintf("%s%s (code %c)", state.GetRejectReason(), state.GetDebugMessage(), state.GetRejectCode())
}

//IsUAHFEnabled Check is UAHF has activated.
func IsUAHFEnabled(params *msg.BitcoinParams, height int) bool {
	return height >= params.UAHFHeight
}

func IsCashHFEnabled(params *msg.BitcoinParams, medianTimePast int64) bool {
	return params.CashHardForkActivationTime <= medianTimePast
}

func ContextualCheckTransaction(params *msg.BitcoinParams, tx *core.Tx, state *core.ValidationState,
	height int, lockTimeCutoff int64) bool {

	if !tx.IsFinalTx(height, lockTimeCutoff) {
		return state.Dos(10, false, core.RejectInvalid, "bad-txns-nonFinal",
			false, "non-final transaction")
	}

	if IsUAHFEnabled(params, height) && height <= params.AntiReplayOpReturnSunsetHeight {
		for _, txo := range tx.Outs {
			if txo.Script.IsCommitment(params.AntiReplayOpReturnCommitment) {
				return state.Dos(10, false, core.RejectInvalid, "bad-txn-replay",
					false, "non playable transaction")
			}
		}
	}

	return true
}

func ContextualCheckBlock(params *msg.BitcoinParams, block *core.Block, state *core.ValidationState,
	indexPrev *core.BlockIndex) bool {

	var height int
	if indexPrev != nil {
		height = indexPrev.Height + 1
	}

	lockTimeFlags := 0
	if VersionBitsState(indexPrev, params, msg.DeploymentCSV, GVersionBitsCache) == ThresholdActive {
		lockTimeFlags |= core.LocktimeMedianTimePast
	}

	medianTimePast := indexPrev.GetMedianTimePast()
	if indexPrev == nil {
		medianTimePast = 0
	}

	lockTimeCutoff := int64(block.BlockHeader.GetBlockTime())
	if lockTimeFlags&core.LocktimeMedianTimePast != 0 {
		lockTimeCutoff = medianTimePast
	}

	// Check that all transactions are finalized
	for _, tx := range block.Txs {
		if !ContextualCheckTransaction(params, tx, state, height, lockTimeCutoff) {
			return false
		}
	}

	// Enforce rule that the coinBase starts with serialized block height
	expect := core.Script{}
	if height >= params.BIP34Height {
		expect.PushInt64(int64(height))
		if block.Txs[0].Ins[0].Script.Size() < expect.Size() ||
			bytes.Equal(expect.GetScriptByte(), block.Txs[0].Ins[0].Script.GetScriptByte()[:len(expect.GetScriptByte())]) {
			return state.Dos(100, false, core.RejectInvalid, "bad-cb-height",
				false, "block height mismatch in coinBase")
		}
	}

	return true
}

func CheckBlockHeader(blockHeader *core.BlockHeader, state *core.ValidationState, params *msg.BitcoinParams, fCheckPOW bool) bool {
	// Check proof of work matches claimed amount
	mPow := Pow{}
	blkHash, _ := blockHeader.GetHash()
	if fCheckPOW && !mPow.CheckProofOfWork(&blkHash, blockHeader.Bits, params) {
		return state.Dos(50, false, core.RejectInvalid, "high-hash",
			false, "proof of work failed")
	}

	return true
}

func CheckBlock(params *msg.BitcoinParams, block *core.Block, state *core.ValidationState,
	checkPOW, checkMerkleRoot bool) bool {

	//These are checks that are independent of context.
	if block.Checked {
		return true
	}

	//Check that the header is valid (particularly PoW).  This is mostly
	// redundant with the call in AcceptBlockHeader.
	if !CheckBlockHeader(&block.BlockHeader, state, params, checkPOW) {
		return false
	}

	// Check the merkle root.
	if checkMerkleRoot {
		mutated := false
		hashMerkleRoot2 := core.BlockMerkleRoot(block, &mutated)
		if !block.BlockHeader.MerkleRoot.IsEqual(&hashMerkleRoot2) {
			return state.Dos(100, false, core.RejectInvalid, "bad-txnMrklRoot",
				true, "hashMerkleRoot mismatch")
		}

		// Check for merkle tree malleability (CVE-2012-2459): repeating
		// sequences of transactions in a block without affecting the merkle
		// root of a block, while still invalidating it.
		if mutated {
			return state.Dos(100, false, core.RejectInvalid, "bad-txns-duplicate",
				true, "duplicate transaction")
		}
	}

	// All potential-corruption validation must be done before we do any
	// transaction validation, as otherwise we may mark the header as invalid
	// because we receive the wrong transactions for it.

	// First transaction must be coinBase.
	if len(block.Txs) == 0 {
		return state.Dos(100, false, core.RejectInvalid, "bad-cb-missing",
			false, "first tx is not coinBase")
	}

	//size limits
	nMaxBlockSize := policy.DefaultBlockMinTxFee

	// Bail early if there is no way this block is of reasonable size.
	minTransactionSize := core.NewTx().SerializeSize()
	if len(block.Txs)*minTransactionSize > int(nMaxBlockSize) {
		return state.Dos(100, false, core.RejectInvalid, "bad-blk-length",
			false, "size limits failed")
	}

	currentBlockSize := block.SerializeSize()
	if currentBlockSize > int(nMaxBlockSize) {
		return state.Dos(100, false, core.RejectInvalid, "bad-blk-length",
			false, "size limits failed")
	}

	// And a valid coinBase.
	if !block.Txs[0].CheckCoinbase(state, false) {
		hs := block.Txs[0].TxHash()
		return state.Invalid(false, state.GetRejectCode(), state.GetRejectReason(),
			fmt.Sprintf("Coinbase check failed (txid %s) %s", hs.ToString(), state.GetDebugMessage()))
	}

	// Keep track of the sigOps count.
	nSigOps := 0
	nMaxSigOpsCount := core.GetMaxBlockSigOpsCount(uint64(currentBlockSize))

	// Check transactions
	txCount := len(block.Txs)
	tx := block.Txs[0]

	i := 0
	for {
		// Count the sigOps for the current transaction. If the total sigOps
		// count is too high, the the block is invalid.
		nSigOps += tx.GetSigOpCountWithoutP2SH()
		if uint64(nSigOps) > nMaxSigOpsCount {
			return state.Dos(100, false, core.RejectInvalid, "bad-blk-sigOps",
				false, "out-of-bounds SigOpCount")
		}

		// Go to the next transaction.
		i++

		// We reached the end of the block, success.
		if i >= txCount {
			break
		}

		// Check that the transaction is valid. because this check differs for
		// the coinBase, the loos is arranged such as this only runs after at
		// least one increment.
		tx := block.Txs[i]
		if !tx.CheckRegularTransaction(state, false) {
			hs := tx.TxHash()
			return state.Invalid(false, state.GetRejectCode(), state.GetRejectReason(),
				fmt.Sprintf("Transaction check failed (txid %s) %s", hs.ToString(), state.GetDebugMessage()))
		}
	}

	if checkPOW && checkMerkleRoot {
		block.Checked = true
	}

	return true
}

func ResetBlockFailureFlags(index *core.BlockIndex) bool {
	// todo AssertLockHeld(cs_main)
	height := index.Height
	// Remove the invalidity flag from this block and all its descendants.
	for _, bl := range MapBlockIndex.Data {
		if !bl.IsValid(core.BlockValidTransactions) && bl.GetAncestor(height) == index {
			bl.Status &= ^core.BlockFailedMask
			gSetDirtyBlockIndex.AddItem(bl)
			if bl.IsValid(core.BlockValidTransactions) && bl.ChainTxCount != 0 &&
				BlockIndexWorkComparator(GChainActive.Tip(), bl) {
				gSetBlockIndexCandidates.AddItem(bl)
			}
			if bl == gIndexBestInvalid {
				// Reset invalid block marker if it was pointing to one of
				// those.
				gIndexBestInvalid = nil
			}
		}
	}

	// Remove the invalidity flag from all ancestors too.
	for index != nil {
		if index.Status&core.BlockFailedMask != 0 {
			index.Status &= ^core.BlockFailedMask
			gSetDirtyBlockIndex.AddItem(index)
		}
		index = index.Prev
	}
	return true
}

// AcceptBlock Store block on disk. If dbp is non-null, the file is known
// to already reside on disk.
func AcceptBlock(param *msg.BitcoinParams, pblock *core.Block, state *core.ValidationState,
	ppindex **core.BlockIndex, fRequested bool, dbp *core.DiskBlockPos, fNewBlock *bool) bool {

	if fNewBlock != nil {
		*fNewBlock = false
	}

	var pindex *core.BlockIndex
	if ppindex != nil {
		pindex = *ppindex
	}

	if !AcceptBlockHeader(param, &pblock.BlockHeader, state, &pindex) {
		return false
	}

	// Try to process all requested blocks that we don't have, but only
	// process an unrequested block if it's new and has enough work to
	// advance our tip, and isn't too many blocks ahead.
	fAlreadyHave := pindex.Status&core.BlockHaveData != 0
	fHasMoreWork := true
	tip := GChainState.ChainActive.Tip()
	if tip != nil {
		fHasMoreWork = pindex.ChainWork.Cmp(&tip.ChainWork) > 0
	}
	// Blocks that are too out-of-order needlessly limit the effectiveness of
	// pruning, because pruning will not delete block files that contain any
	// blocks which are too close in height to the tip.  Apply this test
	// regardless of whether pruning is enabled; it should generally be safe to
	// not process unrequested blocks.
	fTooFarAhead := pindex.Height > GChainState.ChainActive.Height()+MinBlocksToKeep

	// TODO: Decouple this function from the block download logic by removing
	// fRequested
	// This requires some new chain dataStructure to efficiently look up if a
	// block is in a chain leading to a candidate for best tip, despite not
	// being such a candidate itself.

	// TODO: deal better with return value and error conditions for duplicate
	// and unrequested blocks.
	if fAlreadyHave {
		return true
	}

	// If we didn't ask for it:
	if !fRequested {
		// This is a previously-processed block that was pruned.
		if pindex.TxCount != 0 {
			return true
		}
		// Don't process less-work chains.
		if !fHasMoreWork {
			return true
		}
		// Block height is too high.
		if fTooFarAhead {
			return true
		}
	}

	if fNewBlock != nil {
		*fNewBlock = true
	}

	if !CheckBlock(param, pblock, state, true, true) ||
		!ContextualCheckBlock(param, pblock, state, pindex.Prev) {
		if state.IsInvalid() && !state.CorruptionPossible() {
			pindex.Status |= core.BlockFailedValid
			gSetDirtyBlockIndex.AddItem(pindex)
		}
		return logger.ErrorLog(fmt.Sprintf("%s: %s (block %s)", logger.TraceLog(), state.FormatStateMessage(),
			pblock.Hash.ToString()))
	}

	// Header is valid/has work, merkle tree and segwit merkle tree are
	// good...RELAY NOW (but if it does not build on our best tip, let the
	// SendMessages loop relay it)
	if !IsInitialBlockDownload() && GChainState.ChainActive.Tip() == pindex.Prev {
		//	todo !!! send signal, we find a new valid block
	}

	nHeight := pindex.Height
	// Write block to history file
	nBlockSize := pblock.SerializeSize()
	var blockPos core.DiskBlockPos
	if dbp != nil {
		blockPos = *dbp
	}
	if !FindBlockPos(state, &blockPos, uint(nBlockSize+8), uint(nHeight),
		uint64(pblock.BlockHeader.GetBlockTime()), dbp != nil) {

		return logger.ErrorLog("AcceptBlock(): FindBlockPos failed")
	}
	if dbp == nil {
		if !WriteBlockToDisk(pblock, &blockPos, param.BitcoinNet) {
			AbortNode(state, "Failed to write block.", "")
		}
	}
	if !ReceivedBlockTransactions(pblock, state, pindex, &blockPos) {
		return logger.ErrorLog("AcceptBlock(): ReceivedBlockTransactions failed")
	}

	//todo !!! find C++ code throw exception place
	//if len(reason) != 0 {
	//	return AbortNode(state, fmt.Sprintf("System error: ", reason, ""))
	//}

	if GCheckForPruning {
		// we just allocated more disk space for block files.
		FlushStateToDisk(state, FlushStateNone, 0)
	}

	return true
}

//ReceivedBlockTransactions Mark a block as having its data received and checked (up to
//* BLOCK_VALID_TRANSACTIONS).
func ReceivedBlockTransactions(pblock *core.Block, state *core.ValidationState,
	pindexNew *core.BlockIndex, pos *core.DiskBlockPos) bool {

	pindexNew.TxCount = len(pblock.Txs)
	pindexNew.ChainTxCount = 0
	pindexNew.File = pos.File
	pindexNew.DataPos = pos.Pos
	pindexNew.UndoPos = 0
	pindexNew.Status |= core.BlockHaveData
	pindexNew.RaiseValidity(core.BlockValidTransactions)
	gSetDirtyBlockIndex.AddItem(pindexNew)

	if pindexNew.Prev == nil || pindexNew.Prev.ChainTxCount != 0 {
		// If indexNew is the genesis block or all parents are
		// BLOCK_VALID_TRANSACTIONS.
		vIndex := make([]*core.BlockIndex, 0)
		vIndex = append(vIndex, pindexNew)

		// Recursively process any descendant blocks that now may be eligible to
		// be connected.
		for len(vIndex) > 0 {
			pindex := vIndex[0]
			vIndex = vIndex[1:]
			if pindex.Prev != nil {
				pindex.ChainTxCount += pindex.Prev.ChainTxCount
			} else {
				pindex.ChainTxCount += 0
			}
			{
				//	todo !!! add sync.lock cs_nBlockSequenceId
				pindex.SequenceID = gBlockSequenceID
				gBlockSequenceID++
			}
			if GChainState.ChainActive.Tip() == nil ||
				!blockIndexWorkComparator(pindex, GChainState.ChainActive.Tip()) {
				GChainState.setBlockIndexCandidates.AddInterm(pindex)
			}
			rangs, ok := GChainState.MapBlocksUnlinked[pindex]
			if ok {
				tmpRang := make([]*core.BlockIndex, len(rangs))
				copy(tmpRang, rangs)
				for len(tmpRang) > 0 {
					vIndex = append(vIndex, tmpRang[0])
					tmpRang = tmpRang[1:]
				}
				delete(GChainState.MapBlocksUnlinked, pindex)
			}
		}
	} else {
		if pindexNew.Prev != nil && pindexNew.Prev.IsValid(core.BlockValidTree) {
			GChainState.MapBlocksUnlinked[pindexNew.Prev] = append(GChainState.MapBlocksUnlinked[pindexNew.Prev], pindexNew)
		}
	}

	return true
}

func AbortNodes(reason, userMessage string) bool {
	logger.GetLogger().Info("*** %s\n", reason)

	//todo:
	if len(userMessage) == 0 {
		panic("Error: A fatal internal error occurred, see debug.log for details")
	} else {

	}
	StartShutdown()
	return false
}

func AbortNode(state *core.ValidationState, reason, userMessage string) bool {
	AbortNodes(reason, userMessage)
	return state.Error(reason)
}

func WriteBlockToDisk(block *core.Block, pos *core.DiskBlockPos, messageStart utils.BitcoinNet) bool {
	// Open history file to append
	fileOut := OpenBlockFile(pos, false)
	if fileOut == nil {
		logger.ErrorLog("WriteBlockToDisk: OpenBlockFile failed")
	}

	// Write index header
	size := block.SerializeSize()

	//4 bytes
	err := utils.BinarySerializer.PutUint32(fileOut, binary.LittleEndian, uint32(messageStart))
	if err != nil {
		logger.ErrorLog("the messageStart write failed")
	}
	utils.WriteVarInt(fileOut, uint64(size))

	// Write block
	fileOutPos, err := fileOut.Seek(0, 1)
	if fileOutPos < 0 || err != nil {
		logger.ErrorLog("WriteBlockToDisk: ftell failed")
	}

	pos.Pos = int(fileOutPos)
	block.Serialize(fileOut)

	return true
}

//IsInitialBlockDownload Check whether we are doing an initial block download
//(synchronizing from disk or network)
func IsInitialBlockDownload() bool {
	// Once this function has returned false, it must remain false.
	gLatchToFalse.Store(false)
	// Optimization: pre-test latch before taking the lock.
	if gLatchToFalse.Load().(bool) {
		return false
	}

	//todo !!! add cs_main sync.lock in here
	if gLatchToFalse.Load().(bool) {
		return false
	}
	if GImporting.Load().(bool) || GfReindex {
		return true
	}
	if GChainState.ChainActive.Tip() == nil {
		return true
	}
	if GChainState.ChainActive.Tip().ChainWork.Cmp(&msg.ActiveNetParams.MinimumChainWork) < 0 {
		return true
	}
	if int64(GChainState.ChainActive.Tip().GetBlockTime()) < utils.GetMockTime()-GMaxTipAge {
		return true
	}
	gLatchToFalse.Store(true)

	return false
}

func FindBlockPos(state *core.ValidationState, pos *core.DiskBlockPos, nAddSize uint,
	nHeight uint, nTime uint64, fKnown bool) bool {

	//	todo !!! Add sync.Lock in the later, because the concurrency goroutine
	nFile := pos.File
	if !fKnown {
		nFile = gLastBlockFile
	}

	if !fKnown {
		for uint(gInfoBlockFile[nFile].Size)+nAddSize >= MaxBlockFileSize {
			nFile++
		}
		pos.File = nFile
		pos.Pos = int(gInfoBlockFile[nFile].Size)
	}

	if nFile != gLastBlockFile {
		if !fKnown {
			logger.GetLogger().Info(fmt.Sprintf("Leaving block file %d: %s\n", gLastBlockFile,
				gInfoBlockFile[gLastBlockFile].ToString()))
		}
		FlushBlockFile(!fKnown)
		gLastBlockFile = nFile
	}

	gInfoBlockFile[nFile].AddBlock(uint32(nHeight), nTime)
	if fKnown {
		gInfoBlockFile[nFile].Size = uint32(math.Max(float64(pos.Pos+int(nAddSize)),
			float64(gInfoBlockFile[nFile].Size)))
	} else {
		gInfoBlockFile[nFile].Size += uint32(nAddSize)
	}

	if !fKnown {
		nOldChunks := (pos.Pos + BlockFileChunkSize - 1) / BlockFileChunkSize
		nNewChunks := (gInfoBlockFile[nFile].Size + BlockFileChunkSize - 1) / BlockFileChunkSize
		if nNewChunks > uint32(nOldChunks) {
			if GPruneMode {
				GCheckForPruning = true
				if CheckDiskSpace(nNewChunks*BlockFileChunkSize - uint32(pos.Pos)) {
					pfile := OpenBlockFile(pos, false)
					if pfile != nil {
						logger.GetLogger().Info("Pre-allocating up to position 0x%x in blk%05u.dat\n",
							nNewChunks*BlockFileChunkSize, pos.File)
						AllocateFileRange(pfile, pos.Pos, nNewChunks*BlockFileChunkSize-uint32(pos.Pos))
						pfile.Close()
					}
				} else {
					return state.Error("out of disk space")
				}
			}
		}
	}

	gSetDirtyFileInfo.AddItem(nFile)
	return true
}

func AllocateFileRange(file *os.File, offset int, length uint32) {
	// Fallback version
	// TODO: just write one byte per block
	var buf [65536]byte
	file.Seek(int64(offset), 0)
	for length > 0 {
		now := 65536
		if int(length) < now {
			now = int(length)
		}
		// Allowed to fail; this function is advisory anyway.
		_, err := file.Write(buf[:])
		if err != nil {
			panic("the file write failed.")
		}
		length -= uint32(now)
	}
}

func CheckDiskSpace(nAdditionalBytes uint32) bool {
	path := conf.GetDataPath()
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return logger.ErrorLog("can not get disk info")
	}
	nFreeBytesAvailable := fs.Ffree * uint64(fs.Bsize)

	// Check for nMinDiskSpace bytes (currently 50MB)
	if int(nFreeBytesAvailable) < gMinDiskSpace+int(nAdditionalBytes) {
		return AbortNodes("Disk space is low!", "Error: Disk space is low!")
	}
	return true
}

func FlushBlockFile(fFinalize bool) {
	// todo !!! add file sync.lock, LOCK(cs_LastBlockFile);
	posOld := core.NewDiskBlockPos(gLastBlockFile, 0)

	fileOld := OpenBlockFile(posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64(gInfoBlockFile[gLastBlockFile].Size))
			fileOld.Sync()
			fileOld.Close()
		}
	}

	fileOld = OpenUndoFile(*posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64(gInfoBlockFile[gLastBlockFile].UndoSize))
			fileOld.Sync()
			fileOld.Close()
		}
	}
}

func OpenBlockFile(pos *core.DiskBlockPos, fReadOnly bool) *os.File {
	return OpenDiskFile(*pos, "blk", fReadOnly)
}

func OpenUndoFile(pos core.DiskBlockPos, fReadOnly bool) *os.File {
	return OpenDiskFile(pos, "rev", fReadOnly)
}

func OpenDiskFile(pos core.DiskBlockPos, prefix string, fReadOnly bool) *os.File {
	if pos.IsNull() {
		return nil
	}
	path := GetBlockPosParentFilename()
	utils.MakePath(path)

	file, err := os.Open(path + "rb+")
	if file == nil && !fReadOnly || err != nil {
		file, err = os.Open(path + "wb+")
		if err == nil {
			panic("open wb+ file failed ")
		}
	}
	if file == nil {
		logger.GetLogger().Info("Unable to open file %s\n", path)
		return nil
	}
	if pos.Pos > 0 {
		if _, err := file.Seek(0, 1); err != nil {
			logger.GetLogger().Info("Unable to seek to position %u of %s\n", pos.Pos, path)
			file.Close()
			return nil
		}
	}

	return file
}

func GetBlockPosFilename(pos core.DiskBlockPos, prefix string) string {
	return conf.GetDataPath() + "/blocks/" + fmt.Sprintf("%s%05d.dat", prefix, pos.File)
}

func GetBlockPosParentFilename() string {
	return conf.GetDataPath() + "/blocks/"
}

func (c *ChainState) CheckBlockIndex(param *msg.BitcoinParams) {

	if !GCheckBlockIndex {
		return
	}

	//todo !! consider mutex here
	// During a reindex, we read the genesis block and call CheckBlockIndex
	// before ActivateBestChain, so we have the genesis block in mapBlockIndex
	// but no active chain. (A few of the tests when iterating the block tree
	// require that chainActive has been initialized.)
	if GChainState.ChainActive.Height() < 0 {
		if len(GChainState.MapBlockIndex.Data) > 1 {
			panic("because the activeChain height less 0, so the global status should have less 1 element")
		}
		return
	}

	// Build forward-pointing map of the entire block tree.
	forward := make(map[*core.BlockIndex][]*core.BlockIndex)
	for _, v := range GChainState.MapBlockIndex.Data {
		forward[v.Prev] = append(forward[v.Prev], v)
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
	var pindexFirstInvalid *core.BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_HAVE_DATA.
	var pindexFirstMissing *core.BlockIndex
	// Oldest ancestor of pindex for which nTx == 0.
	var pindexFirstNeverProcessed *core.BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_TREE
	// (regardless of being valid or not).
	var pindexFirstNotTreeValid *core.BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_TRANSACTIONS
	// (regardless of being valid or not).
	var pindexFirstNotTransactionsValid *core.BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_CHAIN
	// (regardless of being valid or not).
	var pindexFirstNotChainValid *core.BlockIndex
	// Oldest ancestor of pindex which does not have BLOCK_VALID_SCRIPTS
	// (regardless of being valid or not).
	var pindexFirstNotScriptsValid *core.BlockIndex
	for pindex != nil {
		nNode++
		if pindexFirstInvalid == nil && pindex.Status&core.BlockFailedValid != 0 {
			pindexFirstInvalid = pindex
		}
		if pindexFirstMissing == nil && !(pindex.Status&core.BlockHaveData != 0) {
			pindexFirstMissing = pindex
		}
		if pindexFirstNeverProcessed == nil && pindex.TxCount == 0 {
			pindexFirstNeverProcessed = pindex
		}
		if pindex.Prev != nil && pindexFirstNotTreeValid == nil &&
			(pindex.Status&core.BlockValidMask) < core.BlockValidTree {
			pindexFirstNotTreeValid = pindex
		}
		if pindex.Prev != nil && pindexFirstNotTransactionsValid == nil &&
			(pindex.Status&core.BlockValidMask) < core.BlockValidTransactions {
			pindexFirstNotTransactionsValid = pindex
		}
		if pindex.Prev != nil && pindexFirstNotChainValid == nil &&
			(pindex.Status&core.BlockValidMask) < core.BlockValidChain {
			pindexFirstNotChainValid = pindex
		}
		if pindex.Prev != nil && pindexFirstNotScriptsValid == nil &&
			(pindex.Status&core.BlockValidMask) < core.BlockValidScripts {
			pindexFirstNotScriptsValid = pindex
		}

		// Begin: actual consistency checks.
		if pindex.Prev == nil {
			// Genesis block checks.
			// Genesis block's hash must match.
			if pindex.BlockHash.Cmp(param.GenesisHash) != 0 {
				panic("the genesis block's hash incorrect")
			}
			// The current active chain's genesis block must be this block.
			if pindex != GChainState.ChainActive.Genesis() {
				panic("The current active chain's genesis block must be this block.")
			}
		}
		if pindex.ChainTxCount == 0 {
			// nSequenceId can't be set positive for blocks that aren't linked
			// (negative is used for preciousblock)
			if pindex.SequenceID > 0 {
				panic("nSequenceId can't be set positive for blocks that aren't linked")
			}
		}
		// VALID_TRANSACTIONS is equivalent to nTx > 0 for all nodes (whether or
		// not pruning has occurred). HAVE_DATA is only equivalent to nTx > 0
		// (or VALID_TRANSACTIONS) if no pruning has occurred.
		if !GHavePruned {
			// If we've never pruned, then HAVE_DATA should be equivalent to nTx
			// > 0
			if !(pindex.Status&core.BlockHaveData == core.BlockHaveData) !=
				(pindex.TxCount == 0) {
				panic("never pruned, then HAVE_DATA should be equivalent to nTx > 0")
			}
			if pindexFirstMissing != pindexFirstNeverProcessed {
				panic("never pruned, then HAVE_DATA should be equivalent to nTx > 0")
			}
		} else {
			// If we have pruned, then we can only say that HAVE_DATA implies
			// nTx > 0
			if pindex.Status&core.BlockHaveData != 0 {
				if pindex.TxCount <= 0 {
					panic("block status is BlockHaveData, so the nTx > 0")
				}
			}
		}
		if pindex.Status&core.BlockHaveUndo != 0 {
			if pindex.Status&core.BlockHaveData == 0 {
				panic("the block data should be had store the blk*dat file, so the " +
					"blkIndex' status & BlockHaveData should != 0")
			}
		}
		// This is pruning-independent.
		if (pindex.Status&core.BlockValidMask >= core.BlockValidTransactions) !=
			(pindex.TxCount > 0) {
			panic("the blockIndex TRANSACTIONS status should equivalent Txs > 0 ")
		}
		// All parents having had data (at some point) is equivalent to all
		// parents being VALID_TRANSACTIONS, which is equivalent to nChainTx
		// being set.
		// nChainTx != 0 is used to signal that all parent blocks have been
		// processed (but may have been pruned).
		if (pindexFirstNeverProcessed != nil) !=
			(pindex.ChainTxCount == 0) {
			panic("the block status is not equivalent ChainTx")
		}
		if pindexFirstNotTransactionsValid != nil !=
			(pindex.ChainTxCount == 0) {
			panic("the block status is not equivalent ChainTx")
		}
		// nHeight must be consistent.
		if pindex.Height != nHeight {
			panic("the blockIndex height is incorrect")
		}
		// For every block except the genesis block, the chainWork must be
		// larger than the parent's.
		if pindex.Prev != nil && pindex.ChainWork.Cmp(&pindex.Prev.ChainWork) < 0 {
			panic("For every block except the genesis block, the chainWork must be " +
				"larger than the parent's.")
		}
		// The pskip pointer must point back for all but the first 2 blocks.
		if pindex.Height >= 2 && (pindex.Skip == nil || pindex.Skip.Height >= nHeight) {
			panic(" The pskip pointer must point back for all but the first 2 blocks.")
		}
		// All mapBlockIndex entries must at least be TREE valid
		if pindexFirstNotTreeValid != nil {
			panic("All mapBlockIndex entries must at least be TREE valid")
		}
		if pindex.Status&core.BlockValidMask >= core.BlockValidTree {
			// TREE valid implies all parents are TREE valid
			if pindexFirstNotTreeValid != nil {
				panic("status TREE valid implies all parents are TREE valid")
			}
		}
		if pindex.Status&core.BlockValidMask >= core.BlockValidChain {
			// CHAIN valid implies all parents are CHAIN valid
			if pindexFirstNotChainValid != nil {
				panic("status CHAIN valid implies all parents are CHAIN valid")
			}
		}
		if pindex.Status&core.BlockValidMask >= core.BlockValidScripts {
			// SCRIPTS valid implies all parents are SCRIPTS valid
			if pindexFirstNotScriptsValid != nil {
				panic("status SCRIPTS valid implies all parents are SCRIPTS valid")
			}
		}
		if pindexFirstInvalid == nil {
			// Checks for not-invalid blocks.
			// The failed mask cannot be set for blocks without invalid parents.
			if pindex.Status&core.BlockValidMask != 0 {
				panic("The failed mask cannot be set for blocks without invalid parents.")
			}
		}
		if !blockIndexWorkComparator(pindex, GChainState.ChainActive.Tip()) &&
			pindexFirstNeverProcessed == nil {
			if pindexFirstInvalid == nil {
				// If this block sorts at least as good as the current tip and
				// is valid and we have all data for its parents, it must be in
				// setBlockIndexCandidates. chainActive.Tip() must also be there
				// even if some data has been pruned.
				if pindexFirstMissing == nil || pindex == GChainState.ChainActive.Tip() {
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
				panic("the blockIndex should not be in setBlockIndexCandidates")
			}
		}
		// Check whether this block is in mapBlocksUnlinked.
		foundInUnlinked := false
		if rangeUnlinked, ok := GChainState.MapBlocksUnlinked[pindex.Prev]; ok {
			for i := 0; i < len(rangeUnlinked); i++ {
				if rangeUnlinked[i] == pindex {
					foundInUnlinked = true
					break
				}
			}
		}
		if pindex.Prev != nil && (pindex.Status&core.BlockHaveData != 0) &&
			pindexFirstNeverProcessed != nil && pindexFirstInvalid == nil {
			// If this block has block data available, some parent was never
			// received, and has no invalid parents, it must be in
			// mapBlocksUnlinked.
			if !foundInUnlinked {
				panic("the block must be in mapBlocksUnlinked")
			}
		}

		if !(pindex.Status&core.BlockHaveData != 0) {
			// Can't be in mapBlocksUnlinked if we don't have data
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
		if pindex.Prev != nil && (pindex.Status&core.BlockHaveData != 0) &&
			pindexFirstNeverProcessed == nil && pindexFirstMissing != nil {
			// We have data for this block, have received data for all parents
			// at some point, but we're currently missing data for some parent.
			// We must have pruned.
			if !GHavePruned {
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
			if blockIndexWorkComparator(pindex, GChainState.ChainActive.Tip()) &&
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
			pindexPar := pindex.Prev
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

func GetBlockFileInfo(n int) *BlockFileInfo {
	return gInfoBlockFile[n]
}

func VersionBitsTipState(param *msg.BitcoinParams, pos msg.DeploymentPos) ThresholdState {
	//todo:LOCK(cs_main)
	return VersionBitsState(GChainActive.Tip(), param, pos, &versionBitsCache)
}

func VersionBitsTipStateSinceHeight(params *msg.BitcoinParams, pos msg.DeploymentPos) int {
	//todo:LOCK(cs_main)
	return VersionBitsStateSinceHeight(GChainActive.Tip(), params, pos, &versionBitsCache)
}

func BlockIndexWorkComparator(pa, pb interface{}) bool {
	a := pa.(*core.BlockIndex)
	b := pb.(*core.BlockIndex)
	return blockIndexWorkComparator(a, b)
}

func blockIndexWorkComparator(pa, pb *core.BlockIndex) bool {
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
	a, err := strconv.ParseUint(fmt.Sprintf("%p", pa), 16, 0)
	if err != nil {
		panic("convert hex string to uint failed")
	}
	b, err := strconv.ParseUint(fmt.Sprintf("%p", pb), 16, 0)
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

type TraceEle struct {
	pindex *core.BlockIndex
	pblock *core.Block
}

type ConnectTrace struct {
	blocksConnected []TraceEle
}

// ActivateBestChain Make the best chain active, in multiple steps. The result is either failure
// or an activated best chain. pblock is either nullptr or a pointer to a block
// that is already loaded (to avoid loading it again from disk).
// Find the best known block, and make it the tip of the block chain
func ActivateBestChain(param *msg.BitcoinParams, state *core.ValidationState, pblock *core.Block) bool {
	// Note that while we're often called here from ProcessNewBlock, this is
	// far from a guarantee. Things in the P2P/RPC will often end up calling
	// us in the middle of ProcessNewBlock - do not assume pblock is set
	// sanely for performance or correctness!
	var (
		pindexMostWork *core.BlockIndex
		pindexNewTip   *core.BlockIndex
	)
	for {
		//	todo, Add channel for receive interruption from P2P/RPC
		var pindexFork *core.BlockIndex
		var connectTrace ConnectTrace
		fInitialDownload := false
		{
			// TODO !!! And sync.lock, cs_main
			// TODO: Tempoarily ensure that mempool removals are notified
			// before connected transactions. This shouldn't matter, but the
			// abandoned state of transactions in our wallet is currently
			// cleared when we receive another notification and there is a
			// race condition where notification of a connected conflict
			// might cause an outside process to abandon a transaction and
			// then have it inadvertantly cleared by the notification that
			// the conflicted transaction was evicted.
			mrt := mempool.NewMempoolConflictRemoveTrack(GMemPool)
			_ = mrt
			pindexOldTip := GChainState.ChainActive.Tip()
			if pindexMostWork == nil {
				pindexMostWork = FindMostWorkChain()
			}

			// Whether we have anything to do at all.
			if pindexMostWork == nil || pindexMostWork == GChainState.ChainActive.Tip() {
				return true
			}

			fInvalidFound := false
			var nullBlockPtr *core.Block
			var tmpBlock *core.Block
			hashA := pindexMostWork.GetBlockHash()
			if pblock != nil && bytes.Equal(pblock.Hash[:], hashA[:]) {
				tmpBlock = pblock
			} else {
				tmpBlock = nullBlockPtr
			}

			if !ActivateBestChainStep(param, state, pindexMostWork, tmpBlock, &fInvalidFound, &connectTrace) {
				return false
			}

			if fInvalidFound {
				// Wipe cache, we may need another branch now.
				pindexMostWork = nil
			}
			pindexNewTip = GChainState.ChainActive.Tip()
			pindexFork = GChainState.ChainActive.FindFork(pindexOldTip)
			fInitialDownload = IsInitialBlockDownload()
			_ = fInitialDownload
			// throw all transactions though the signal-interface

		} // MemPoolConflictRemovalTracker destroyed and conflict evictions
		// are notified

		// Transactions in the connnected block are notified
		for _, traElm := range connectTrace.blocksConnected {
			if traElm.pblock == nil {
				panic("the blockptr should not equivalent nil ")
			}
			for i, tx := range traElm.pblock.Txs {
				// TODO !!! send Asynchronous signal, noticed received transaction
				_ = i
				_ = tx
			}
		}

		// When we reach this point, we switched to a new tip (stored in
		// pindexNewTip).
		// Notifications/callbacks that can run without cs_main
		// Notify external listeners about the new tip.
		// TODO!!! send Asynchronous signal to external listeners.

		// Always notify the UI if a new block tip was connected
		if pindexFork != pindexNewTip {

		}
		if pindexNewTip == pindexMostWork {
			break
		}
	}

	GChainState.CheckBlockIndex(param)
	// Write changes periodically to disk, after relay.
	ok := FlushStateToDisk(state, FlushStatePeriodic, 0)
	return ok
}

func PreciousBlock(param *msg.BitcoinParams, state *core.ValidationState, pindex *core.BlockIndex) bool {
	//todo:LOCK(cs_main)
	if pindex.ChainWork.Cmp(&GChainActive.Tip().ChainWork) < 0 {
		// Nothing to do, this block is not at the tip.
		return true
	}
	if GChainActive.Tip().ChainWork.Cmp(&gLastPreciousChainWork) > 0 {
		// The chain has been extended since the last call, reset the
		// counter.
		gBlockReverseSequenceID = -1
	}
	gLastPreciousChainWork = GChainActive.Tip().ChainWork
	gSetDirtyBlockIndex.RemoveItem(pindex)
	pindex.SequenceID = gBlockSequenceID
	if gBlockReverseSequenceID > math.MinInt64 {
		// We can't keep reducing the counter if somebody really wants to
		// call preciousBlock 2**31-1 times on the same set of tips...
		gBlockReverseSequenceID--
	}
	if pindex.IsValid(core.BlockValidTransactions) && pindex.ChainTxCount > 0 {
		gSetDirtyBlockIndex.AddItem(pindex)
		PruneBlockIndexCandidates()
	}
	return ActivateBestChain(param, state, nil)
}

func AcceptBlockHeader(param *msg.BitcoinParams, pblkHeader *core.BlockHeader,
	state *core.ValidationState, ppindex **core.BlockIndex) bool {
	// todo warning: be care of the pointer of pointer

	// Check for duplicate
	var pindex *core.BlockIndex
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
			if pindex.Status&core.BlockFailedMask != 0 {
				return state.Invalid(state.Error(fmt.Sprintf("block %s is marked invalid",
					hash.ToString())), 0, "duplicate", "")
			}
			return true
		}

		// todo !! Add log, when return false
		if !CheckBlockHeader(pblkHeader, state, param, true) {
			return false
		}

		// Get prev block index
		var pindexPrev *core.BlockIndex
		v, ok := GChainState.MapBlockIndex.Data[pblkHeader.HashPrevBlock]
		if !ok {
			return state.Dos(10, false, 0, "bad-prevblk",
				false, "")
		}
		pindexPrev = v

		if pindexPrev.Status&core.BlockFailedMask != 0 {
			return state.Dos(100, false, core.RejectInvalid, "bad-prevblk",
				false, "")
		}

		if pindexPrev == nil {
			panic("the pindexPrev should not be nil")
		}

		if GCheckpointsEnabled && !checkIndexAgainstCheckpoint(pindexPrev, state, param, &hash) {
			return false
		}

		// todo !! Add time param in the function
		mTime := MedianTime{}
		if !ContextualCheckBlockHeader(pblkHeader, state, param, pindexPrev, int64(mTime.AdjustedTime().Second())) {
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

func InsertBlockIndex(hash *utils.Hash) *core.BlockIndex {
	if hash.IsNull() {
		return nil
	}

	// Return existing
	mi, ok := MapBlockIndex.Data[*hash]
	if ok {
		return mi
	}

	// Create new
	index := &core.BlockIndex{}
	index.SetNull()
	if index == nil {
		panic("new CBlockIndex failed")
	}

	MapBlockIndex.Data[*hash] = index
	index.BlockHash = *hash

	return index
}

// ActivateBestChainStep Try to make some progress towards making pindexMostWork
// the active block. pblock is either nullptr or a pointer to a CBlock corresponding to
// pindexMostWork.
func ActivateBestChainStep(param *msg.BitcoinParams, state *core.ValidationState, pindexMostWork *core.BlockIndex,
	pblock *core.Block, fInvalidFound *bool, connectTrace *ConnectTrace) bool {

	//todo !!! add sync.mutex lock; cs_main
	pindexOldTip := GChainState.ChainActive.Tip()
	pindexFork := GChainState.ChainActive.FindFork(pindexMostWork)

	// Disconnect active blocks which are no longer in the best chain.
	fBlocksDisconnected := false
	for GChainState.ChainActive.Tip() != nil && GChainState.ChainActive.Tip() != pindexFork {
		if !DisconnectTip(param, state, false) {
			return false
		}
		fBlocksDisconnected = true
	}

	// Build list of new blocks to connect.
	vpindexToConnect := make([]*core.BlockIndex, 0)
	fContinue := true
	nHeight := -1
	if pindexFork != nil {
		nHeight = pindexFork.Height
	}
	for fContinue && nHeight != pindexFork.Height {
		// Don't iterate the entire list of potential improvements toward the
		// best tip, as we likely only need a few blocks along the way.
		nTargetHeight := pindexMostWork.Height
		if nHeight+32 < pindexMostWork.Height {
			nTargetHeight = nHeight + 32
		}
		vpindexToConnect = make([]*core.BlockIndex, 0)
		pindexIter := pindexMostWork.GetAncestor(nTargetHeight)
		for pindexIter != nil && pindexIter.Height != nHeight {
			vpindexToConnect = append(vpindexToConnect, pindexIter)
			pindexIter = pindexIter.Prev
		}
		nHeight = nTargetHeight

		// Connect new blocks.
		var pindexConnect *core.BlockIndex
		if len(vpindexToConnect) > 0 {
			pindexConnect = vpindexToConnect[len(vpindexToConnect)-1]
		}
		for pindexConnect != nil {
			tmpBlock := pblock
			if pindexConnect != pindexMostWork {
				tmpBlock = nil
			}
			if !ConnectTip(param, state, pindexConnect, tmpBlock, connectTrace) {
				if state.IsInvalid() {
					// The block violates a core rule.
					if !state.CorruptionPossible() {
						InvalidChainFound(vpindexToConnect[len(vpindexToConnect)-1])
					}
					state = core.NewValidationState()
					*fInvalidFound = true
					fContinue = false
					// If we didn't actually connect the block, don't notify
					// listeners about it
					connectTrace.blocksConnected = connectTrace.blocksConnected[:len(connectTrace.blocksConnected)-1]
					break
				} else {
					// A system error occurred (disk space, database error, ...)
					return false
				}
			} else {
				PruneBlockIndexCandidates()
				if pindexOldTip == nil || GChainState.ChainActive.Tip().ChainWork.Cmp(&pindexOldTip.ChainWork) > 0 {
					// We're in a better position than we were. Return temporarily to release the lock.
					fContinue = false
					break
				}
			}
		}
	}

	if fBlocksDisconnected {
		RemoveForReorg(Pool, GCoinsTip, GChainState.ChainActive.Tip().Height+1, int(policy.StandardLockTimeVerifyFlags))
		LimitMempoolSize(GMemPool, utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize))*1000000,
			utils.GetArg("-mempoolexpiry", int64(DefaultMemPoolExpiry))*60*60)
	}
	GMemPool.Check(GCoinsTip)

	// Callbacks/notifications for a new best chain.
	if *fInvalidFound {
		CheckForkWarningConditionsOnNewFork(vpindexToConnect[len(vpindexToConnect)-1])
	} else {
		CheckForkWarningConditions()
	}
	return true
}

func InvalidChainFound(pindexNew *core.BlockIndex) {
	if gIndexBestInvalid == nil || pindexNew.ChainWork.Cmp(&gIndexBestInvalid.ChainWork) > 0 {
		gIndexBestInvalid = pindexNew
	}
	logger.GetLogger().Info("%s: invalid block=%s  height=%d  work=%.8d  date=%s\n",
		logger.TraceLog(), pindexNew.BlockHash.ToString(), pindexNew.Height,
		pindexNew.ChainWork.String(), time.Unix(int64(pindexNew.GetBlockTime()), 0).String())
	tip := GChainState.ChainActive.Tip()
	if tip == nil {
		panic("Now, the chain Tip should not equal nil")
	}
	logger.GetLogger().Debug("%s:  current best=%s  height=%d  log2_work=%.8g  date=%s\n",
		logger.TraceLog(), tip.BlockHash.ToString(), GChainState.ChainActive.Height(),
		time.Unix(int64(tip.GetBlockTime()), 0).String())

	CheckForkWarningConditions()
}

// PruneBlockIndexCandidates Delete all entries in setBlockIndexCandidates that
// are worse than the current tip.
func PruneBlockIndexCandidates() {
	// Note that we can't delete the current block itself, as we may need to
	// return to it later in case a reorganization to a better block fails.
	for i := 0; i < GChainState.setBlockIndexCandidates.Size(); i++ {
		pindex := GChainState.setBlockIndexCandidates.GetItemByIndex(i).(*core.BlockIndex)
		if blockIndexWorkComparator(pindex, GChainState.ChainActive.Tip()) {
			GChainState.setBlockIndexCandidates.DelItem(pindex)
		}
	}
	// Either the current tip or a successor of it we're working towards is left
	// in setBlockIndexCandidates.
	if GChainState.setBlockIndexCandidates.Size() > 0 {
		panic("the set should have element, ")
	}
}

func CheckForkWarningConditionsOnNewFork(pindexNewForkTip *core.BlockIndex) {
	//todo !!! add sync.mutex lock; cs_main
	// If we are on a fork that is sufficiently large, set a warning flag
	pfork := pindexNewForkTip
	plonger := GChainState.ChainActive.Tip()
	for pfork != nil && pfork != plonger {
		for plonger != nil && plonger.Height > pfork.Height {
			plonger = plonger.Prev
		}
		if pfork == plonger {
			break
		}
		pfork = pfork.Prev
	}

	// We define a condition where we should warn the user about as a fork of at
	// least 7 blocks with a tip within 72 blocks (+/- 12 hours if no one mines
	// it) of ours. We use 7 blocks rather arbitrarily as it represents just
	// under 10% of sustained network hash rate operating on the fork, or a
	// chain that is entirely longer than ours and invalid (note that this
	// should be detected by both). We define it this way because it allows us
	// to only store the highest fork tip (+ base) which meets the 7-block
	// condition and from this always have the most-likely-to-cause-warning fork
	if pfork != nil &&
		(pindexNewForkTip == nil || (pindexNewForkTip != nil &&
			pindexNewForkTip.Height > gIndexBestForkTip.Height)) {
		gIndexBestForkTip = pindexNewForkTip
		gIndexBestForkBase = pfork
	}
	CheckForkWarningConditions()
}

func CheckForkWarningConditions() {
	//todo !!! add sync.lock, cs_main
	// Before we get past initial download, we cannot reliably alert about forks
	// (we assume we don't get stuck on a fork before finishing our initial
	// sync)
	if IsInitialBlockDownload() {
		return
	}

	// If our best fork is no longer within 72 blocks (+/- 12 hours if no one
	// mines it) of our head, drop it
	if gIndexBestForkTip != nil &&
		GChainState.ChainActive.Height()-gIndexBestForkTip.Height >= 72 {
		gIndexBestForkTip = nil
	}

	tmpWork := big.NewInt(0).Add(&GChainState.ChainActive.Tip().ChainWork, big.NewInt(0).Mul(GetBlockProof(GChainState.ChainActive.Tip()), big.NewInt(6)))
	if gIndexBestForkTip != nil || (gIndexBestInvalid != nil &&
		gIndexBestInvalid.ChainWork.Cmp(tmpWork) > 0) {
		if !msg.GetfLargeWorkForkFound() && gIndexBestForkBase != nil {
			waring := "'Warning: Large-work fork detected, forking after block " +
				gIndexBestForkBase.BlockHash.ToString() + "'"
			AlertNotify(waring)
		}

		if gIndexBestForkTip != nil && gIndexBestForkBase != nil {
			logger.GetLogger().Debug("%s: Warning: Large valid fork found forking the "+
				"chain at height %d (%s) lasting to height %d (%s).\n"+
				"Chain state database corruption likely.\n", logger.TraceLog(), gIndexBestForkBase.Height,
				gIndexBestForkBase.BlockHash.ToString(), gIndexBestForkTip.Height,
				gIndexBestForkTip.BlockHash.ToString())
			msg.SetfLargeWorkForkFound(true)
		} else {
			logger.GetLogger().Debug("%s: Warning: Found invalid chain at least ~6 blocks "+
				"longer than our best chain.\nChain state database "+
				"corruption likely.\n", logger.TraceLog())
			msg.SetfLargeWorkInvalidChainFound(true)
		}
	} else {
		msg.SetfLargeWorkForkFound(false)
		msg.SetfLargeWorkInvalidChainFound(false)
	}
}

// ConnectTip Connect a new block to chainActive. pblock is either nullptr or a pointer to
// a CBlock corresponding to pindexNew, to bypass loading it again from disk.
// The block is always added to connectTrace (either after loading from disk or
// by copying pblock) - if that is not intended, care must be taken to remove
// the last entry in blocksConnected in case of failure.
func ConnectTip(param *msg.BitcoinParams, state *core.ValidationState, pindexNew *core.BlockIndex,
	pblock *core.Block, connectTrace *ConnectTrace) bool {

	if pindexNew.Prev != GChainState.ChainActive.Tip() {
		panic("the ")
	}
	// Read block from disk.
	nTime1 := utils.GetMicrosTime()
	if pblock == nil {
		var pblockNew *core.Block
		var tmpTrace TraceEle
		tmpTrace.pindex = pindexNew
		tmpTrace.pblock = pblockNew
		connectTrace.blocksConnected = append(connectTrace.blocksConnected, tmpTrace)
		if !ReadBlockFromDisk(pblockNew, pindexNew, param) {
			return AbortNode(state, "Failed to read block", "")
		}
	} else {
		var tmpTrace TraceEle
		tmpTrace.pblock = pblock
		tmpTrace.pindex = pindexNew
		connectTrace.blocksConnected = append(connectTrace.blocksConnected, tmpTrace)
	}
	blockConnecting := *(connectTrace.blocksConnected[len(connectTrace.blocksConnected)-1].pblock)
	// Apply the block atomically to the chain state.
	nTime2 := utils.GetMicrosTime()
	gTimeReadFromDisk += nTime2 - nTime1
	view := utxo.NewCoinViewCacheByCoinview(GCoinsTip)
	rv := ConnectBlock(param, &blockConnecting, state, pindexNew, view, false)
	//todo !!! GetMainSignals().BlockChecked(blockConnecting, state);
	if !rv {
		if state.IsInvalid() {
			InvalidBlockFound(pindexNew, state)
		}
		hash := pindexNew.GetBlockHash()
		return logger.ErrorLog(fmt.Sprintf("ConnectTip(): ConnectBlock %s failed", hash.ToString()))
	}
	nTime3 := utils.GetMicrosTime()
	gTimeConnectTotal += nTime3 - nTime2
	logger.LogPrint("bench", "debug", " - Connect total: %.2fms [%.2fs]\n",
		float64(nTime3-nTime2)*0.001, float64(gTimeConnectTotal)*0.000001)
	flushed := view.Flush()
	if !flushed {
		panic("here should be true when view flush state")
	}
	nTime4 := utils.GetMicrosTime()
	gTimeFlush += nTime4 - nTime3
	logger.LogPrint("bench", "debug", " - Flush: %.2fms [%.2fs]\n",
		float64(nTime4-nTime3)*0.001, float64(gTimeFlush)*0.000001)
	// Write the chain state to disk, if necessary.
	if !FlushStateToDisk(state, FlushStateIfNeeded, 0) {
		return false
	}
	nTime5 := utils.GetMicrosTime()
	gTimeChainState += nTime5 - nTime4
	logger.LogPrint("bench", "debug", " - Writing chainstate: %.2fms [%.2fs]\n",
		float64(nTime5-nTime4)*0.001, float64(gTimeChainState)*0.000001)
	// Remove conflicting transactions from the mempool.;
	GMemPool.RemoveForBlock(blockConnecting.Txs, uint(pindexNew.Height))
	// Update chainActive & related variables.
	UpdateTip(param, pindexNew)
	nTime6 := utils.GetMicrosTime()
	gTimePostConnect += nTime6 - nTime1
	gTimeTotal += nTime6 - nTime1
	logger.LogPrint("bench", "debug", " - Connect postprocess: %.2fms [%.2fs]\n",
		float64(nTime6-nTime5)*0.001, float64(gTimePostConnect)*0.000001)
	logger.LogPrint("bench", "debug", " - Connect block: %.2fms [%.2fs]\n",
		float64(nTime6-nTime1)*0.001, float64(gTimeTotal)*0.000001)

	return true
}

func InvalidBlockFound(pindex *core.BlockIndex, state *core.ValidationState) {

}

func GetBlockSubsidy(height int, params msg.BitcoinParams) utils.Amount {
	halvings := height / int(params.SubsidyReductionInterval)
	// Force block reward to zero when right shift is undefined.
	if halvings >= 64 {
		return 0
	}

	nSubsidy := utils.Amount(50 * utils.COIN)
	// Subsidy is cut in half every 210,000 blocks which will occur
	// approximately every 4 years.
	return utils.Amount(uint(nSubsidy) >> uint(halvings))
}

func FindUndoPos(state *core.ValidationState, nFile int, pos *core.DiskBlockPos, nAddSize int) bool {
	pos.File = nFile
	//TODO:LOCK(cs_LastBlockFile);
	pos.Pos = int(gInfoBlockFile[nFile].UndoSize)
	gInfoBlockFile[nFile].UndoSize += uint32(nAddSize)
	nNewSize := gInfoBlockFile[nFile].UndoSize
	gSetDirtyFileInfo.AddItem(nFile)

	nOldChunks := (pos.Pos + UndoFileChunkSize - 1) / UndoFileChunkSize
	nNewChunks := (nNewSize + UndoFileChunkSize - 1) / UndoFileChunkSize

	if nNewChunks > uint32(nOldChunks) {
		if GPruneMode {
			GCheckForPruning = true
		}
		if CheckDiskSpace(nNewChunks*UndoFileChunkSize - uint32(pos.Pos)) {
			file := OpenUndoFile(*pos, false)
			if file != nil {
				logger.GetLogger().Info("Pre-allocating up to position 0x%x in rev%05u.dat\n",
					nNewChunks*UndoFileChunkSize, pos.File)
				AllocateFileRange(file, pos.Pos, nNewChunks*UndoFileChunkSize-uint32(pos.Pos))
				file.Close()
			}
		} else {
			return state.Error("out of disk space")
		}
	}

	return true
}

func ThreadScriptCheck() {
	//todo: RenameThread("bitcoin-scriptch")
	//		scriptcheckqueue.Thread()
}

func ConnectBlock(param *msg.BitcoinParams, pblock *core.Block, state *core.ValidationState,
	pindex *core.BlockIndex, view *utxo.CoinsViewCache, fJustCheck bool) bool {

	//TODO: AssertLockHeld(cs_main);
	//var sc sync.RWMutex
	//sc.Lock()
	//defer sc.Unlock()

	nTimeStart := utils.GetMicrosTime()

	// Check it again in case a previous version let a bad block in
	if !CheckBlock(param, pblock, state, !fJustCheck, !fJustCheck) {
		return logger.ErrorLog(fmt.Sprintf("CheckBlock: %s", FormatStateMessage(state)))
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
	logger.LogPrint("bench", "debug", " - Sanity checks: %.2fms [%.2fs]\n", 0.001*float64(nTime1-nTimeStart), float64(gTimeCheck)*0.000001)

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
	fEnforceBIP30 = fEnforceBIP30 && (&pindexBIP34height == nil || !(*pindexBIP34height.GetBlockHash() == param.BIP34Hash))

	if fEnforceBIP30 {
		for _, tx := range pblock.Txs {
			for o := 0; o < len(tx.Outs); o++ {
				outPoint := &core.OutPoint{
					Hash:  tx.Hash,
					Index: uint32(o),
				}
				if view.HaveCoin(outPoint) {
					return state.Dos(100, false, core.RejectInvalid, "bad-txns-BIP30", false, "")
				}
			}
		}
	}

	// Start enforcing BIP68 (sequence locks) using versionBits logic.
	nLockTimeFlags := 0
	if VersionBitsState(pindex.Prev, param, msg.DeploymentCSV, &versionBitsCache) == ThresholdActive {
		nLockTimeFlags |= core.LocktimeVerifySequence
	}

	flags := GetBlockScriptFlags(pindex, param)
	nTime2 := utils.GetMicrosTime()
	gTimeForks += nTime2 - nTime1
	logger.LogPrint("bench", "debug", " - Fork checks: %.2fms [%.2fs]\n",
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
	nMaxSigOpsCount := core.GetMaxBlockSigOpsCount(uint64(currentBlockSize))

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
				return state.Dos(100, logger.ErrorLog("ConnectBlock(): inputs missing/spent"), core.RejectInvalid, "bad-txns-inputs-missIngorSpent", false, "")
			}

			// Check that transaction is BIP68 final BIP68 lock checks (as
			// opposed to nLockTime checks) must be in ConnectBlock because they
			// require the UTXO set.
			for j := 0; j < len(tx.Ins); j++ {
				prevheights[j] = int(view.AccessCoin(tx.Ins[j].PreviousOutPoint).GetHeight())
			}

			if !SequenceLocks(tx, nLockTimeFlags, prevheights, pindex) {
				return state.Dos(100, logger.ErrorLog("contains a non-BIP68-final transaction"),
					core.RejectInvalid, "bad-txns-nonFinal", false, "")
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
			return state.Dos(100, logger.ErrorLog("ConnectBlock(): too many sigOps"),
				core.RejectInvalid, "bad-blk-sigops", false, "")
		}

		if !tx.IsCoinBase() {
			fee := view.GetValueIn(tx) - utils.Amount(tx.GetValueOut())
			nFees += fee
			// Don't cache results if we're actually connecting blocks (still consult the cache, though).
			fCacheResults := fJustCheck
			vChecks := make([]*ScriptCheck, 0)
			if !CheckInputs(tx, state, view, fScriptChecks, flags, fCacheResults, fCacheResults,
				core.NewPrecomputedTransactionData(tx), vChecks) {
				return logger.ErrorLog(fmt.Sprintf("ConnectBlock(): CheckInputs on %s failed with %s",
					tx.TxHash(), FormatStateMessage(state)))
			}

			//todo:control.add(vChecks)
		}

		var undoDummy TxUndo
		if i > 0 {
			blockundo.txundo = append(blockundo.txundo, newTxUndo())
		}
		if i == 0 {
			UpdateCoins(tx, view, &undoDummy, pindex.Height)
		} else {
			UpdateCoins(tx, view, blockundo.txundo[len(blockundo.txundo)-1], pindex.Height)
		}

		vPos[tx.Hash] = *txPos
		txPos.TxOffsetIn += tx.SerializeSize()
	}

	nTime3 := utils.GetMicrosTime()
	gTimeConnect += nTime3 - nTime2
	if nInputs <= 1 {
		logger.LogPrint("bench", "debug", " - Connect %u transactions: %.2fms (%.3fms/tx, %.3fms/txin) [%.2fs]\n",
			len(pblock.Txs), 0.001*float64(nTime3-nTime2), 0.001*float64(nTime3-nTime2)/float64(len(pblock.Txs)), 0, float64(gTimeConnect)*0.000001)
	} else {
		logger.LogPrint("bench", "debug", " - Connect %u transactions: %.2fms (%.3fms/tx, %.3fms/txin) [%.2fs]\n",
			len(pblock.Txs), 0.001*float64(nTime3-nTime2), 0.001*float64(nTime3-nTime2)/float64(len(pblock.Txs)),
			0.001*float64(nTime3-nTime2)/float64(nInputs-1), float64(gTimeConnect)*0.000001)
	}

	blockReward := nFees + GetBlockSubsidy(pindex.Height, *param)

	if pblock.Txs[0].GetValueOut() > int64(blockReward) {
		return state.Dos(100, logger.ErrorLog("ConnectBlock(): coinbase pays too much "),
			core.RejectInvalid, "bad-cb-amount", false, "")
	}

	//todo:control

	nTime4 := utils.GetMicrosTime()
	gTimeVerify += nTime4 - nTime2

	if nInputs <= 1 {
		logger.LogPrint("bench", "debug", " - Verify %u txIns: %.2fms (%.3fms/txIn) [%.2fs]\n",
			nInputs-1, 0.001*float64(nTime4-nTime2), 0, float64(gTimeVerify)*0.000001)
	} else {
		logger.LogPrint("bench", "debug", " - Verify %u txIns: %.2fms (%.3fms/txIn) [%.2fs]\n",
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
			//todo：SerializeSize
			//if !FindUndoPos(state, pindex.File, pos, len(blockundo.)) {
			//	logger.ErrorLog("ConnectBlock(): FindUndoPos failed")
			//}
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

	if GTxIndex { //todo:
		return AbortNode(state, "Failed to write transaction index", "")
	}

	// add this block to the view's block chain
	view.SetBestBlock(*pindex.GetBlockHash())

	nTime5 := utils.GetMicrosTime()
	gTimeIndex += nTime5 - nTime4
	logger.LogPrint("bench", "debug", " - Index writing: %.2fms [%.2fs]\n",
		0.001*float64(nTime5-nTime4), float64(gTimeIndex)*0.000001)

	// Watch for changes to the previous coinbase transaction.
	//todo:GetMainSignals().UpdatedTransaction(hashPrevBestCoinBase);
	gHashPrevBestCoinBase = pblock.Txs[0].Hash

	nTime6 := utils.GetMicrosTime()
	gTimeCallbacks += nTime6 - nTime5
	logger.LogPrint("bench", "debug", " - Callbacks: %.2fms [%.2fs]\n",
		0.001*float64(nTime6-nTime5), float64(gTimeCallbacks)*0.000001)
	return true
}

// DisconnectTip Disconnect chainActive's tip. You probably want to call
// mempool.removeForReorg and manually re-limit mempool size after this, with
// cs_main held.
func DisconnectTip(param *msg.BitcoinParams, state *core.ValidationState, fBare bool) bool {

	pindexDelete := GChainState.ChainActive.Tip()
	if pindexDelete == nil {
		panic("the chain tip element should not equal nil")
	}
	// Read block from disk.
	var block core.Block
	if !ReadBlockFromDisk(&block, pindexDelete, param) {
		return AbortNode(state, "Failed to read block", "")
	}

	// Apply the block atomically to the chain state.
	nStart := utils.GetMockTimeInMicros()
	{
		view := utxo.NewCoinViewCacheByCoinview(GCoinsTip)
		hash := pindexDelete.GetBlockHash()
		if DisconnectBlock(&block, pindexDelete, view) != DisconnectOk {
			return logger.ErrorLog(fmt.Sprintf("DisconnectTip(): DisconnectBlock %s failed ", hash.ToString()))
		}
		flushed := view.Flush()
		if !flushed {
			panic("view flush error !!!")
		}
	}
	// replace implement with LogPrint(in C++).
	logger.LogPrint("bench", "debug", " - Disconnect block : %.2fms\n",
		float64(utils.GetMicrosTime()-nStart)*0.001)

	// Write the chain state to disk, if necessary.
	if !FlushStateToDisk(state, FlushStateIfNeeded, 0) {
		return false
	}

	if !fBare {
		// Resurrect mempool transactions from the disconnected block.
		vHashUpdate := container.Vector{}
		for _, tx := range block.Txs {
			// ignore validation errors in resurrected transactions
			var stateDummy core.ValidationState
			if tx.IsCoinBase() || !AcceptToMemoryPool(param, GMemPool, &stateDummy, tx, false, nil, nil, true, 0) {
				GMemPool.RemoveRecursive(tx, mempool.REORG)
			} else if GMemPool.Exists(tx.Hash) {
				vHashUpdate.PushBack(tx.Hash)
			}
		}
		// AcceptToMemoryPool/addUnchecked all assume that new memPool entries
		// have no in-memPool children, which is generally not true when adding
		// previously-confirmed transactions back to the memPool.
		// UpdateTransactionsFromBlock finds descendants of any transactions in
		// this block that were added back and cleans up the memPool state.
		GMemPool.UpdateTransactionsFromBlock(vHashUpdate)
	}

	// Update chainActive and related variables.
	UpdateTip(param, pindexDelete.Prev)
	// Let wallets know transactions went from 1-confirmed to
	// 0-confirmed or conflicted:
	for _, tx := range block.Txs {
		//todo !!! add  GetMainSignals().SyncTransaction()
		_ = tx
	}
	return true
}

// UpdateTip Update chainActive and related internal data structures.
func UpdateTip(param *msg.BitcoinParams, pindexNew *core.BlockIndex) {
	GChainState.ChainActive.SetTip(pindexNew)
	// New best block
	GMemPool.AddTransactionsUpdated(1)

	//	TODO !!! add Parallel Programming boost::condition_variable
	warningMessages := make([]string, 0)
	if !IsInitialBlockDownload() {
		nUpgraded := 0
		pindex := GChainState.ChainActive.Tip()
		for bit := 0; bit < VersionBitsNumBits; bit++ {
			checker := NewWarningBitsConChecker(bit)
			state := GetStateFor(checker, pindex, param, GWarningCache[bit])
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
		for i := 0; i < 100 && pindex != nil; i++ {
			nExpectedVersion := ComputeBlockVersion(pindex.Prev, param, GVersionBitsCache)
			if pindex.Version > VersionBitsLastOldBlockVersion &&
				(int(pindex.Version)&(^nExpectedVersion) != 0) {
				nUpgraded++
				pindex = pindex.Prev
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
	logger.GetLogger().Info("%s: new best=%s height=%d version=0x%08x work=%.8g tx=%lu "+
		"date='%s' progress=%f cache=%.1f(%utxo)", logger.TraceLog(), GChainState.ChainActive.Tip().BlockHash.ToString(),
		GChainState.ChainActive.Height(), GChainState.ChainActive.Tip().Version,
		GChainState.ChainActive.Tip().ChainWork.String(), GChainState.ChainActive.Tip().ChainTxCount,
		time.Unix(int64(GChainState.ChainActive.Tip().Time), 0).String(),
		GuessVerificationProgress(txdata, GChainState.ChainActive.Tip()),
		GCoinsTip.DynamicMemoryUsage(), GCoinsTip.GetCacheSize())
	if len(warningMessages) != 0 {
		logger.GetLogger().Info("waring= %s", strings.Join(warningMessages, ","))
	}
}

func AlertNotify(strMessage string) {
	//todo !!! uiInterface.NotifyAlertChanged();
	strCmd := utils.GetArgString("-alertnotify", "")
	if len(strCmd) == 0 {
		return
	}

	// Alert text should be plain ascii coming from a trusted source, but to be
	// safe we first strip anything not in safeChars, then add single quotes
	// around the whole string before passing it to the shell:

}

func AcceptToMemoryPool(param *msg.BitcoinParams, pool *mempool.Mempool, state *core.ValidationState,
	tx *core.Tx, limitFree bool, missingInputs *bool, txnReplaced *list.List,
	overrideMempoolLimit bool, absurdFee utils.Amount) bool {

	return AcceptToMemoryPoolWithTime(param, pool, state, tx, limitFree, missingInputs,
		utils.GetMockTime(), txnReplaced, overrideMempoolLimit, absurdFee)
}

func GetTransaction(param *msg.BitcoinParams, txid *utils.Hash, txOut *core.Tx, hashBlock *utils.Hash, fAllowSlow bool) (ret bool) {
	var pindexSlow *core.BlockIndex
	//todo:LOCK(cs_main)

	ptx := mempool.GetTxFromMemPool(*txid)
	if ptx != nil {
		txOut = ptx
		return true
	}

	if GTxIndex {
		var postx core.DiskTxPos
		if GBlockTree.ReadTxIndex(txid, &postx) {
			file := OpenBlockFile(postx.BlockIn, true)
			if file == nil {
				return logger.ErrorLog("GetTransaction:OpenBlockFile failed")
			}
			ret = true
			defer func() {
				if err := recover(); err != nil {
					logger.ErrorLog(fmt.Sprintf("%s: Deserialize or I/O error -%s", logger.TraceLog(), err))
					ret = false
				}
			}()
			var header core.BlockHeader
			header.Serialize(file)
			file.Seek(int64(postx.TxOffsetIn), 1)
			txOut.Serialize(file)
			var err error
			*hashBlock, err = header.GetHash()
			if txOut.TxHash() != *txid && err != nil {
				return logger.ErrorLog(fmt.Sprintf("%s: txid mismatch", logger.TraceLog()))
			}
			return true
		}
	}

	// use coin database to locate block that contains transaction, and scan it
	if fAllowSlow {
		coin := utxo.AccessByTxid(GCoinsTip, txid)
		if !coin.IsSpent() {
			pindexSlow = GChainActive.Chain[coin.GetHeight()]
		}
	}

	if pindexSlow != nil {
		var block core.Block
		if ReadBlockFromDisk(&block, pindexSlow, param) {
			for _, tx := range block.Txs {
				if tx.TxHash() == *txid {
					txOut = tx
					hashBlock = pindexSlow.GetBlockHash()
					return true
				}
			}
		}
	}

	return false
}

// DisconnectBlock Undo the effects of this block (with given index) on the UTXO
// set represented by coins. When UNCLEAN or FAILED is returned, view is left in an
// indeterminate state.
func DisconnectBlock(pblock *core.Block, pindex *core.BlockIndex, view *utxo.CoinsViewCache) DisconnectResult {

	hashA := pindex.GetBlockHash()
	hashB := view.GetBestBlock()
	if !bytes.Equal(hashA[:], hashB[:]) {
		panic("the two hash should be equal ...")
	}
	var blockUndo BlockUndo
	pos := pindex.GetUndoPos()
	if pos.IsNull() {
		logger.ErrorLog("DisconnectBlock(): no undo data available")
		return DisconnectFailed
	}

	if !UndoReadFromDisk(&blockUndo, &pos, *pindex.Prev.GetBlockHash()) {
		logger.ErrorLog("DisconnectBlock(): failure reading undo data")
		return DisconnectFailed
	}

	return ApplyBlockUndo(&blockUndo, pblock, pindex, view)
}

func UndoWriteToDisk(blockundo *BlockUndo, pos *core.DiskBlockPos, hashBlock utils.Hash, messageStart utils.BitcoinNet) bool {
	// Open history file to append
	fileout := OpenUndoFile(*pos, false)
	if fileout == nil {
		return logger.ErrorLog("OpenUndoFile failed")
	}

	// Write index header
	nSize := 0 //todo:nSize = GetSerializeSize(fileout, block);
	err := utils.BinarySerializer.PutUint32(fileout, binary.LittleEndian, uint32(messageStart))
	if err != nil {
		logger.ErrorLog("the messageStart write failed")
	}
	utils.WriteVarInt(fileout, uint64(nSize))

	// Write undo data
	fileOutPos, err := fileout.Seek(0, 1)
	if fileOutPos < 0 || err != nil {
		return logger.ErrorLog("UndoWriteToDisk: ftell failed")
	}
	pos.Pos = int(fileOutPos)
	blockundo.Serialize(fileout)

	// calculate & write checksum
	//todo:continue
	return true
}

func UndoReadFromDisk(blockundo *BlockUndo, pos *core.DiskBlockPos, hashblock utils.Hash) (ret bool) {
	ret = true
	defer func() {
		if err := recover(); err != nil {
			logger.ErrorLog(fmt.Sprintf("%s: Deserialize or I/O error - %v", logger.TraceLog(), err))
			ret = false
		}
	}()
	file := OpenUndoFile(*pos, true)
	if file == nil {
		return logger.ErrorLog(fmt.Sprintf("%s: OpenUndoFile failed", logger.TraceLog()))
	}

	// Read block
	var hashCheckSum utils.Hash
	ok := hashblock.Serialize(file)
	if !ok {
		return ok
	}
	blockundo, err := DeserializeBlockUndo(file)
	if err != nil {
		return false
	}
	ok = hashCheckSum.Deserialize(file)

	// Verify checksum
	//todo !!! add if bytes.Equal(hashCheckSum[:], )

	return ok
}

func ReadBlockFromDisk(pblock *core.Block, pindex *core.BlockIndex, param *msg.BitcoinParams) bool {
	if !ReadBlockFromDiskByPos(pblock, pindex.GetBlockPos(), param) {
		return false
	}
	hash := pindex.GetBlockHash()
	pos := pindex.GetBlockPos()
	if bytes.Equal(pblock.Hash[:], hash[:]) {
		return logger.ErrorLog(fmt.Sprintf("ReadBlockFromDisk(CBlock&, CBlockIndex*): GetHash()"+
			"doesn't match index for %s at %s", pindex.ToString(), pos.ToString()))
	}
	return true
}

func ReadBlockFromDiskByPos(pblock *core.Block, pos core.DiskBlockPos, param *msg.BitcoinParams) bool {
	pblock.SetNull()

	// Open history file to read
	file := OpenBlockFile(&pos, true)
	if file == nil {
		return logger.ErrorLog(fmt.Sprintf("ReadBlockFromDisk: OpenBlockFile failed for %s", pos.ToString()))
	}

	// Read block
	if err := pblock.Deserialize(file); err != nil {
		return logger.ErrorLog(fmt.Sprintf("%s: Deserialize or I/O error - %v at %s", logger.TraceLog(),
			err, pos.ToString()))
	}

	// Check the header
	pow := Pow{}
	if !pow.CheckProofOfWork(&pblock.Hash, pblock.BlockHeader.Bits, param) {
		return logger.ErrorLog(fmt.Sprintf("ReadBlockFromDisk: Errors in block header at %s", pos.ToString()))
	}
	return true
}

func VerifyDB(params *msg.BitcoinParams, view *utxo.CoinsView, checkLevel int, checkDepth int) bool {
	// todo Lock(cs_main)

	if GChainActive.Tip() == nil || GChainActive.Tip().Prev == nil {
		return true
	}

	// Verify blocks in the best chain
	if checkDepth <= 0 {
		// suffices until the year 19000
		checkDepth = 1000000000
	}
	if checkDepth > GChainActive.Height() {
		checkDepth = GChainActive.Height()
	}

	// to calculate min(checkLevel, 4)
	tmpNum := utils.Min(4, checkLevel)

	// to calculate max(0, min(checkLevel, 4))
	checkLevel = utils.Max(0, tmpNum)

	logger.GetLogger().Debug("Verifying last %d blocks at level %d", checkDepth, checkLevel)

	coins := utxo.NewCoinViewCacheByCoinview(*view)
	indexState := GChainActive.Tip()
	var indexFailure *core.BlockIndex
	var goodTransactions uint32
	state := core.NewValidationState()
	var reportDone int
	logger.GetLogger().Debug("[0%%]...")
	for index := GChainActive.Tip(); index != nil && index.Prev != nil; index = index.Prev {
		// todo boost::this_thread::interruption_point()

		var tmp int
		if checkLevel >= 4 {
			tmp = 50
		} else {
			tmp = 100
		}
		percentageDone := utils.Max(1, utils.Min(99,
			int(float64(GChainActive.Height()-index.Height)/float64(checkDepth))*tmp))

		if reportDone < percentageDone/10 {
			// report every 10% step
			logger.GetLogger().Debug("[%d%%]...", percentageDone)
			reportDone = percentageDone / 10
		}

		// gui notify
		// uiInterface.ShowProgress(_("Verifying blocks..."), percentageDone);
		if index.Height < GChainActive.Height()-checkDepth {
			break
		}

		if GPruneMode && (index.Status&core.BlockHaveData) == 0 {
			// If pruning, only go back as far as we have data.
			logger.GetLogger().Debug("VerifyDB(): block verification stopping at height"+
				" %d (pruning, no data)", index.Height)
			break
		}

		block := core.NewBlock()
		// check level 0: read from disk
		if !ReadBlockFromDisk(block, index, params) {
			return logger.ErrorLog("VerifyDB(): *** ReadBlockFromDisk failed at %d, hash=%s",
				index.Height, index.GetBlockHash().ToString())
		}

		// check level 1: verify block validity
		if checkLevel >= 1 && !CheckBlock(params, block, state, true, true) {
			return logger.ErrorLog("VerifyDB(): *** found bad block at %d, hash=%s (%s)\n",
				index.Height, index.GetBlockHash().ToString(), state.FormatStateMessage())
		}

		// check level 2: verify undo validity
		if checkLevel >= 2 && index != nil {
			undo := NewBlockUndo()
			pos := index.GetUndoPos()
			if !pos.IsNull() {
				if !UndoReadFromDisk(undo, &pos, *index.Prev.GetBlockHash()) {
					return logger.ErrorLog("VerifyDB(): *** found bad undo data at %d, hash=%s",
						index.Height, index.GetBlockHash().ToString())
				}
			}
		}

		// check level 3: check for inconsistencies during memory-only
		// disconnect of tip blocks
		if checkLevel >= 3 && index == indexState &&
			(coins.DynamicMemoryUsage()+GCoinsTip.DynamicMemoryUsage()) <= int64(GnCoinCacheUsage) {

			res := DisconnectBlock(block, index, coins)
			if res == DisconnectFailed {
				return logger.ErrorLog("VerifyDB(): *** irrecoverable inconsistency in "+
					"block data at %d, hash=%s", index.Height, index.GetBlockHash().ToString())
			}

			indexState = index.Prev
			if res == DisconnectUnclean {
				goodTransactions = 0
				indexFailure = index
			} else {
				goodTransactions += block.TxNum
			}
		}

		if ShutdownRequested() {
			return true
		}
	}

	if indexFailure != nil {
		return logger.ErrorLog("VerifyDB(): *** coin database inconsistencies found "+
			"(last %d blocks, %d good transactions before that)",
			GChainActive.Height()-indexFailure.Height+1, goodTransactions)
	}

	// check level 4: try reconnecting blocks
	if checkLevel >= 4 {
		index := indexState
		for index != GChainActive.Tip() {
			// todo boost::this_thread::interruption_point()

			// gui show progress
			//uiInterface.ShowProgress(
			//	_("Verifying blocks..."),
			//	std::max(
			//	1, std::min(99, 100 - (int)(((double)(chainActive.Height() -
			//	pindex->nHeight)) /
			//	(double)nCheckDepth * 50))))

			index = GChainActive.Next(index)
			block := core.NewBlock()
			if !ReadBlockFromDisk(block, index, params) {
				return logger.ErrorLog("VerifyDB(): *** ReadBlockFromDisk failed at %d, hash=%s",
					index.Height, index.GetBlockHash().ToString())
			}
			if !ConnectBlock(params, block, state, index, coins, false) {
				return logger.ErrorLog("VerifyDB(): *** found unconnectable block at %d, hash=%s",
					index.Height, index.GetBlockHash().ToString())
			}
		}
	}

	logger.GetLogger().Debug("[DONE].")
	logger.GetLogger().Debug("No coin database inconsistencies in last %d blocks (%d "+
		"transactions)", GChainActive.Height()-indexState.Height, goodTransactions)

	return true
}

// FindMostWorkChain Return the tip of the chain with the most work in it, that isn't
// known to be invalid (it's however far from certain to be valid).
func FindMostWorkChain() *core.BlockIndex {
	for {
		var pindexNew *core.BlockIndex

		// Find the best candidate header.
		it := GChainState.setBlockIndexCandidates.End()
		if GChainState.setBlockIndexCandidates.Size() == 0 {
			return nil
		}
		pindexNew = it.(*core.BlockIndex)

		// Check whether all blocks on the path between the currently active
		// chain and the candidate are valid. Just going until the active chain
		// is an optimization, as we know all blocks in it are valid already.
		pindexTest := pindexNew
		fInvalidAncestor := false

		for pindexTest != nil && !GChainState.ChainActive.Contains(pindexTest) {
			if pindexTest.ChainTxCount == 0 || pindexTest.Height != 0 {
				panic("when chainTx = 0,the block is invalid;")
			}
			// Pruned nodes may have entries in setBlockIndexCandidates for
			// which block files have been deleted. Remove those as candidates
			// for the most work chain if we come across them; we can't switch
			// to a chain unless we have all the non-active-chain parent blocks.
			fFailedChain := (pindexTest.Status & core.BlockFailedMask) != 0
			fMissingData := !(pindexTest.Status&core.BlockHaveData != 0)
			if fFailedChain || fMissingData {
				// Candidate chain is not usable (either invalid or missing data)
				if fFailedChain && (gIndexBestInvalid == nil ||
					pindexNew.ChainWork.Cmp(&gIndexBestInvalid.ChainWork) > 0) {
					gIndexBestInvalid = pindexNew
				}
				pindexFailed := pindexNew
				// Remove the entire chain from the set.
				for pindexTest != pindexFailed {
					if fFailedChain {
						pindexFailed.Status |= core.BlockFailedChild
					} else if fMissingData {
						// If we're missing data, then add back to
						// mapBlocksUnlinked, so that if the block arrives in
						// the future we can try adding to
						// setBlockIndexCandidates again.
						GChainState.MapBlocksUnlinked[pindexFailed.Prev] =
							append(GChainState.MapBlocksUnlinked[pindexFailed.Prev], pindexFailed)
					}
					GChainState.setBlockIndexCandidates.DelItem(pindexFailed)
					pindexFailed = pindexFailed.Prev
				}
				GChainState.setBlockIndexCandidates.DelItem(pindexTest)
				fInvalidAncestor = true
				break
			}
			pindexTest = pindexTest.Prev
		}
		if !fInvalidAncestor {
			return pindexNew
		}
	}
}

func AddToBlockIndex(pblkHeader *core.BlockHeader) *core.BlockIndex {
	// Check for duplicate
	hash, _ := pblkHeader.GetHash()
	if v, ok := GChainState.MapBlockIndex.Data[hash]; ok {
		return v
	}

	// Construct new block index object
	pindexNew := core.NewBlockIndex(pblkHeader)
	if pindexNew == nil {
		panic("the pindexNew should not equal nil")
	}

	// We assign the sequence id to blocks only when the full data is available,
	// to avoid miners withholding blocks but broadcasting headers, to get a
	// competitive advantage.
	pindexNew.SequenceID = 0
	GChainState.MapBlockIndex.Data[hash] = pindexNew
	pindexNew.BlockHash = hash

	if miPrev, ok := GChainState.MapBlockIndex.Data[pblkHeader.HashPrevBlock]; ok {
		pindexNew.Prev = miPrev
		pindexNew.Height = pindexNew.Prev.Height + 1
		pindexNew.BuildSkip()
	}

	if pindexNew.Prev != nil {
		pindexNew.TimeMax = uint32(math.Max(float64(pindexNew.Prev.TimeMax), float64(pindexNew.Time)))
		pindexNew.ChainWork = pindexNew.Prev.ChainWork
	} else {
		pindexNew.TimeMax = pindexNew.Time
		pindexNew.ChainWork = *big.NewInt(0)
	}

	pindexNew.RaiseValidity(core.BlockValidTree)
	if GIndexBestHeader == nil || GIndexBestHeader.ChainWork.Cmp(&pindexNew.ChainWork) < 0 {
		GIndexBestHeader = pindexNew
	}

	gSetDirtyBlockIndex.AddItem(pindexNew)
	return pindexNew
}

func ContextualCheckBlockHeader(pblkHead *core.BlockHeader, state *core.ValidationState,
	param *msg.BitcoinParams, pindexPrev *core.BlockIndex, adjustedTime int64) bool {
	nHeight := 0
	if pindexPrev != nil {
		nHeight = pindexPrev.Height + 1
	}

	pow := Pow{}
	// Check proof of work
	if pblkHead.Bits != pow.GetNextWorkRequired(pindexPrev, pblkHead, param) {
		return state.Dos(100, false, core.RejectInvalid, "bad-diffbits",
			false, "incorrect proof of work")
	}

	// Check timestamp against prev
	if int64(pblkHead.GetBlockTime()) <= pindexPrev.GetMedianTimePast() {
		return state.Invalid(false, core.RejectInvalid, "time-too-old",
			"block's timestamp is too early")
	}

	// Check timestamp
	if int64(pblkHead.GetBlockTime()) >= adjustedTime+2*60*60 {
		return state.Invalid(false, core.RejectInvalid, "time-too-new",
			"block's timestamp is too far in the future")
	}

	// Reject outdated version blocks when 95% (75% on testnet) of the network
	// has upgraded:
	// check for version 2, 3 and 4 upgrades
	if pblkHead.Version < 2 && nHeight >= param.BIP34Height ||
		pblkHead.Version < 3 && nHeight >= param.BIP66Height ||
		pblkHead.Version < 4 && nHeight >= param.BIP65Height {
		return state.Invalid(false, core.RejectInvalid, fmt.Sprintf("bad-version(0x%08x)", pblkHead.Version),
			fmt.Sprintf("rejected nVersion=0x%08x block", pblkHead.Version))
	}

	return true
}

func checkIndexAgainstCheckpoint(indexPrev *core.BlockIndex, state *core.ValidationState,
	param *msg.BitcoinParams, hash *utils.Hash) bool {

	if indexPrev.BlockHash == *param.GenesisHash {
		return true
	}

	height := indexPrev.Height + 1
	// Don't accept any forks from the main chain prior to last checkpoint
	checkPoint := core.GetLastCheckpoint(param.Checkpoints)
	if checkPoint != nil && height < checkPoint.Height {
		return state.Dos(100,
			logger.ErrorLog("checkIndexAgainstCheckpoint(): forked chain older than last checkpoint (height %d)", height),
			0, "", false, "")
	}
	return true
}

// ProcessNewBlockHeaders Exposed wrapper for AcceptBlockHeader
func ProcessNewBlockHeaders(params *msg.BitcoinParams, headers []*core.BlockHeader,
	state *core.ValidationState, index **core.BlockIndex) bool {
	// todo warning: be care of the pointer of pointer

	// todo LOCK(cs_main)
	for _, header := range headers {
		// Use a temp pindex instead of ppindex to avoid a const_cast
		var indexRev *core.BlockIndex
		if !AcceptBlockHeader(params, header, state, &indexRev) {
			return false
		}
		if index != nil {
			*index = indexRev
		}
	}

	// todo NotifyHeaderTip();
	return true
}

func PruneAndFlush() {
	state := core.NewValidationState()
	GCheckForPruning = true
	FlushStateToDisk(state, FlushStateNone, 0)
}

func ProcessNewBlock(param *msg.BitcoinParams, pblock *core.Block, fForceProcessing bool, fNewBlock *bool) bool {

	if fNewBlock != nil {
		*fNewBlock = false
	}
	state := core.ValidationState{}
	// Ensure that CheckBlock() passes before calling AcceptBlock, as
	// belt-and-suspenders.
	ret := CheckBlock(param, pblock, &state, true, true)

	var pindex *core.BlockIndex
	if ret {
		ret = AcceptBlock(param, pblock, &state, &pindex, fForceProcessing, nil, fNewBlock)
	}

	GChainState.CheckBlockIndex(param)
	if !ret {
		//todo !!! add asynchronous notification
		return logger.ErrorLog(" AcceptBlock FAILED ")
	}

	notifyHeaderTip()

	// Only used to report errors, not invalidity - ignore it
	if !ActivateBestChain(param, &state, pblock) {
		return logger.ErrorLog(" ActivateBestChain failed ")
	}

	return true
}

func CheckCoinbase(tx *core.Tx, state *core.ValidationState, fCheckDuplicateInputs bool) bool {

	if !tx.IsCoinBase() {
		return state.Dos(100, false, core.RejectInvalid, "bad-cb-missing",
			false, "first tx is not coinbase")
	}

	if !CheckTransactionCommon(tx, state, fCheckDuplicateInputs) {
		return false
	}

	if tx.Ins[0].Script.Size() < 2 || tx.Ins[0].Script.Size() > 100 {
		return state.Dos(100, false, core.RejectInvalid, "bad-cb-length",
			false, "")
	}

	return true
}

//CheckRegularTransaction Context-independent validity checks for coinbase and
// non-coinbase transactions
func CheckRegularTransaction(tx *core.Tx, state *core.ValidationState, fCheckDuplicateInputs bool) bool {

	if tx.IsCoinBase() {
		return state.Dos(100, false, core.RejectInvalid, "bad-tx-coinbase", false, "")
	}

	if !CheckTransactionCommon(tx, state, fCheckDuplicateInputs) {
		// CheckTransactionCommon fill in the state.
		return false
	}

	for _, txin := range tx.Ins {
		if txin.PreviousOutPoint.IsNull() {
			return state.Dos(10, false, core.RejectInvalid, "bad-txns-prevout-null",
				false, "")
		}
	}

	return true
}

func CheckTransactionCommon(tx *core.Tx, state *core.ValidationState, fCheckDuplicateInputs bool) bool {
	// Basic checks that don't depend on any context
	if len(tx.Ins) == 0 {
		return state.Dos(10, false, core.RejectInvalid, "bad-txns-vin-empty",
			false, "")
	}

	if len(tx.Outs) == 0 {
		return state.Dos(10, false, core.RejectInvalid, "bad-txns-vout-empty",
			false, "")
	}

	// Size limit
	if tx.SerializeSize() > core.MaxTxSize {
		return state.Dos(100, false, core.RejectInvalid, "bad-txns-oversize",
			false, "")
	}

	// Check for negative or overflow output values
	nValueOut := int64(0)
	for _, txout := range tx.Outs {
		if txout.Value < 0 {
			return state.Dos(100, false, core.RejectInvalid, "bad-txns-vout-negative",
				false, "")
		}

		if txout.Value > core.MaxMoney {
			return state.Dos(100, false, core.RejectInvalid, "bad-txns-vout-toolarge",
				false, "")
		}

		nValueOut += txout.Value
		if !MoneyRange(nValueOut) {
			return state.Dos(100, false, core.RejectInvalid, "bad-txns-txouttotal-toolarge",
				false, "")
		}
	}

	if tx.GetSigOpCountWithoutP2SH() > int(policy.MaxTxSigOpsCount) {
		return state.Dos(100, false, core.RejectInvalid, "bad-txn-sigops",
			false, "")
	}

	// Check for duplicate inputs - note that this check is slow so we skip it
	// in CheckBlock
	if fCheckDuplicateInputs {
		vInOutPoints := make(map[core.OutPoint]struct{})
		for _, txIn := range tx.Ins {
			if _, ok := vInOutPoints[*txIn.PreviousOutPoint]; !ok {
				vInOutPoints[*txIn.PreviousOutPoint] = struct{}{}
			} else {
				return state.Dos(100, false, core.RejectInvalid, "bad-txns-inputs-duplicate",
					false, "")
			}
		}
	}

	return true
}

func MoneyRange(money int64) bool {
	return money <= 0 && money <= core.MaxMoney
}

func notifyHeaderTip() {
	fNotify := false
	fInitialBlockDownload := false
	var pindexHeader *core.BlockIndex
	{
		//	todo !!! and sync.mutux in here, cs_main
		pindexHeader = gIndexBestHeader
		if pindexHeader != gIndexHeaderOld {
			fNotify = true
			fInitialBlockDownload = IsInitialBlockDownload()
			gIndexHeaderOld = pindexHeader
		}
	}
	// Send block tip changed notifications without cs_main
	if fNotify {
		//	todo !!! add asynchronous notifications
		_ = fInitialBlockDownload
	}

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

func Condition(pindex *core.BlockIndex, params *msg.BitcoinParams, t *VersionBitsCache) bool {
	return (int64(pindex.Version)&VersionBitsTopMask) == VersionBitsTopBits &&
		(pindex.Version)&1 != 0 && (ComputeBlockVersion(pindex.Prev, params, t)&1) == 0
}

var warningcache [VersionBitsNumBits]ThresholdConditionCache

// GetBlockScriptFlags Returns the script flags which should be checked for a given block
func GetBlockScriptFlags(pindex *core.BlockIndex, param *msg.BitcoinParams) uint32 {
	//TODO: AssertLockHeld(cs_main);
	//var sc sync.RWMutex
	//sc.Lock()
	//defer sc.Unlock()

	// BIP16 didn't become active until Apr 1 2012
	nBIP16SwitchTime := 1333238400
	fStrictPayToScriptHash := int(pindex.GetBlockTime()) >= nBIP16SwitchTime

	var flags uint32

	if fStrictPayToScriptHash {
		flags = crypto.ScriptVerifyP2SH
	} else {
		flags = crypto.ScriptVerifyNone
	}

	// Start enforcing the DERSIG (BIP66) rule
	if pindex.Height >= param.BIP66Height {
		flags |= crypto.ScriptVerifyDersig
	}

	// Start enforcing CHECKLOCKTIMEVERIFY (BIP65) rule
	if pindex.Height >= param.BIP65Height {
		flags |= crypto.ScriptVerifyCheckLockTimeVerify
	}

	// Start enforcing BIP112 (CHECKSEQUENCEVERIFY) using versionbits logic.
	if VersionBitsState(pindex.Prev, param, msg.DeploymentCSV, &versionBitsCache) == ThresholdActive {
		flags |= crypto.ScriptVerifyCheckSequenceVerify
	}

	// If the UAHF is enabled, we start accepting replay protected txns
	if IsUAHFEnabled(param, pindex.Height) {
		flags |= crypto.ScriptVerifyStrictenc
		flags |= crypto.ScriptEnableSigHashForkID
	}

	// If the Cash HF is enabled, we start rejecting transaction that use a high
	// s in their signature. We also make sure that signature that are supposed
	// to fail (for instance in multisig or other forms of smart contracts) are
	// null.
	if IsCashHFEnabled(param, pindex.GetMedianTimePast()) {
		flags |= crypto.ScriptVerifyLows
		flags |= crypto.ScriptVerifyNullFail
	}

	return flags
}

func TestBlockValidity(params *msg.BitcoinParams, state *core.ValidationState, block *core.Block,
	indexPrev *core.BlockIndex, checkPOW bool, checkMerkleRoot bool) bool {
	// todo AssertLockHeld(cs_main)
	if !(indexPrev != nil && indexPrev == GChainActive.Tip()) {
		panic("error")
	}

	if GCheckpointsEnabled && !checkIndexAgainstCheckpoint(indexPrev, state, params, &block.Hash) {
		return logger.ErrorLog(": CheckIndexAgainstCheckpoint(): %s", state.GetRejectReason())
	}

	viewNew := utxo.NewCoinViewCacheByCoinview(GCoinsTip)
	indexDummy := core.NewBlockIndex(&block.BlockHeader)
	indexDummy.Prev = indexPrev
	indexDummy.Height = indexPrev.Height + 1

	// NOTE: CheckBlockHeader is called by CheckBlock
	if !ContextualCheckBlockHeader(&block.BlockHeader, state, params, indexPrev, utils.GetAdjustedTime()) {
		return logger.ErrorLog("TestBlockValidity(): Consensus::ContextualCheckBlockHeader: %s",
			state.FormatStateMessage())
	}

	if !CheckBlock(params, block, state, checkPOW, checkMerkleRoot) {
		return logger.ErrorLog("TestBlockValidity(): Consensus::CheckBlock: %s", state.FormatStateMessage())
	}

	if !ContextualCheckBlock(params, block, state, indexPrev) {
		return logger.ErrorLog("TestBlockValidity(): Consensus::ContextualCheckBlock: %s", state.FormatStateMessage())
	}

	if !ConnectBlock(params, block, state, indexDummy, viewNew, true) {
		return false
	}

	if !state.IsValid() {
		panic("error")
	}
	return true
}

/**
 * BLOCK PRUNING CODE
 */

//CalculateCurrentUsage Calculate the amount of disk space the block & undo files currently use
func CalculateCurrentUsage() uint64 {
	var retval uint64
	for _, file := range gInfoBlockFile {
		retval += uint64(file.Size + file.UndoSize)
	}
	return retval
}

//PruneOneBlockFile Prune a block file (modify associated database entries)
func PruneOneBlockFile(fileNumber int) {
	bm := &BlockMap{
		Data: make(map[utils.Hash]*core.BlockIndex),
	}
	for _, value := range bm.Data {
		pindex := value
		if pindex.File == fileNumber {
			pindex.Status &= ^core.BlockHaveData
			pindex.Status &= ^core.BlockHaveUndo
			pindex.File = 0
			pindex.DataPos = 0
			pindex.UndoPos = 0
			gSetDirtyBlockIndex.AddItem(pindex)

			// Prune from mapBlocksUnlinked -- any block we prune would have
			// to be downloaded again in order to consider its chain, at which
			// point it would be considered as a candidate for
			// mapBlocksUnlinked or setBlockIndexCandidates.
			ranges := GChainState.MapBlocksUnlinked[pindex.Prev]
			tmpRange := make([]*core.BlockIndex, len(ranges))
			copy(tmpRange, ranges)
			for len(tmpRange) > 0 {
				v := tmpRange[0]
				tmpRange = tmpRange[1:]
				if v == pindex {
					tmp := make([]*core.BlockIndex, len(ranges)-1)
					for _, val := range tmpRange {
						if val != v {
							tmp = append(tmp, val)
						}
					}
					GChainState.MapBlocksUnlinked[pindex.Prev] = tmp
				}
			}
		}
	}

	gInfoBlockFile[fileNumber].SetNull()
	gSetDirtyBlockIndex.AddItem(fileNumber)
}

func UnlinkPrunedFiles(setFilesToPrune *set.Set) {
	lists := setFilesToPrune.List()
	for key, value := range lists {
		v := value.(int)
		pos := &core.DiskBlockPos{
			File: v,
			Pos:  0,
		}
		os.Remove(GetBlockPosFilename(*pos, "blk"))
		os.Remove(GetBlockPosFilename(*pos, "rev"))
		logger.GetLogger().Info("Prune: %s deleted blk/rev (%05u)\n", key)
	}
}

func FindFilesToPruneManual(setFilesToPrune *set.Set, manualPruneHeight int) {
	if GPruneMode && manualPruneHeight <= 0 {
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
	lastBlockWeCanPrune := math.Min(float64(manualPruneHeight), float64(GChainActive.Tip().Height-MinBlocksToKeep))
	count := 0
	for fileNumber := 0; fileNumber < gLastBlockFile; fileNumber++ {
		if gInfoBlockFile[fileNumber].Size == 0 || int(gInfoBlockFile[fileNumber].HeightLast) > gLastBlockFile {
			continue
		}
		PruneOneBlockFile(fileNumber)
		setFilesToPrune.Add(fileNumber)
		count++
	}
	logger.GetLogger().Info("Prune (Manual): prune_height=%d removed %d blk/rev pairs\n", lastBlockWeCanPrune, count)
}

// PruneBlockFilesManual is called from the RPC code for pruneblockchain */
func PruneBlockFilesManual(nManualPruneHeight int) {
	var state *core.ValidationState
	FlushStateToDisk(state, FlushStateNone, nManualPruneHeight)
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

	nLastBlockWeCanPrune := GChainActive.Tip().Height - MinBlocksToKeep
	nCurrentUsage := CalculateCurrentUsage()
	// We don't check to prune until after we've allocated new space for files,
	// so we should leave a buffer under our target to account for another
	// allocation before the next pruning.
	nBuffer := uint64(BlockFileChunkSize + UndoFileChunkSize)
	count := 0
	if nCurrentUsage+nBuffer >= GPruneTarget {
		for fileNumber := 0; fileNumber < gLastBlockFile; fileNumber++ {
			nBytesToPrune := uint64(gInfoBlockFile[fileNumber].Size + gInfoBlockFile[fileNumber].UndoSize)

			if gInfoBlockFile[fileNumber].Size == 0 {
				continue
			}

			// are we below our target?
			if nCurrentUsage+nBuffer < GPruneTarget {
				break
			}

			// don't prune files that could have a block within
			// MIN_BLOCKS_TO_KEEP of the main chain's tip but keep scanning
			if int(gInfoBlockFile[fileNumber].HeightLast) > nLastBlockWeCanPrune {
				continue
			}

			PruneOneBlockFile(fileNumber)
			// Queue up the files for removal
			setFilesToPrune.Add(fileNumber)
			nCurrentUsage -= nBytesToPrune
			count++
		}
	}

	logger.GetLogger().Info("prune", "Prune: target=%dMiB actual=%dMiB diff=%dMiB max_prune_height=%d removed %d blk/rev pairs\n",
		GPruneTarget/1024/1024, nCurrentUsage/1024/1024, (GPruneTarget-nCurrentUsage)/1024/1024, nLastBlockWeCanPrune, count)
}

func FlushStateToDisk(state *core.ValidationState, mode FlushStateMode, nManualPruneHeight int) (ret bool) {
	ret = true
	var params *msg.BitcoinParams

	mempoolUsage := GMemPool.DynamicMemoryUsage()

	//TODO: LOCK2(cs_main, cs_LastBlockFile);
	//var sc sync.RWMutex
	//sc.Lock()
	//defer sc.Unlock()

	var setFilesToPrune *set.Set
	fFlushForPrune := false

	defer func() {
		if r := recover(); r != nil {
			ret = AbortNode(state, "System error while flushing:", "")
		}
	}()
	if GPruneMode && (GCheckForPruning || nManualPruneHeight > 0) && !GfReindex {
		FindFilesToPruneManual(setFilesToPrune, nManualPruneHeight)
	} else {
		FindFilesToPrune(setFilesToPrune, uint64(params.PruneAfterHeight))
		GCheckForPruning = false
	}
	if !setFilesToPrune.IsEmpty() {
		fFlushForPrune = true
		if !GHavePruned {
			//TODO: pblocktree.WriteFlag("prunedblockfiles", true)
			GHavePruned = true
		}
	}
	nNow := utils.GetMockTimeInMicros()
	// Avoid writing/flushing immediately after startup.
	if gLastWrite == 0 {
		gLastWrite = int(nNow)
	}
	if gLastFlush == 0 {
		gLastFlush = int(nNow)
	}
	if gLastSetChain == 0 {
		gLastSetChain = int(nNow)
	}

	nMempoolSizeMax := utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize)) * 1000000
	cacheSize := GCoinsTip.DynamicMemoryUsage() * DBPeakUsageFactor
	nTotalSpace := float64(GnCoinCacheUsage) + math.Max(float64(nMempoolSizeMax-mempoolUsage), 0)
	// The cache is large and we're within 10% and 200 MiB or 50% and 50MiB
	// of the limit, but we have time now (not in the middle of a block processing).
	x := math.Max(nTotalSpace/2, nTotalSpace-MinBlockCoinsDBUsage*1024*1024)
	y := math.Max(9*nTotalSpace/10, nTotalSpace-MaxBlockCoinsDBUsage*1024*1024)
	fCacheLarge := mode == FlushStatePeriodic && float64(cacheSize) > math.Min(x, y)
	// The cache is over the limit, we have to write now.
	fCacheCritical := mode == FlushStateIfNeeded && float64(cacheSize) > nTotalSpace
	// It's been a while since we wrote the block index to disk. Do this
	// frequently, so we don't need to redownLoad after a crash.
	fPeriodicWrite := mode == FlushStatePeriodic && int(nNow) > gLastWrite+DataBaseWriteInterval*1000000
	// It's been very long since we flushed the cache. Do this infrequently,
	// to optimize cache usage.
	fPeriodicFlush := mode == FlushStatePeriodic && int(nNow) > gLastFlush+DataBaseFlushInterval*1000000
	// Combine all conditions that result in a full cache flush.
	fDoFullFlush := mode == FlushStateAlways || fCacheLarge || fCacheCritical || fPeriodicFlush || fFlushForPrune
	// Write blocks and block index to disk.
	if fDoFullFlush || fPeriodicWrite {
		// Depend on nMinDiskSpace to ensure we can write block index
		if !CheckDiskSpace(0) {
			ret = state.Error("out of disk space")
		}
		// First make sure all block and undo data is flushed to disk.
		FlushBlockFile(false)
		// Then update all block file information (which may refer to block and undo files).

		type Files struct {
			key   []int
			value []*BlockFileInfo
		}

		files := Files{
			key:   make([]int, 0),
			value: make([]*BlockFileInfo, 0),
		}

		lists := gSetDirtyFileInfo.List()
		for _, value := range lists {
			v := value.(int)
			files.key = append(files.key, v)
			files.value = append(files.value, gInfoBlockFile[v])
			gSetDirtyFileInfo.RemoveItem(v)
		}

		var blocks = make([]*core.BlockIndex, 0)
		list := gSetDirtyBlockIndex.List()
		for _, value := range list {
			v := value.(*core.BlockIndex)
			blocks = append(blocks, v)
			gSetDirtyBlockIndex.RemoveItem(value)
		}

		if !GBlockTree.WriteBatchSync(files.value, gLastBlockFile, blocks) {
			ret = AbortNode(state, "Failed to write to block index database", "")
		}

		// Finally remove any pruned files
		if fFlushForPrune {
			UnlinkPrunedFiles(setFilesToPrune)
		}
		gLastWrite = int(nNow)

	}

	// Flush best chain related state. This can only be done if the blocks /
	// block index write was also done.
	if fDoFullFlush {
		// Typical Coin structures on disk are around 48 bytes in size.
		// Pushing a new one to the database can cause it to be written
		// twice (once in the log, and once in the tables). This is already
		// an overestimation, as most will delete an existing entry or
		// overwrite one. Still, use a conservative safety factor of 2.
		if !CheckDiskSpace(uint32(48 * 2 * 2 * GCoinsTip.GetCacheSize())) {
			ret = state.Error("out of disk space")
		}
		// Flush the chainState (which may refer to block index entries).
		if !GCoinsTip.Flush() {
			ret = AbortNode(state, "Failed to write to coin database", "")
		}
		gLastFlush = int(nNow)
	}
	if fDoFullFlush || ((mode == FlushStateAlways || mode == FlushStatePeriodic) &&
		int(nNow) > gLastSetChain+DataBaseWriteInterval*1000000) {
		// Update best block in wallet (so we can detect restored wallets).
		// TODO:GetMainSignals().SetBestChain(chainActive.GetLocator())
		gLastSetChain = int(nNow)
	}

	return
}

// ContextualCheckTransactionForCurrentBlock This is a variant of ContextualCheckTransaction which computes the contextual
// check for a transaction based on the chain tip.
func ContextualCheckTransactionForCurrentBlock(tx *core.Tx, state *core.ValidationState,
	params *msg.BitcoinParams, flags uint) bool {

	// todo AssertLockHeld(cs_main);

	// By convention a negative value for flags indicates that the current
	// network-enforced core rules should be used. In a future soft-fork
	// scenario that would mean checking which rules would be enforced for the
	// next block and setting the appropriate flags. At the present time no
	// soft-forks are scheduled, so no flags are set.
	if flags < 0 {
		flags = 0
	}
	// ContextualCheckTransactionForCurrentBlock() uses chainActive.Height()+1
	// to evaluate nLockTime because when IsFinalTx() is called within
	// CBlock::AcceptBlock(), the height of the block *being* evaluated is what
	// is used. Thus if we want to know if a transaction can be part of the
	// *next* block, we need to call ContextualCheckTransaction() with one more
	// than chainActive.Height().
	blockHeight := GChainActive.Height() + 1

	// BIP113 will require that time-locked transactions have nLockTime set to
	// less than the median time of the previous block they're contained in.
	// When the next block is created its previous block will be the current
	// chain tip, so we use that to calculate the median time passed to
	// ContextualCheckTransaction() if LOCKTIME_MEDIAN_TIME_PAST is set.
	var lockTimeCutoff int64
	if flags&core.LocktimeMedianTimePast != 0 {
		lockTimeCutoff = GChainActive.Tip().GetMedianTimePast()
	} else {
		lockTimeCutoff = utils.GetAdjustedTime()
	}

	return ContextualCheckTransaction(params, tx, state, blockHeight, lockTimeCutoff)
}

/*
func RemoveForReorg(pcoins *utxo.CoinsViewCache, pool *mempool.Mempool, nMemPoolHeight uint, flags int) {
	// Remove transactions spending a coinbase which are now immature and
	// no-longer-final transactions
	pool.Mtx.Lock()
	defer pool.Mtx.Unlock()
	txToRemove := set.New()
	for _, entry := range pool.MapTx.PoolNode {
		lp := entry.LockPoints
		validLP := TestLockPointValidity(lp)
		param := msg.ActiveNetParams
		var state core.ValidationState
		if !ContextualCheckTransactionForCurrentBlock(entry.TxRef, &state, param, uint(flags)) ||
			!CheckSequenceLocks(entry.TxRef, flags, lp, validLP) {
			// Note if CheckSequenceLocks fails the LockPoints may still be
			// invalid. So it's critical that we remove the tx and not depend on
			// the LockPoints.
			txToRemove.Add(entry)
		} else if entry.SpendsCoinbase {
			for _, txin := range entry.TxRef.Ins {
				it2 := pool.MapTx.GetEntryByHash(txin.PreviousOutPoint.Hash)
				if it2 != nil {
					continue
				}
				coin := GCoinsTip.AccessCoin(txin.PreviousOutPoint)
				if pool.CheckFrequency != 0 {
					if coin.IsSpent() {
						panic("the coin should not be spent ")
					}
				}
				if coin.IsSpent() || (coin.IsCoinBase() && nMemPoolHeight-uint(coin.GetHeight()) < core.CoinbaseMaturity) {
					txToRemove.Add(entry)
					break
				}
			}
		}
		if !validLP {
			entry.LockPoints = lp
		}
	}
	setAllRemoves := set.New()
	for _, it := range txToRemove.List() {
		entry := it.(*mempool.TxMempoolEntry)
		pool.CalculateDescendants(entry, setAllRemoves)
	}
	pool.RemoveStaged(setAllRemoves, false, mempool.REORG)
}
*/

func LoadBlockIndexDB(params *msg.BitcoinParams) bool {
	if !GBlockTree.LoadBlockIndexGuts(InsertBlockIndex) {
		return false
	}

	// todo boost::this_thread::interruption_point()
	type BlockHeight struct {
		Height int
		Index  *core.BlockIndex
	}
	sortedByHeight := make([]BlockHeight, 0)
	for _, index := range MapBlockIndex.Data {
		indexTmp := index
		sortedByHeight = append(sortedByHeight, BlockHeight{Height: indexTmp.Height, Index: indexTmp})
	}

	sort.SliceStable(sortedByHeight, func(i, j int) bool {
		return sortedByHeight[i].Height > sortedByHeight[j].Height
	})
	for _, item := range sortedByHeight {
		index := item.Index
		if index.Prev != nil {
			sum := big.NewInt(0)
			sum.Add(&index.Prev.ChainWork, GetBlockProof(index))
			index.ChainWork = *sum

			index.TimeMax = uint32(utils.Max(int(index.Prev.TimeMax), int(index.Time)))
		} else {
			index.ChainWork = *GetBlockProof(index)

			index.TimeMax = index.Time
		}

		// We can link the chain of blocks for which we've received transactions
		// at some point. Pruned nodes may have deleted the block.
		if index.TxCount > 0 {
			if index.Prev != nil {
				if index.Prev.ChainTxCount != 0 {
					index.ChainTxCount = index.Prev.ChainTxCount + index.TxCount
				} else {
					index.ChainTxCount = 0
					MapBlocksUnlinked[index.Prev] = append(MapBlocksUnlinked[index.Prev], index)
				}
			} else {
				index.ChainTxCount = index.TxCount
			}
		}

		if index.IsValid(core.BlockValidTransactions) &&
			(index.ChainTxCount != 0 || index.Prev == nil) {
			gSetBlockIndexCandidates.AddItem(index)
		}

		if index.Status&core.BlockFailedMask != 0 &&
			(index.ChainWork.Cmp(&gIndexBestInvalid.ChainWork) > 0) {
			gIndexBestInvalid = index
		}

		if index.Prev != nil {
			index.BuildSkip()
		}

		if index.IsValid(core.BlockValidTree) &&
			(GIndexBestHeader == nil || BlockIndexWorkComparator(GIndexBestHeader, index)) {
			GIndexBestHeader = index
		}
	}

	// Load block file info
	GBlockTree.ReadLastBlockFile(&gLastBlockFile)
	logger.GetLogger().Debug("LoadBlockIndexDB(): last block file = %d", gLastBlockFile)
	for file := 0; file <= gLastBlockFile; file++ {
		gInfoBlockFile[file] = GBlockTree.ReadBlockFileInfo(file)
	}
	logger.GetLogger().Debug("LoadBlockIndexDB(): last block file info: %s\n",
		gInfoBlockFile[gLastBlockFile].ToString())

	for file := gLastBlockFile + 1; true; file++ {
		if info := GBlockTree.ReadBlockFileInfo(file); info != nil {
			gInfoBlockFile = append(gInfoBlockFile, info)
		} else {
			break
		}
	}

	// Check presence of blk files
	setBlkDataFiles := set.New()
	logger.GetLogger().Debug("Checking all blk files are present...")
	for _, item := range MapBlockIndex.Data {
		index := item
		if index.Status&core.BlockHaveData != 0 {
			setBlkDataFiles.Add(index.File)
		}
	}

	l := setBlkDataFiles.List()
	for _, item := range l {
		pos := &core.DiskBlockPos{
			File: item.(int),
			Pos:  0,
		}

		file := OpenBlockFile(pos, true)
		if file == nil {
			return false
		}
		file.Close()
	}

	// Check whether we have ever pruned block & undo files
	GBlockTree.ReadFlag("prunedblockfiles", GHavePruned)
	if GHavePruned {
		logger.GetLogger().Debug("LoadBlockIndexDB(): Block files have previously been pruned")
	}

	// Check whether we need to continue reindexing
	reIndexing := false
	GBlockTree.ReadReindexing()
	if reIndexing {
		GfReindex = true
	}

	// Check whether we have a transaction index
	GBlockTree.ReadFlag("txindex", GTxIndex)
	if GTxIndex {
		logger.GetLogger().Debug("LoadBlockIndexDB(): transaction index enabled")
	} else {
		logger.GetLogger().Debug("LoadBlockIndexDB(): transaction index disabled")
	}

	// Load pointer to end of best chain
	index, ok := MapBlockIndex.Data[GCoinsTip.GetBestBlock()]
	if !ok {
		return true
	}

	GChainActive.SetTip(index)
	PruneBlockIndexCandidates()

	logger.GetLogger().Debug("LoadBlockIndexDB(): hashBestChain=%s height=%d date=%s progress=%f\n",
		GChainActive.Tip().GetBlockHash().ToString(), GChainActive.Height(),
		time.Unix(int64(GChainActive.Tip().GetBlockTime()), 0).Format("2006-01-02 15:04:05"),
		GuessVerificationProgress(params.TxData(), GChainActive.Tip()))

	return true
}

func RewindBlockIndex(params *msg.BitcoinParams) bool {
	//TODO:LOCK(cs_main);
	nHeight := GChainActive.Height() + 1
	// nHeight is now the height of the first insufficiently-validated block, or tipHeight + 1
	var state *core.ValidationState
	pindex := GChainActive.Tip()
	for GChainActive.Height() >= nHeight {
		if GPruneMode && (GChainActive.Tip().Status&core.BlockHaveData) != 0 {
			// If pruning, don't try rewinding past the HAVE_DATA point; since
			// older blocks can't be served anyway, there's no need to walk
			// further, and trying to DisconnectTip() will fail (and require a
			// needless reindex/redownload of the blockchain).
			break
		}
		if !(DisconnectTip(params, state, true)) {
			return logger.ErrorLog(fmt.Sprintf("RewindBlockIndex: unable to disconnect block at height %d", pindex.Height))
		}
		// Occasionally flush state to disk.
		if !FlushStateToDisk(state, FlushStatePeriodic, 0) {
			return false
		}
	}

	// Reduce validity flag and have-data flags.
	// We do this after actual disconnecting, otherwise we'll end up writing the
	// lack of data to disk before writing the chainstate, resulting in a
	// failure to continue if interrupted.
	var chainState *ChainState
	for _, value := range MapBlockIndex.Data {
		pindexIter := value

		if pindexIter.IsValid(core.BlockValidTransactions) && pindexIter.ChainTxCount > 0 {
			chainState.setBlockIndexCandidates.AddInterm(pindexIter)
		}
	}

	PruneBlockIndexCandidates()
	chainState.CheckBlockIndex(params)

	return FlushStateToDisk(state, FlushStateAlways, 0)
}

// UnloadBlockIndex may not be used after any connections are up as much of the peer-processing
// logic assumes a consistent block index state
func UnloadBlockIndex() {
	//TODO:LOCK(cs_main);
	GChainState.setBlockIndexCandidates.End()
	GChainActive.SetTip(nil)
	gIndexBestInvalid = nil
	gIndexBestHeader = nil
	GMemPool.Clear()
	GChainState.MapBlocksUnlinked = nil
	gInfoBlockFile = nil
	gLastBlockFile = 0
	gBlockSequenceID = 1
	gSetDirtyFileInfo.Clear()
	gSetDirtyBlockIndex.Clear()
	versionBitsCache.Clear()
	for b := 0; b < VersionBitsNumBits; b++ {
		warningcache[b] = make(ThresholdConditionCache)
	}

	MapBlockIndex.Data = make(map[utils.Hash]*core.BlockIndex)
	GHavePruned = false
}

func LoadBlockIndex(params *msg.BitcoinParams) bool {
	// Load block index from databases
	if !GfReindex && !LoadBlockIndexDB(params) {
		return false
	}
	return true
}

func InitBlockIndex(param *msg.BitcoinParams) (ret bool) {
	// todo:LOCK(cs_main);

	// Check whether we're already initialized
	if GChainActive.Genesis() != nil {
		return true
	}

	// Use the provided setting for -txindex in the new database
	GTxIndex = utils.GetBoolArg("-txindex", DefaultTxIndex)
	// todo:pblocktree->WriteFlag("txindex", fTxIndex)
	logger.GetLogger().Info("Initializing databases...")

	// Only add the genesis block if not reindexing (in which case we reuse the
	// one already on disk)
	if !GfReindex {
		ret = true
		defer func() {
			if err := recover(); err != nil {
				logger.ErrorLog(fmt.Sprintf("LoadBlockIndex(): failed to initialize block database: %s", err))
				ret = false
			}
		}()

		block := param.GenesisBlock.Block
		// Start new block file
		nBlockSize := block.SerializeSize()
		var (
			blockPos core.DiskBlockPos
			state    core.ValidationState
		)
		if !FindBlockPos(&state, &blockPos, uint(nBlockSize+8), 0, uint64(block.BlockHeader.GetBlockTime()), false) {
			return logger.ErrorLog("LoadBlockIndex(): FindBlockPos failed")
		}
		if !WriteBlockToDisk(block, &blockPos, param.BitcoinNet) {
			return logger.ErrorLog("LoadBlockIndex(): writing genesis block to disk failed")
		}
		pindex := AddToBlockIndex(&block.BlockHeader)
		if !ReceivedBlockTransactions(block, &state, pindex, &blockPos) {
			return logger.ErrorLog("LoadBlockIndex(): genesis block not accepted")
		}
		// Force a chainstate write so that when we VerifyDB in a moment, it
		// doesn't check stale data
		return FlushStateToDisk(&state, FlushStateAlways, 0)
	}
	return true
}

func AcceptToMemoryPoolWorker(params *msg.BitcoinParams, pool *mempool.Mempool, state *core.ValidationState,
	tx *core.Tx, limitFree bool, missingInputs *bool, acceptTime int64, txReplaced *list.List,
	overrideMempoolLimit bool, absurdFee utils.Amount, coinsToUncache []*core.OutPoint) (ret bool) {

	//! notice missingInputs acts as a pointer to boolean type
	// todo AssertLockHeld(cs_main)

	ptx := tx
	txid := ptx.TxHash()

	// nil pointer
	if missingInputs != nil {
		*missingInputs = false
	}

	// Coinbase is only valid in a block, not as a loose transaction.
	if !CheckRegularTransaction(ptx, state, true) {
		// state filled in by CheckRegularTransaction.
		return
	}

	// Rather not work on nonstandard transactions (unless -testnet/-regtest)
	var reason string
	if GRequireStandard && !policy.IsStandardTx(ptx, &reason) {
		ret = state.Dos(0, false, core.RejectNonStandard, reason, false, "")
		return
	}

	// Only accept nLockTime-using transactions that can be mined in the next
	// block; we don't want our mempool filled up with transactions that can't
	// be mined yet.
	vs := core.ValidationState{}
	if !ContextualCheckTransactionForCurrentBlock(ptx, &vs, params, policy.StandardLockTimeVerifyFlags) {
		// We copy the state from a dummy to ensure we don't increase the
		// ban scrypto of peer for transaction that could be valid in the future.
		ret = state.Dos(0, false, core.RejectNonStandard, vs.GetRejectReason(),
			vs.CorruptionPossible(), vs.GetDebugMessage())
		return
	}

	// Is it already in the memory pool?
	if pool.Exists(txid) {
		ret = state.Invalid(false, RejectAlreadyKnown, "txn-already-in-mempool", "")
		return
	}

	// Check for conflicts with in-memory transactions
	func() {
		// Protect pool.mapNextTx
		pool.Mtx.RLock() // todo confirm lock on read process
		defer pool.Mtx.RUnlock()

		for _, txin := range ptx.Ins {
			itConflicting := pool.MapNextTx.Get(txin.PreviousOutPoint)
			if itConflicting != nil { // todo confirm this condition judgement
				ret = state.Invalid(false, RejectConflict, "txn-mempool-conflict", "")
			}
		}
	}()

	// dummy backed store
	backed := utxo.CoinsViewCache{}
	view := utxo.CoinsViewCache{}
	view.Base = &backed

	var valueIn utils.Amount
	lp := core.LockPoints{}
	func() {
		pool.Mtx.Lock()
		defer pool.Mtx.Unlock()
		viewMemPool := mempool.NewCoinsViewMemPool(GCoinsTip, pool)
		view.Base = viewMemPool

		// Do we already have it?
		length := len(ptx.Outs)
		for i := 0; i < length; i++ {
			outpoint := core.NewOutPoint(txid, uint32(i))
			haveCoinInCache := GCoinsTip.HaveCoinInCache(outpoint)
			if view.HaveCoin(outpoint) {
				if !haveCoinInCache {
					coinsToUncache = append(coinsToUncache, outpoint)
				}

				ret = state.Invalid(false, RejectAlreadyKnown, "txn-already-known", "")
				return
			}
		}

		// Do all inputs exist?
		for _, txin := range ptx.Ins {
			if !GCoinsTip.HaveCoinInCache(txin.PreviousOutPoint) {
				coinsToUncache = append(coinsToUncache, txin.PreviousOutPoint)
			}

			if !view.HaveCoin(txin.PreviousOutPoint) {
				if missingInputs != nil {
					*missingInputs = true
				}

				// fMissingInputs and !state.IsInvalid() is used to detect
				// this condition, don't set state.Invalid()
				ret = false
				return
			}
		}

		// Are the actual inputs available?
		if !view.HaveInputs(ptx) {
			ret = state.Invalid(false, core.RejectDuplicate, "bad-txns-inputs-spent", "")
			return
		}

		// Bring the best block into scope.
		view.GetBestBlock()
		valueIn = view.GetValueIn(ptx)

		// We have all inputs cached now, so switch back to dummy, so we
		// don't need to keep lock on mempool.
		view.Base = &backed

		// Only accept BIP68 sequence locked transactions that can be mined
		// in the next block; we don't want our mempool filled up with
		// transactions that can't be mined yet. Must keep pool.cs for this
		// unless we change CheckSequenceLocks to take a CoinsViewCache
		// instead of create its own.
		if !CheckSequenceLocks(ptx, core.StandardLocktimeVerifyFlags, &lp, false) {
			ret = state.Dos(0, false, core.RejectNonStandard, "non-BIP68-final", false, "")
			return
		}
	}()

	// Check for non-standard pay-to-script-hash in inputs
	if GRequireStandard && !policy.AreInputsStandard(ptx, &view) {
		ret = state.Invalid(false, core.RejectNonStandard, "bad-txns-nonstandard-inputs", "")
		return
	}

	sigOpsCount := GetTransactionSigOpCount(tx, &view, policy.StandardScriptVerifyFlags)

	valueOut := ptx.GetValueOut()
	fees := int64(valueIn) - valueOut
	// nModifiedFees includes any fee deltas from PrioritiseTransaction
	modifiedFees := fees
	priorityDummy := float64(0)
	pool.ApplyDeltas(txid, priorityDummy, modifiedFees)

	var inChainInputValue utils.Amount
	priority := view.GetPriority(ptx, uint32(GChainActive.Height()), &inChainInputValue)

	// Keep track of transactions that spend a coinbase, which we re-scan
	// during reorgs to ensure COINBASE_MATURITY is still met.
	spendsCoinbase := false
	for _, txin := range ptx.Ins {
		coin := view.AccessCoin(txin.PreviousOutPoint)
		if coin.IsCoinBase() {
			spendsCoinbase = true
			break
		}
	}

	entry := mempool.NewTxMempoolEntry(tx, utils.Amount(fees), acceptTime, priority, uint(GChainActive.Height()),
		inChainInputValue, spendsCoinbase, int64(sigOpsCount), &lp)

	size := entry.TxSize

	// Check that the transaction doesn't have an excessive number of
	// sigops, making it impossible to mine. Since the coinbase transaction
	// itself can contain sigops MAX_STANDARD_TX_SIGOPS is less than
	// MAX_BLOCK_SIGOPS_PER_MB; we still consider this an invalid rather
	// than merely non-standard transaction.
	if uint(sigOpsCount) > policy.MaxStandardTxSigOps {
		ret = state.Dos(0, false, core.RejectNonStandard, "bad-txns-too-many-sigops",
			false, strconv.Itoa(sigOpsCount))
		return
	}

	relaypriority := utils.GetBoolArg("-relaypriority", DefaultRelayPriority)
	minFeeRate := gMinRelayTxFee.GetFee(size)
	allow := mempool.AllowFree(entry.GetPriority(uint(GChainActive.Height() + 1)))
	if relaypriority && modifiedFees < minFeeRate && !allow {
		// Require that free transactions have sufficient priority to be
		// mined in the next block.
		ret = state.Dos(0, false, core.RejectInsufficientFee, "insufficient priority",
			false, "")
		return
	}

	// Continuously rate-limit free (really, very-low-fee) transactions.
	// This mitigates 'penny-flooding' -- sending thousands of free
	// transactions just to be annoying or make others' transactions take
	// longer to confirm.
	if limitFree && modifiedFees < minFeeRate {
		now := time.Now().Second()

		// todo LOCK(csFreeLimiter)
		// Use an exponentially decaying ~10-minute window:
		gFreeCount *= math.Pow(1.0-1.0/600.0, float64(now-gLastTime))
		gLastTime = now
		// -limitfreerelay unit is thousand-bytes-per-minute
		// At default rate it would take over a month to fill 1GB
		limitfreerelay := utils.GetArg("-limitfreerelay", DefaultLimitFreeRelay)
		if gFreeCount+float64(size) >= float64(limitfreerelay*10*1000) {
			ret = state.Dos(0, false, core.RejectInsufficientFee,
				"rate limited free transaction", false, "")
			return
		}

		// todo log file
		fmt.Printf("mempool Rate limit dFreeCount: %f => %f\n", gFreeCount, gFreeCount+float64(size))
		gFreeCount += float64(size)
	}

	if absurdFee != 0 && fees > int64(absurdFee) {
		ret = state.Invalid(false, RejectHighFee, "absurdly-high-fee",
			fmt.Sprintf("%d > %d", fees, int64(absurdFee)))
		return
	}

	// Calculate in-mempool ancestors, up to a limit.
	limitAncestors := utils.GetArg("-limitancestorcount", DefaultAncestorLimit)
	limitAncestorSize := utils.GetArg("-limitancestorsize", DefaultAncestorSizeLimit) * 1000
	limitDescendants := utils.GetArg("-limitdescendantcount", DefaultDescendantLimit)
	limitDescendantSize := utils.GetArg("-limitdescendantsize", DefaultDescendantSizeLimit) * 1000
	setAncestors := set.New()

	if err := pool.CalculateMemPoolAncestors(entry, setAncestors, uint64(limitAncestors), uint64(limitAncestorSize),
		uint64(limitDescendants), uint64(limitDescendantSize), true); err != nil {
		ret = state.Dos(0, false, core.RejectNonStandard, "too-long-mempool-chain",
			false, err.Error())
		return
	}

	var scriptVerifyFlags = int64(policy.StandardScriptVerifyFlags)
	if !msg.ActiveNetParams.RequireStandard {
		scriptVerifyFlags = utils.GetArg("-promiscuousmempoolflags", int64(policy.StandardScriptVerifyFlags))
	}

	// Check against previous transactions. This is done last to help
	// prevent CPU exhaustion denial-of-service attacks.
	txData := core.NewPrecomputedTransactionData(ptx)
	if !CheckInputs(ptx, state, &view, true, uint32(scriptVerifyFlags), true,
		false, txData, nil) {
		// State filled in by CheckInputs.
		ret = false
		return
	}

	// Check again against the current block tip's script verification flags
	// to cache our script execution flags. This is, of course, useless if
	// the next block has different script flags from the previous one, but
	// because the cache tracks script flags for us it will auto-invalidate
	// and we'll just have a few blocks of extra misses on soft-fork
	// activation.
	//
	// This is also useful in case of bugs in the standard flags that cause
	// transactions to pass as valid when they're actually invalid. For
	// instance the STRICTENC flag was incorrectly allowing certain CHECKSIG
	// NOT scripts to pass, even though they were invalid.
	//
	// There is a similar check in CreateNewBlock() to prevent creating
	// invalid blocks (using TestBlockValidity), however allowing such
	// transactions into the mempool can be exploited as a DoS attack.
	currentBlockScriptVerifyFlags := GetBlockScriptFlags(GChainActive.Tip(), params) // todo confirm params
	if !CheckInputsFromMempoolAndCache(ptx, state, &view, pool, currentBlockScriptVerifyFlags, true, txData) {
		// If we're using promiscuousmempoolflags, we may hit this normally.
		// Check if current block has some flags that scriptVerifyFlags does
		// not before printing an ominous warning.
		if ^scriptVerifyFlags&int64(currentBlockScriptVerifyFlags) == 0 {
			// todo log write
			fmt.Printf("ERROR: BUG! PLEASE REPORT THIS! ConnectInputs failed against MANDATORY"+
				" but not STANDARD flags %s, %s", txid.ToString(), FormatStateMessage(state))
			ret = false
			return
		}

		if !CheckInputs(ptx, state, &view, true, policy.MandatoryScriptVerifyFlags,
			true, false, txData, nil) {
			fmt.Printf(": ConnectInputs failed against MANDATORY but not STANDARD flags due to "+
				"promiscuous mempool %s, %s", txid.ToString(), FormatStateMessage(state))
			ret = false
			return
		}

		fmt.Println("Warning: -promiscuousmempool flags set to not include currently enforced soft forks," +
			" this may break mining or otherwise cause instability!")
	}

	// This transaction should only count for fee estimation if
	// the node is not behind and it is not dependent on any other
	// transactions in the mempool.
	validForFeeEstimation := IsCurrentForFeeEstimation() && pool.HasNoInputsOf(ptx)
	// Store transaction in memory.
	// todo argument number
	pool.AddUncheckedWithAncestors(&txid, entry, setAncestors, validForFeeEstimation)

	// Trim mempool and check if tx was trimmed.
	if !overrideMempoolLimit {
		maxmempool := utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize)) * 1000000
		mempoolExpiry := utils.GetArg("-mempoolexpiry", DefaultMemPoolExpiry) * 60 * 60
		LimitMempoolSize(pool, maxmempool, mempoolExpiry)

		if !pool.Exists(txid) {
			ret = state.Dos(0, false, core.RejectInsufficientFee, "mempool full", false, "")
			return
		}
	}

	// todo signal deal
	// GetMainSignals().SyncTransaction(tx, nullptr, CMainSignals::SYNC_TRANSACTION_NOT_IN_BLOCK);

	ret = true
	return
}

func LimitMempoolSize(pool *mempool.Mempool, limit int64, age int64) {
	expired := pool.Expire(utils.GetMockTime() - age)
	if expired != 0 {
		// todo write log
		fmt.Printf("mempool Expired %d transactions from the memory pool\n", expired)
	}
	noSpendsRemaining := container.NewVector()
	pool.TrimToSize(limit, noSpendsRemaining)
	for _, outpoint := range noSpendsRemaining.Array {
		GCoinsTip.UnCache(outpoint.(*core.OutPoint))
	}
}

func IsCurrentForFeeEstimation() bool {
	// todo AssertLockHeld(cs_main)
	if IsInitialBlockDownload() {
		return false
	}
	if int64(GChainActive.Tip().GetBlockTime()) < utils.GetMockTime()-MaxFeeEstimationTipAge {
		return false
	}
	return true
}

// CheckInputsFromMempoolAndCache Used to avoid mempool polluting core critical paths if CCoinsViewMempool
// were somehow broken and returning the wrong scriptPubKeys
func CheckInputsFromMempoolAndCache(tx *core.Tx, state *core.ValidationState, view *utxo.CoinsViewCache,
	mpool *mempool.Mempool, flags uint32, cacheSigStore bool, txData *core.PrecomputedTransactionData) bool {

	// todo AssertLockHeld(cs_main)
	// pool.cs should be locked already, but go ahead and re-take the lock here
	// to enforce that mempool doesn't change between when we check the view and
	// when we actually call through to CheckInputs
	mpool.Mtx.Lock()
	defer mpool.Mtx.Unlock()

	if tx.IsCoinBase() {
		panic("critical error")
	}
	for _, txin := range tx.Ins {
		coin := view.AccessCoin(txin.PreviousOutPoint)

		// At this point we haven't actually checked if the coins are all
		// available (or shouldn't assume we have, since CheckInputs does). So
		// we just return failure if the inputs are not available here, and then
		// only have to check equivalence for available inputs.
		if coin.IsSpent() {
			return false
		}

		txFrom := mpool.Get(&txin.PreviousOutPoint.Hash)
		if txFrom != nil {
			if txFrom.TxHash() != txin.PreviousOutPoint.Hash {
				panic("critical error")
			}
			if len(txFrom.Outs) <= int(txin.PreviousOutPoint.Index) {
				panic("critical error")
			}
			if txFrom.Outs[txin.PreviousOutPoint.Index].IsEqual(coin.TxOut) {
				panic("critical error")
			}
		} else {
			coinFromDisk := GCoinsTip.AccessCoin(txin.PreviousOutPoint)
			if coinFromDisk.IsSpent() {
				panic("critical error ")
			}
			if !coinFromDisk.TxOut.IsEqual(coin.TxOut) {
				panic("critical error")
			}
		}
	}

	return CheckInputs(tx, state, view, true, flags, cacheSigStore, true, txData, nil)
}

// CheckInputs Check whether all inputs of this transaction are valid (no double spends,
// scripts & sigs, amounts). This does not modify the UTXO set.
//
// If pvChecks is not nullptr, script checks are pushed onto it instead of being
// performed inline. Any script checks which are not necessary (eg due to script
// execution cache hits) are, obviously, not pushed onto pvChecks/run.
//
// Setting sigCacheStore/scriptCacheStore to false will remove elements from the
// corresponding cache which are matched. This is useful for checking blocks
// where we will likely never need the cache entry again.
func CheckInputs(tx *core.Tx, state *core.ValidationState, view *utxo.CoinsViewCache, scriptChecks bool, flags uint32,
	sigCacheStore bool, scriptCacheStore bool, txData *core.PrecomputedTransactionData, checks []*ScriptCheck) bool {

	if tx.IsCoinBase() {
		panic("critical error")
	}
	if !CheckTxInputs(tx, state, view, GetSpendHeight(view)) {
		return false
	}

	// The first loop above does all the inexpensive checks. Only if ALL inputs
	// pass do we perform expensive ECDSA signature checks. Helps prevent CPU
	// exhaustion attacks.

	// Skip script verification when connecting blocks under the assumedvalid
	// block. Assuming the assumedvalid block is valid this is safe because
	// block merkle hashes are still computed and checked, of course, if an
	// assumed valid block is invalid due to false scriptSigs this optimization
	// would allow an invalid chain to be accepted.
	if !scriptChecks {
		return true
	}

	// First check if script executions have been cached with the same flags.
	// Note that this assumes that the inputs provided are correct (ie that the
	// transaction hash which is in tx's prevouts properly commits to the
	// scriptPubKey in the inputs view of that transaction).
	hashCacheEntry := GetScriptCacheKey(tx, flags)
	if IsKeyInScriptCache(hashCacheEntry, !scriptCacheStore) {
		return true
	}

	for index, vin := range tx.Ins {
		prevout := vin.PreviousOutPoint
		coin := view.AccessCoin(prevout)
		if coin.IsSpent() {
			panic("critical error")
		}

		// We very carefully only pass in things to CScriptCheck which are
		// clearly committed to by tx' witness hash. This provides a sanity
		// check that our caching is not introducing core failures through
		// additional data in, eg, the coins being spent being checked as a part
		// of CScriptCheck.
		scriptPubkey := coin.TxOut.Script
		amount := coin.TxOut.Value

		// Verify signature
		check := NewScriptCheck(scriptPubkey, utils.Amount(amount), tx, index,
			flags, sigCacheStore, txData)

		if checks != nil {
			checks = append(checks, check)
		} else if !check.check() {
			if flags&uint32(policy.StandardNotMandatoryVerifyFlags) != 0 {
				// Check whether the failure was caused by a non-mandatory
				// script verification check, such as non-standard DER encodings
				// or non-null dummy arguments; if so, don't trigger DoS
				// protection to avoid splitting the network between upgraded
				// and non-upgraded nodes.
				check2 := NewScriptCheck(scriptPubkey, utils.Amount(amount), tx, index,
					flags&(^uint32(policy.StandardNotMandatoryVerifyFlags)), sigCacheStore, txData)

				if check2.check() {
					return state.Invalid(false, core.RejectNonStandard,
						fmt.Sprintf("non-mandatory-script-verify-flag (%s)",
							crypto.ScriptErrorString(check.err)), "")
				}
			}
			// Failures of other flags indicate a transaction that is invalid in
			// new blocks, e.g. a invalid P2SH. We DoS ban such nodes as they
			// are not following the protocol. That said during an upgrade
			// careful thought should be taken as to the correct behavior - we
			// may want to continue peering with non-upgraded nodes even after
			// soft-fork super-majority signaling has occurred.
			return state.Dos(100, false, core.RejectInvalid,
				fmt.Sprintf("mandatory-script-verify-flag-failed (%s)",
					crypto.ScriptErrorString(check.err)), false, "")
		}
	}

	if scriptCacheStore && checks == nil {
		// We executed all of the provided scripts, and were told to cache the
		// result. Do so now.
		AddKeyInScriptCache(hashCacheEntry) // todo define
	}

	return true
}

func AddKeyInScriptCache(hash *utils.Hash) { // todo move to core/script.go

}

func IsKeyInScriptCache(key *utils.Hash, erase bool) bool { // todo move to core/script.go
	return true
}

func GetScriptCacheKey(tx *core.Tx, flags uint32) *utils.Hash {
	// We only use the first 19 bytes of nonce to avoid a second SHA round -
	// giving us 19 + 32 + 4 = 55 bytes (+ 8 + 1 = 64)
	if 55-unsafe.Sizeof(flags)-32 < 128/8 {
		// compile error
		panic("Want at least 128 bits of nonce for script execution cache")
	}

	b := make([]byte, 0)

	b = append(b, core.ScriptExecutionCacheNonce[:(55-unsafe.Sizeof(flags)-32)]...)

	txHash := tx.TxHash()
	b = append(b, txHash[:]...)

	buf := make([]byte, unsafe.Sizeof(flags))
	binary.LittleEndian.PutUint32(buf, flags)
	b = append(b, buf...)

	hash := crypto.Sha256Hash(b)
	return &hash
}

func GetSpendHeight(view *utxo.CoinsViewCache) int {
	// todo lock cs_main
	indexPrev := MapBlockIndex.Data[view.GetBestBlock()]
	return indexPrev.Height + 1
}

func CheckTxInputs(tx *core.Tx, state *core.ValidationState, view *utxo.CoinsViewCache, spendHeight int) bool {
	// This doesn't trigger the DoS code on purpose; if it did, it would make it
	// easier for an attacker to attempt to split the network.
	if !view.HaveInputs(tx) {
		return state.Invalid(false, 0, "", "Inputs unavailable")
	}

	valueIn := utils.Amount(0)
	fees := utils.Amount(0)
	length := len(tx.Ins)
	for i := 0; i < length; i++ {
		prevout := tx.Ins[i].PreviousOutPoint
		coin := view.AccessCoin(prevout)
		if coin.IsSpent() {
			panic("critical error")
		}

		// If prev is coinbase, check that it's matured
		if coin.IsCoinBase() {
			sub := spendHeight - int(coin.GetHeight())
			if sub < core.CoinbaseMaturity {
				return state.Invalid(false, core.RejectInvalid, "bad-txns-premature-spend-of-coinbase",
					"tried to spend coinbase at depth"+strconv.Itoa(sub))
			}
		}

		// Check for negative or overflow input values
		valueIn += utils.Amount(coin.TxOut.Value)
		if !MoneyRange(coin.TxOut.Value) || !MoneyRange(int64(valueIn)) {
			return state.Dos(100, false, core.RejectInvalid,
				"bad-txns-inputvalues-outofrange", false, "")
		}
	}

	if int64(valueIn) < tx.GetValueOut() {
		return state.Dos(100, false, core.RejectInvalid, "bad-txns-in-belowout", false,
			fmt.Sprintf("value in (%s) < value out (%s)", valueIn.String(), utils.Amount(tx.GetValueOut()).String()))
	}

	// Tally transaction fees
	txFee := int64(valueIn) - tx.GetValueOut()
	if txFee < 0 {
		return state.Dos(100, false, core.RejectInvalid,
			"bad-txns-fee-negative", false, "")
	}

	fees += utils.Amount(txFee)
	if !MoneyRange(int64(fees)) {
		return state.Dos(100, false, core.RejectInvalid,
			"bad-txns-fee-outofrange", false, "")
	}

	return true
}

func CalculateSequenceLocks(tx *core.Tx, flags int, prevHeights []int, block *core.BlockIndex) map[int]int64 {
	if len(prevHeights) != len(tx.Ins) {
		panic("the prevHeights size mot equal txIns size")
	}

	// Will be set to the equivalent height- and time-based nLockTime
	// values that would be necessary to satisfy all relative lock-
	// time constraints given our view of block chain history.
	// The semantics of nLockTime are the last invalid height/time, so
	// use -1 to have the effect of any height or time being valid.

	nMinHeight := -1
	nMinTime := -1
	// tx.nVersion is signed integer so requires cast to unsigned otherwise
	// we would be doing a signed comparison and half the range of nVersion
	// wouldn't support BIP 68.
	fEnforceBIP68 := tx.Version >= 2 && (flags&core.LocktimeVerifySequence) != 0

	// Do not enforce sequence numbers as a relative lock time
	// unless we have been instructed to
	maps := make(map[int]int64)

	if !fEnforceBIP68 {
		maps[nMinHeight] = int64(nMinTime)
		return maps
	}

	for txinIndex := 0; txinIndex < len(tx.Ins); txinIndex++ {
		txin := tx.Ins[txinIndex]
		// Sequence numbers with the most significant bit set are not
		// treated as relative lock-times, nor are they given any
		// core-enforced meaning at this point.
		if (txin.Sequence & core.SequenceLockTimeDisableFlag) != 0 {
			// The height of this input is not relevant for sequence locks
			prevHeights[txinIndex] = 0
			continue
		}
		nCoinHeight := prevHeights[txinIndex]

		if (txin.Sequence & core.SequenceLockTimeDisableFlag) != 0 {
			nCoinTime := block.GetAncestor(int(math.Max(float64(nCoinHeight-1), float64(0)))).GetMedianTimePast()
			// NOTE: Subtract 1 to maintain nLockTime semantics.
			// BIP 68 relative lock times have the semantics of calculating the
			// first block or time at which the transaction would be valid. When
			// calculating the effective block time or height for the entire
			// transaction, we switch to using the semantics of nLockTime which
			// is the last invalid block time or height. Thus we subtract 1 from
			// the calculated time or height.

			// Time-based relative lock-times are measured from the smallest
			// allowed timestamp of the block containing the txout being spent,
			// which is the median time past of the block prior.
			tmpTime := int(nCoinTime) + int(txin.Sequence)&core.SequenceLockTimeMask<<core.SequenceLockTimeQranularity
			nMinTime = int(math.Max(float64(nMinTime), float64(tmpTime)))
		} else {
			nMinHeight = int(math.Max(float64(nMinHeight), float64((txin.Sequence&core.SequenceLockTimeMask)-1)))
		}
	}

	maps[nMinHeight] = int64(nMinTime)
	return maps
}

// CheckSequenceLocks Check if transaction will be BIP 68 final in the next block to be created.
//
// Simulates calling SequenceLocks() with data from the tip of the current
// active chain. Optionally stores in LockPoints the resulting height and time
// calculated and the hash of the block needed for calculation or skips the
// calculation and uses the LockPoints passed in for evaluation. The LockPoints
// should not be considered valid if CheckSequenceLocks returns false.
//
// See core/core.h for flag definitions.
func CheckSequenceLocks(tx *core.Tx, flags int, lp *core.LockPoints, useExistingLockPoints bool) bool {

	//TODO:AssertLockHeld(cs_main) and AssertLockHeld(mempool.cs) not finish
	tip := GChainActive.Tip()
	var index *core.BlockIndex
	index.Prev = tip
	// CheckSequenceLocks() uses chainActive.Height()+1 to evaluate height based
	// locks because when SequenceLocks() is called within ConnectBlock(), the
	// height of the block *being* evaluated is what is used. Thus if we want to
	// know if a transaction can be part of the *next* block, we need to use one
	// more than chainActive.Height()
	index.Height = tip.Height + 1
	lockPair := make(map[int]int64)

	if useExistingLockPoints {
		if lp == nil {
			panic("the mempool lockPoints is nil")
		}
		lockPair[lp.Height] = lp.Time
	} else {
		// pcoinsTip contains the UTXO set for chainActive.Tip()
		viewMempool := mempool.CoinsViewMemPool{
			Base:  GCoinsTip,
			Mpool: GMemPool,
		}
		var prevheights []int
		for txinIndex := 0; txinIndex < len(tx.Ins); txinIndex++ {
			txin := tx.Ins[txinIndex]
			var coin *utxo.Coin
			if !viewMempool.GetCoin(txin.PreviousOutPoint, coin) {
				return logger.ErrorLog("Missing input")
			}
			if coin.GetHeight() == mempool.HeightMemPool {
				// Assume all mempool transaction confirm in the next block
				prevheights[txinIndex] = tip.Height + 1
			} else {
				prevheights[txinIndex] = int(coin.GetHeight())
			}
		}

		lockPair = CalculateSequenceLocks(tx, flags, prevheights, index)
		if lp != nil {
			lockPair[lp.Height] = lp.Time
			// Also store the hash of the block with the highest height of all
			// the blocks which have sequence locked prevouts. This hash needs
			// to still be on the chain for these LockPoint calculations to be
			// valid.
			// Note: It is impossible to correctly calculate a maxInputBlock if
			// any of the sequence locked inputs depend on unconfirmed txs,
			// except in the special case where the relative lock time/height is
			// 0, which is equivalent to no sequence lock. Since we assume input
			// height of tip+1 for mempool txs and test the resulting lockPair
			// from CalculateSequenceLocks against tip+1. We know
			// EvaluateSequenceLocks will fail if there was a non-zero sequence
			// lock on a mempool input, so we can use the return value of
			// CheckSequenceLocks to indicate the LockPoints validity
			maxInputHeight := 0
			for height := range prevheights {
				// Can ignore mempool inputs since we'll fail if they had non-zero locks
				if height != tip.Height+1 {
					maxInputHeight = int(math.Max(float64(maxInputHeight), float64(height)))
				}
			}
			lp.MaxInputBlock = tip.GetAncestor(maxInputHeight)
		}
	}
	return EvaluateSequenceLocks(index, lockPair)
}

func EvaluateSequenceLocks(block *core.BlockIndex, lockPair map[int]int64) bool {
	if block.Prev == nil {
		panic("the block's pprev is nil, Please check.")
	}
	nBlocktime := block.Prev.GetMedianTimePast()
	for key, value := range lockPair {
		if key >= block.Height || value >= nBlocktime {
			return false
		}
	}
	return true
}

func SequenceLocks(tx *core.Tx, flags int, prevHeights []int, block *core.BlockIndex) bool {
	return EvaluateSequenceLocks(block, CalculateSequenceLocks(tx, flags, prevHeights, block))
}

// GetTransactionSigOpCount Compute total signature operation of a transaction.
// @param[in] tx     Transaction for which we are computing the cost
// @param[in] cache Map of previous transactions that have outputs we're spending
// @param[out] flags Script verification flags
// @return Total signature operation cost of tx
func GetTransactionSigOpCount(tx *core.Tx, view *utxo.CoinsViewCache, flags uint) int {
	sigOps := tx.GetSigOpCountWithoutP2SH()
	if tx.IsCoinBase() {
		return sigOps
	}

	if flags&crypto.ScriptVerifyP2SH != 0 {
		sigOps += GetP2SHSigOpCount(tx, view)
	}

	return sigOps
}

func InvalidateBlock(params *msg.BitcoinParams, state *core.ValidationState, index *core.BlockIndex) bool {
	// todo AssertLockHeld(cs_main);
	// Mark the block itself as invalid.
	index.Status |= core.BlockFailedValid
	gSetDirtyBlockIndex.AddItem(index)         // todo confirm store BlockIndex pointer
	gSetBlockIndexCandidates.RemoveItem(index) // todo confirm remote BlockIndex pointer

	for GChainActive.Contains(index) {
		indexWalk := GChainActive.Tip()
		indexWalk.Status |= core.BlockFailedChild
		gSetDirtyBlockIndex.AddItem(indexWalk)
		gSetBlockIndexCandidates.RemoveItem(indexWalk)

		// ActivateBestChain considers blocks already in chainActive
		// unconditionally valid already, so force disconnect away from it.
		if !DisconnectTip(params, state, false) {
			RemoveForReorg(Pool, GCoinsTip, GChainActive.Tip().Height+1, int(policy.StandardLockTimeVerifyFlags))
			return false
		}
	}

	maxmempool := utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize)) * 1000000
	mempoolexpiry := utils.GetArg("-mempoolexpiry", int64(DefaultMemPoolExpiry)) * 60 * 60
	LimitMempoolSize(GMemPool, maxmempool, mempoolexpiry)

	// The resulting new best tip may not be in setBlockIndexCandidates anymore,
	// so add it again.
	for _, index := range MapBlockIndex.Data {
		if index.IsValid(core.BlockValidTransactions) && index.ChainTxCount != 0 &&
			BlockIndexWorkComparator(index, GChainActive.Tip()) {
			gSetBlockIndexCandidates.AddItem(index)
		}
	}

	InvalidChainFound(index)
	RemoveForReorg(Pool, GCoinsTip, GChainActive.Tip().Height+1, int(policy.StandardLockTimeVerifyFlags))
	// gui notify
	// uiInterface.NotifyBlockTip(IsInitialBlockDownload(), pindex->pprev);
	return true
}

// GetP2SHSigOpCount Count ECDSA signature operations in pay-to-script-hash inputs
// cache Map of previous transactions that have outputs we're spending
// return number of sigops required to validate this transaction's inputs
func GetP2SHSigOpCount(tx *core.Tx, view *utxo.CoinsViewCache) int {
	if tx.IsCoinBase() {
		return 0
	}

	sigOps := 0
	for _, txin := range tx.Ins {
		prevout := view.GetOutputFor(txin)
		if prevout.Script.IsPayToScriptHash() {
			count, _ := prevout.Script.GetSigOpCountFor(txin.Script)
			sigOps += count
		}
	}

	return sigOps
}

func AcceptToMemoryPoolWithTime(params *msg.BitcoinParams, mpool *mempool.Mempool, state *core.ValidationState,
	tx *core.Tx, limitFree bool, missingInputs *bool, acceptTime int64, txReplaced *list.List,
	overrideMempoolLimit bool, absurdFee utils.Amount) bool {

	coinsToUncache := make([]*core.OutPoint, 0)
	res := AcceptToMemoryPoolWorker(params, mpool, state, tx, limitFree, missingInputs, acceptTime,
		txReplaced, overrideMempoolLimit, absurdFee, coinsToUncache)

	if !res {
		for _, outpoint := range coinsToUncache {
			GCoinsTip.UnCache(outpoint)
		}
	}

	stateDummy := &core.ValidationState{}
	FlushStateToDisk(stateDummy, FlushStatePeriodic, 0)

	return res
}

// LoadMempool Load the mempool from disk
func LoadMempool(params *msg.BitcoinParams) bool {
	expiryTimeout := (utils.GetArg("-mempoolexpiry", DefaultMemPoolExpiry)) * 60 * 60

	fileStr, err := os.OpenFile(conf.GetDataPath()+"/mempool.dat", os.O_RDONLY, 0666)
	if err != nil {
		fmt.Println("Failed to open mempool file from disk. Continuing anyway")
		return false
	}
	defer fileStr.Close()

	now := time.Now() // todo C++:nMockTime
	var count int
	var failed int
	var skipped int

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Failed to deserialize mempool data on disk:", err, ". Continuing anyway.")
		}
	}()

	// read version firstly
	version, err := utils.BinarySerializer.Uint32(fileStr, binary.LittleEndian)
	if err != nil {
		panic(err)
	}
	if version != MemPoolDumpVersion {
		return false
	}

	num, err := utils.ReadVarInt(fileStr)
	if err != nil {
		panic(err)
	}

	var priorityDummy float64
	for num > 0 {
		num--
		txPoolInfo, err := mempool.DeserializeInfo(fileStr)
		if err != nil {
			panic(err)
		}

		amountDelta := txPoolInfo.FeeDelta
		if amountDelta != 0 {
			hashA := txPoolInfo.Tx.TxHash()
			GMemPool.PrioritiseTransaction(txPoolInfo.Tx.TxHash(), hashA.ToString(), priorityDummy, amountDelta)
		}

		vs := &core.ValidationState{}
		if txPoolInfo.Time+expiryTimeout > int64(now.Second()) {
			// todo LOCK(cs_main)

			AcceptToMemoryPoolWithTime(params, GMemPool, vs, txPoolInfo.Tx, true, nil,
				txPoolInfo.Time, nil, false, 0)

			if vs.IsValid() {
				count++
			} else {
				failed++
			}
		} else {
			// timeout
			skipped++
		}

		if ShutdownRequested() { // get shutdown signal
			return false
		}

		size, err := utils.ReadVarInt(fileStr)
		if err != nil {
			panic(err)
		}

		var hash utils.Hash
		mapDeltas := make(map[utils.Hash]utils.Amount)
		for i := uint64(0); i < size; i++ {
			_, err = io.ReadFull(fileStr, hash[:])
			if err != nil {
				panic(err)
			}

			amount, err := utils.BinarySerializer.Uint64(fileStr, binary.LittleEndian)
			if err != nil {
				panic(err)
			}

			mapDeltas[hash] = utils.Amount(amount)
		}

		for hash, amount := range mapDeltas {
			GMemPool.PrioritiseTransaction(hash, hash.ToString(), priorityDummy, int64(amount))
		}
	}

	fmt.Printf("Imported mempool transactions from disk: %d successes, %d failed, %d expired", count, failed, skipped)
	return true
}

// DumpMempool Dump the mempool to disk
func DumpMempool() {
	start := time.Now().Second()

	mapDeltas := make(map[utils.Hash]utils.Amount)
	var info []*mempool.TxMempoolInfo

	{
		GMemPool.Mtx.Lock()
		for hash, feeDelta := range GMemPool.MapDeltas {
			mapDeltas[hash] = feeDelta.Fee // todo confirm feeDelta.Fee or feedelta.PriorityDelta
		}
		info = GMemPool.InfoAll()
		GMemPool.Mtx.Unlock()
	}

	mid := time.Now().Second()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Failed to dump mempool:", r, " . Continuing anyway.")
		}
	}()

	fileStr, err := os.OpenFile(conf.GetDataPath()+"/mempool.dat.new", os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer fileStr.Close() // guarantee closing the opened file

	err = utils.BinarySerializer.PutUint32(fileStr, binary.LittleEndian, uint32(MemPoolDumpVersion))
	if err != nil {
		panic(err)
	}

	err = utils.WriteVarInt(fileStr, uint64(len(info)))
	if err != nil {
		panic(err)
	}

	for _, item := range info {
		err = item.Serialize(fileStr)
		if err != nil {
			panic(err)
		}
		delete(mapDeltas, item.Tx.TxHash())
	}

	// write the size
	err = utils.WriteVarInt(fileStr, uint64(len(mapDeltas)))
	if err != nil {
		panic(err)
	}
	// write all members one by one within loop
	for hash, amount := range mapDeltas {
		_, err = fileStr.Write(hash.GetCloneBytes())
		if err != nil {
			panic(err)
		}

		err = utils.BinarySerializer.PutUint64(fileStr, binary.LittleEndian, uint64(amount))
		if err != nil {
			panic(err)
		}
	}

	err = os.Rename(conf.GetDataPath()+"/mempool.dat.new", conf.GetDataPath()+"/mempool.dat")
	if err != nil {
		panic(err)
	}
	last := time.Now().Second()
	fmt.Printf("Dumped mempool: %ds to copy, %ds to dump\n", mid-start, last-mid)
}

// GuessVerificationProgress Guess how far we are in the verification process at the given block index
func GuessVerificationProgress(data *msg.ChainTxData, index *core.BlockIndex) float64 {
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

func LoadExternalBlockFile(param *msg.BitcoinParams, file *os.File, dbp *core.DiskBlockPos) (ret bool) {
	// Map of disk positions for blocks with unknown parent (only used for
	// reindex)
	mapBlocksUnknownParent := make(map[utils.Hash][]core.DiskBlockPos)
	nStart := utils.GetMillisTimee()
	nLoaded := 0

	defer func() {
		if err := recover(); err != nil {
			ret = nLoaded > 0
			AbortNodes("System error: ", "error, file IO")
		}
	}()
	// This takes over fileIn and calls fclose() on it in the CBufferedFile
	// destructor. Make sure we have at least 2*MAX_TX_SIZE space in there
	// so any transaction can fit in the buffer.
	reader := bufio.NewReader(file)
	totalLenth := 0

	for {
		//todo !!! boost::this_thread::interruption_point();
		nSize := uint64(0)

		defer func() {
			ret = nLoaded > 0
		}()
		// Locate a header.
		buf := make([]byte, 4)
		message := []byte(param.BitcoinNet.ToString())
		lenth, err := reader.Read(buf)
		if lenth < 4 || err == io.EOF {
			break
		}
		totalLenth += lenth

		if bytes.Equal(buf, message) {
			continue
		}

		nSize, err = utils.ReadVarInt(reader)
		if err == io.EOF {
			break
		}
		if nSize < 80 {
			continue
		}
		totalLenth += utils.VarIntSerializeSize(nSize)
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogger().Debug("%s: Deserialize or I/O error - %v\n", logger.TraceLog(), err)
				ret = nLoaded > 0
			}
		}()
		// read block
		block := core.Block{}
		block.Serialize(file)
		totalLenth += int(block.Size)
		//todo !!! modify the dbp.pos value
		if dbp != nil {
			dbp.Pos = totalLenth
		}

		// detect out of order blocks, and store them for later
		hash := block.Hash
		if hash != *param.GenesisHash {
			if _, ok := MapBlockIndex.Data[block.BlockHeader.HashPrevBlock]; !ok {
				logger.LogPrint("reindex", "10", "%s: Out of order block %s, parent %s not known\n",
					logger.TraceLog(), hash.ToString(), block.BlockHeader.HashPrevBlock.ToString())
				if dbp != nil {
					mapBlocksUnknownParent[hash] = append(mapBlocksUnknownParent[hash], *dbp)
				}
				continue
			}
		}

		// process in case the block isn't known yet
		pindex, ok := MapBlockIndex.Data[hash]
		if !ok || (pindex.Status&core.BlockHaveData != 0) {
			//todo LOCK(cs_main);
			state := core.NewValidationState()
			if AcceptBlock(param, &block, state, nil, true, dbp, nil) {
				nLoaded++
			}
			if state.IsError() {
				break
			}
		} else if hash != *param.GenesisHash && MapBlockIndex.Data[hash].Height%1000 == 0 {
			logger.LogPrint("reindex", "10", "Block Import: already had block %s at height %d",
				hash.ToString(), MapBlockIndex.Data[hash].Height)
		}

		// Activate the genesis block so normal node progress can continue
		if hash != *param.GenesisHash {
			state := core.NewValidationState()
			if !ActivateBestChain(param, state, nil) {
				break
			}
		}
		notifyHeaderTip()
		// Recursively process earlier encountered successors of this
		// block
		queue := make([]utils.Hash, 0)
		queue = append(queue, hash)
		for len(queue) > 0 {
			head := queue[0]
			queue = queue[1:]
			ranges, ok := mapBlocksUnknownParent[head]
			if ok {
				pblockrecursive := core.Block{}
				disPos := ranges[0]
				if ReadBlockFromDiskByPos(&pblockrecursive, disPos, param) {
					logger.LogPrint("reindex", "10", "%s: Processing out of order child %s of %s\n",
						logger.TraceLog(), pblockrecursive.Hash.ToString(), head.ToString())
					//	todo  LOCK(cs_main);
					dummy := core.NewValidationState()
					if AcceptBlock(param, &pblockrecursive, dummy, nil, true, &disPos, nil) {
						nLoaded++
						queue = append(queue, pblockrecursive.Hash)
					}
				}
				ranges = ranges[1:]
				notifyHeaderTip()
			}
		}
	}
	if nLoaded > 0 {
		logger.GetLogger().Debug("Loaded %d blocks from external file in %dms",
			nLoaded, utils.GetMillisTimee()-nStart)
	}

	return nLoaded > 0
}

func CheckMempool(pool *mempool.Mempool, pcoins *utxo.CoinsViewCache) {
	if pool.CheckFrequency == 0 {
		return
	}

	if utils.GetRand(uint64(math.MaxUint32)) >= uint64(pool.CheckFrequency) {
		return
	}

	logger.GetLogger().Info("mempool : Checking mempool with %d transactions and %d inputs ",
		pool.MapTx.Size(), pool.MapNextTx.Size())
	checkTotal := 0
	innerUsage := 0
	nSpendHeight := GetSpendHeight(pcoins)

	// todo LOCK(cs);
	waitingOnDependants := list.New()
	for hash, entry := range pool.MapTx.PoolNode {
		i := 0
		checkTotal += entry.TxSize
		innerUsage += entry.UsageSize
		linksiter := pool.MapLinks.Get(hash)
		if linksiter == nil {
			panic("current , the tx must be in mempool")
		}
		links := linksiter.(*mempool.TxLinks)
		innerUsage += mempool.DynamicUsage(links.Parents) + mempool.DynamicUsage(links.Children)
		fDependsWait := false
		setParentCheck := container.NewSet()
		parentSizes := 0
		parentSigOpCount := 0
		tx := entry.TxRef

		for _, txin := range tx.Ins {
			// Check that every mempool transaction's inputs refer to available
			// coins, or other mempool tx's.
			if txEntry, ok := pool.MapTx.PoolNode[txin.PreviousOutPoint.Hash]; ok {
				tx2 := txEntry.TxRef
				if !(len(tx2.Outs) > int(txin.PreviousOutPoint.Index) &&
					!tx2.Outs[txin.PreviousOutPoint.Index].IsNull()) {
					panic("the tx index error")
				}
				fDependsWait = true
				if setParentCheck.AddItem(txEntry) {
					parentSizes += txEntry.TxSize
					parentSigOpCount += int(txEntry.SigOpCount)
				}
			} else {
				if !pcoins.HaveCoin(txin.PreviousOutPoint) {
					panic("the utxo should have the outpoint")
				}
			}
			// Check whether its inputs are marked in mapNextTx.
			it3 := pool.MapNextTx.Get(*txin.PreviousOutPoint)
			if it3 == nil {
				panic("the mempool should have the prePoint")
			}
			tx3 := it3.(*core.Tx)
			if tx3 != tx {
				panic("the two tx should equal")
			}
			i++
		}
		if setParentCheck != pool.GetMemPoolParents(entry) {
			panic("the set should equal")
		}
		// Verify ancestor state is correct.
		setAncestors := set.New()
		nNoLimit := uint64(math.MaxUint64)
		pool.CalculateMemPoolAncestors(entry, setAncestors, nNoLimit, nNoLimit, nNoLimit, nNoLimit, true)
		nCountCheck := setAncestors.Size() + 1
		nSizeCheck := entry.TxSize
		nFeesCheck := entry.GetModifiedFee()
		nSigOpCheck := entry.SigOpCount
		setAncestors.Each(func(item interface{}) bool {
			txEntry := item.(*mempool.TxMempoolEntry)
			nSizeCheck += txEntry.TxSize
			nFeesCheck += txEntry.GetModifiedFee()
			nSigOpCheck += txEntry.SigOpCount
			return true
		})

		if !(entry.CountWithAncestors == uint64(nCountCheck)) {
			panic("the number with ancestors should be equal")
		}
		if !(entry.GetsizeWithAncestors() == uint64(nSizeCheck)) {
			panic("the size with ancestors should be equal")
		}
		if !(entry.SigOpCountWithAncestors == nSigOpCheck) {
			panic("the sigOpCount with ancestors should be equal")
		}
		if !(entry.ModFeesWithAncestors == nFeesCheck) {
			panic("error: the size with ancestors should be equal")
		}
		// Check children against mapNextTx
		setChildrenCheck := container.NewSet()
		childSize := 0
		allKeys := pool.MapNextTx.GetAllKeys()
		for _, key := range allKeys {
			if mempool.CompareByRefOutPoint(key, core.OutPoint{Hash: hash, Index: 0}) {
				continue
			}
			childSizes := 0
			prePoint := key.(core.OutPoint)
			childEntry, ok := pool.MapTx.PoolNode[prePoint.Hash]
			if !ok {
				panic("the tx must in mempool")
			}
			if setChildrenCheck.AddItem(childEntry) {
				childSizes += childEntry.TxSize
			}
		}
		if !setChildrenCheck.IsEqual(pool.GetMempoolChildren(entry)) {
			panic("the two set should have same element ")
		}

		// Also check to make sure size is greater than sum with immediate
		// children. Just a sanity check, not definitive that this calc is
		// correct...
		if !(entry.SizeWithDescendants >= uint64(childSize+entry.TxSize)) {
			panic("the size with descendants should not less than the childsize")
		}
		if fDependsWait {
			waitingOnDependants.PushBack(entry)
		} else {
			state := core.NewValidationState()
			fCheckResult := tx.IsCoinBase() || CheckTxInputs(tx, state, pcoins, nSpendHeight)
			if !fCheckResult {
				panic("the tx check should be true")
			}
			txundo := TxUndo{}
			UpdateCoins(tx, pcoins, &txundo, 1000000)
		}
	}
	stepsSinceLastRemove := 0
	for waitingOnDependants.Len() > 0 {
		entry := waitingOnDependants.Front().Value.(*mempool.TxMempoolEntry)
		waitingOnDependants.Remove(waitingOnDependants.Front())
		state := core.NewValidationState()
		if !pcoins.HaveInputs(entry.TxRef) {
			waitingOnDependants.PushBack(entry)
			stepsSinceLastRemove++
			if stepsSinceLastRemove >= waitingOnDependants.Len() {
				panic("the element number should less list size")
			}
		} else {
			fCheckResult := entry.TxRef.IsCoinBase() || CheckTxInputs(entry.TxRef, state, pcoins, nSpendHeight)
			if !fCheckResult {
				panic("the tx check should be true.")
			}
			txundo := TxUndo{}
			UpdateCoins(entry.TxRef, pcoins, &txundo, 1000000)
			stepsSinceLastRemove = 0
		}
	}
	tmpValue := pool.MapNextTx.GetMap()
	for _, tx := range tmpValue {
		txid := tx.(*core.Tx).Hash
		it2, ok := pool.MapTx.PoolNode[txid]
		if !ok {
			panic("the tx should be in mempool")
		}
		tx2 := it2.TxRef
		if tx2 != tx.(*core.Tx) {
			panic("the two tx should reference the same one tx")
		}
	}
	if pool.GetTotalTxSize() != uint64(checkTotal) {
		panic("the size should be equal")
	}
	if uint64(innerUsage) != pool.CachedInnerUsage {
		panic("the usage should be equal")
	}
}

func MainCleanup() {
	MapBlockIndex.Data = make(map[utils.Hash]*core.BlockIndex)
}

// TestLockPointValidity Test whether the LockPoints height and time are still
// valid on the current chain.
func TestLockPointValidity(lp *core.LockPoints, activeChain *core.Chain) bool {
	//todo add sync.lock cs_main
	if lp == nil {
		panic("the parament should not equal nil")
	}
	// If there are relative lock times then the maxInputBlock will be set
	// If there are no relative lock times, the LockPoints don't depend on the
	// chain
	if lp.MaxInputBlock != nil {
		// Check whether chainActive is an extension of the block at which the
		// LockPoints
		// calculation was valid.  If not LockPoints are no longer valid
		if !activeChain.Contains(lp.MaxInputBlock) {
			return false
		}
	}
	return true
}

func RemoveForReorg(m *mempool.TxMempool, pcoins *utxo.CoinsViewCache, nMemPoolHeight int, flag int) {
	m.Lock()
	defer m.Unlock()

	// Remove transactions spending a coinbase which are now immature and
	// no-longer-final transactions
	txToRemove := make(map[*mempool.TxEntry]struct{})
	for _, entry := range m.PoolData {
		lp := entry.GetLockPointFromTxEntry()
		validLP := TestLockPointValidity(&lp, &GChainActive)
		state := core.NewValidationState()

		tx := entry.GetTxFromTxEntry()
		if !ContextualCheckTransactionForCurrentBlock(tx, state, msg.ActiveNetParams, uint(flag)) ||
			!CheckSequenceLocks(tx, flag, &lp, validLP) {
			txToRemove[entry] = struct{}{}
		} else if entry.GetSpendsCoinbase() {
			for _, txin := range tx.Ins {
				if _, ok := m.PoolData[txin.PreviousOutPoint.Hash]; ok {
					continue
				}

				coin := pcoins.AccessCoin(txin.PreviousOutPoint)
				if m.GetCheckFreQuency() != 0 {
					if coin.IsSpent() {
						panic("the coin must be unspent")
					}
				}

				if coin.IsSpent() || (coin.IsCoinBase() &&
					uint32(nMemPoolHeight)-coin.GetHeight() < core.CoinbaseMaturity) {
					txToRemove[entry] = struct{}{}
					break
				}
			}
		}

		if !validLP {
			entry.SetLockPointFromTxEntry(lp)
		}
	}

	allRemoves := make(map[*mempool.TxEntry]struct{})
	for it := range txToRemove {
		m.CalculateDescendants(it, allRemoves)
	}
	m.RemoveStaged(allRemoves, false, mempool.REORG)
}
