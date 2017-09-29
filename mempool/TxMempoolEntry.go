package mempool

import (
	"unsafe"

	"github.com/btcboost/copernicus/model"
	"gopkg.in/fatih/set.v0"
)

/* TxMempoolEntry stores data about the corresponding transaction, as well as
 * data about all in-mempool transactions that depend on the transaction
 * ("descendant" transactions).
 *
 * When a new entry is added to the mempool, we update the descendant state
 * (nCountWithDescendants, nSizeWithDescendants, and nModFeesWithDescendants)
 * for all ancestors of the newly added transaction.
 *
 * If updating the descendant state is skipped, we can mark the entry as
 * "dirty", and set nSizeWithDescendants/nModFeesWithDescendants to equal
 * nTxSize/nFee+feeDelta. (This can potentially happen during a reorg, where we
 * limit the amount of work we're willing to do to avoid consuming too much
 * CPU.)
 */

type TxMempoolEntry struct {
	TxRef         *model.Tx
	Fee           int64
	TxSize        int
	ModSize       int
	UsageSize     int
	LocalTime     int64
	EntryPriority float64
	EntryHeight   uint
	//!< Sum of all txin values that are already in blockchain
	InChainInputValue int64
	SpendsCoinbase    bool
	SigOpCount        int64
	FeeDelta          int64

	LockPoints *LockPoints
	// Information about descendants of this transaction that are in the
	// mempool; if we remove this transaction we must remove all of these
	// descendants as well.  if nCountWithDescendants is 0, treat this entry as
	// dirty, and nSizeWithDescendants and nModFeesWithDescendants will not be
	// correct.
	//!< number of descendant transactions
	CountWithDescendants   uint64
	SizeWithDescendants    uint64
	ModFeesWithDescendants int64

	// Analogous statistics for ancestor transactions
	nCountWithAncestors     uint64
	sizeWithAncestors       uint64
	ModFeesWithAncestors    int64
	SigOpCoungWithAncestors int64
}

func (txMempoolEntry *TxMempoolEntry) GetPriority(currentHeight uint) float64 {
	deltaPriority := (float64(currentHeight-txMempoolEntry.EntryHeight) * float64(txMempoolEntry.InChainInputValue)) / float64(txMempoolEntry.ModSize)
	result := txMempoolEntry.EntryPriority + deltaPriority
	if result < 0 {
		result = 0
	}
	return result
}

func (txMempoolEntry *TxMempoolEntry) UpdateLockPoints(lockPoint *LockPoints) {
	txMempoolEntry.LockPoints = lockPoint
}

func (txMempoolEntry *TxMempoolEntry) GetModifiedFee() int64 {
	return txMempoolEntry.Fee + txMempoolEntry.FeeDelta
}

func (txMempoolEntry *TxMempoolEntry) UpdateFeeDelta(newFeeDelta int64) {
	txMempoolEntry.ModFeesWithDescendants = txMempoolEntry.ModFeesWithDescendants + newFeeDelta - txMempoolEntry.FeeDelta
	txMempoolEntry.ModFeesWithAncestors = txMempoolEntry.ModFeesWithAncestors + newFeeDelta - txMempoolEntry.FeeDelta
	txMempoolEntry.FeeDelta = newFeeDelta

}

func IncrementalDynamicUsageTxMempoolEntry(s *set.Set) int64 {
	var size int64
	for _, entry := range s.List() {
		size += int64(unsafe.Sizeof(entry))
	}
	return size
}
func NewTxMempoolEntry(txRef *model.Tx, fee int64, time int64,
	entryPriority float64, entryHeight uint, inChainInputValue int64, spendCoinbase bool,
	sigOpsCount int64, lockPoints *LockPoints) *TxMempoolEntry {
	txMempoolEntry := TxMempoolEntry{}

	txMempoolEntry.TxRef = txRef
	txMempoolEntry.Fee = fee
	txMempoolEntry.EntryPriority = entryPriority
	txMempoolEntry.EntryHeight = entryHeight
	txMempoolEntry.InChainInputValue = inChainInputValue
	txMempoolEntry.SpendsCoinbase = spendCoinbase
	txMempoolEntry.SigOpCount = sigOpsCount
	txMempoolEntry.LockPoints = lockPoints

	txMempoolEntry.TxSize = txRef.SerializeSize()
	txMempoolEntry.ModSize = txRef.CalculateModifiedSize()
	txMempoolEntry.UsageSize = RecursiveDynamicUsage(txRef)

	txMempoolEntry.CountWithDescendants = 1
	txMempoolEntry.SizeWithDescendants = uint64(txMempoolEntry.TxSize)
	txMempoolEntry.ModFeesWithDescendants = fee
	valueIn := txRef.GetValueOut() + fee

	if inChainInputValue > valueIn {
		panic("error inChainInputValue > valueIn ")
	}
	txMempoolEntry.FeeDelta = 0
	txMempoolEntry.nCountWithAncestors = 1
	txMempoolEntry.sizeWithAncestors = uint64(txMempoolEntry.TxSize)
	txMempoolEntry.ModFeesWithAncestors = fee
	txMempoolEntry.SigOpCoungWithAncestors = sigOpsCount

	return &txMempoolEntry
}
