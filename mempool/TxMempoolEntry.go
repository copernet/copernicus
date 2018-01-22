package mempool

import (
	"fmt"

	"unsafe"

	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
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
	Fee           btcutil.Amount
	TxSize        int
	ModSize       int
	UsageSize     int
	Time          int64
	EntryPriority float64
	EntryHeight   uint
	//!< Sum of all txin values that are already in blockchain
	InChainInputValue btcutil.Amount
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
	ModFeesWithDescendants btcutil.Amount

	// Analogous statistics for ancestor transactions
	CountWithAncestors      uint64
	sizeWithAncestors       uint64
	ModFeesWithAncestors    btcutil.Amount
	SigOpCountWithAncestors int64
	//Index in mempool's vTxHashes
	vTxHashesIdx int
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

func (txMempoolEntry *TxMempoolEntry) GetModifiedFee() btcutil.Amount {
	return txMempoolEntry.Fee + btcutil.Amount(txMempoolEntry.FeeDelta)
}

func (txMempoolEntry *TxMempoolEntry) UpdateFeeDelta(newFeeDelta int64) {
	txMempoolEntry.ModFeesWithDescendants += btcutil.Amount(newFeeDelta - txMempoolEntry.FeeDelta)
	txMempoolEntry.ModFeesWithAncestors += btcutil.Amount(newFeeDelta - txMempoolEntry.FeeDelta)
	txMempoolEntry.FeeDelta = newFeeDelta

}

func (txMempoolEntry *TxMempoolEntry) GetFeeRate() *utils.FeeRate {
	return utils.NewFeeRateWithSize(int64(txMempoolEntry.Fee), txMempoolEntry.TxSize)
}

func (txMempoolEntry *TxMempoolEntry) GetFeeDelta() int64 {
	return int64(txMempoolEntry.GetModifiedFee()) - int64(txMempoolEntry.Fee)
}

func (txMempoolEntry *TxMempoolEntry) UpdateAncestorState(modifySize, modifyCount, modifySigOps int64, modifyFee btcutil.Amount) {
	if modifySize < 0 && uint64(-modifySize) > txMempoolEntry.sizeWithAncestors {
		panic("the Ancestors's object size should not be negative")
	}
	if modifyCount < 0 && uint64(-modifyCount) > txMempoolEntry.CountWithAncestors {
		panic("the Ancestors's number should not be negative")
	}

	if modifySize < 0 {
		txMempoolEntry.sizeWithAncestors -= uint64(-modifySize)
	} else {
		txMempoolEntry.sizeWithAncestors += uint64(modifySize)
	}
	if modifyCount < 0 {
		txMempoolEntry.CountWithAncestors -= uint64(-modifyCount)
	} else {
		txMempoolEntry.CountWithAncestors += uint64(modifyCount)
	}
	txMempoolEntry.ModFeesWithAncestors += modifyFee
	txMempoolEntry.SigOpCountWithAncestors += modifySigOps
	if txMempoolEntry.SigOpCountWithAncestors < 0 {
		panic("the Ancestors's sigOpCode Number should not be negative")
	}
}

func (txMempoolEntry *TxMempoolEntry) UpdateDescendantState(modifySize int64, modifyFee btcutil.Amount, modifyCount int64) {
	if modifySize < 0 && uint64(-modifySize) > txMempoolEntry.SizeWithDescendants {
		panic("the Descendants's object size should not be negative")
	}
	if modifyCount < 0 && uint64(-modifyCount) > txMempoolEntry.CountWithDescendants {
		panic("the Descendants's number should not be negative")
	}
	if modifySize < 0 {
		txMempoolEntry.SizeWithDescendants -= uint64(-modifySize)
	} else {
		txMempoolEntry.SizeWithDescendants += uint64(modifySize)
	}

	if modifyCount < 0 {
		txMempoolEntry.CountWithDescendants -= uint64(-modifyCount)
	} else {
		txMempoolEntry.CountWithDescendants += uint64(modifyCount)
	}
	txMempoolEntry.ModFeesWithDescendants += modifyFee

}

func CompareTxMemPoolEntryByDescendantScore(src, dst interface{}) bool {
	a := src.(*TxMempoolEntry)
	b := dst.(*TxMempoolEntry)
	UseADescendants := useDescendantScore(a)
	UseBDescendants := useDescendantScore(b)

	var aModFee, bModFee uint64
	var aSize, bSize uint64
	if UseADescendants {
		aModFee = uint64(a.ModFeesWithDescendants)
		aSize = a.SizeWithDescendants
	} else {
		aModFee = uint64(a.GetModifiedFee())
		aSize = uint64(a.TxSize)
	}

	if UseBDescendants {
		bModFee = uint64(b.ModFeesWithDescendants)
		bSize = b.SizeWithDescendants
	} else {
		bModFee = uint64(b.GetModifiedFee())
		bSize = uint64(b.TxSize)
	}

	// Avoid division by rewriting (a/b > c/d) as (a*d > c*b).
	f1 := aModFee * bSize
	f2 := aSize * bModFee
	if f1 == f2 {
		fmt.Printf("&&&&&&&&&&&&&&&&&&&&&&\n  origin Hash : %v; \n  dst Hash : %v; \n  "+
			"compare : %v \n",
			a.TxRef.Hash.ToString(), b.TxRef.Hash.ToString(), a.Time >= b.Time)
		return a.Time >= b.Time
	}
	//fmt.Printf("============== oriHash : %v,\n dstHash : %v,\n  compare : %v \n",
	//	a.TxRef.Hash.ToString(), b.TxRef.Hash.ToString(), f1 < f2)

	return f1 < f2
}

func useDescendantScore(a *TxMempoolEntry) bool {
	f1 := uint64(a.GetModifiedFee()) * a.SizeWithDescendants
	f2 := uint64(a.ModFeesWithDescendants) * uint64(a.TxSize)
	return f2 > f1
}

func DepthAndScoreComparator(a *TxMempoolEntry, b *TxMempoolEntry) bool {
	counta := a.CountWithAncestors
	countb := b.CountWithAncestors
	if counta == countb {
		return CompareTxMempoolEntryByScore(a, b)
	}
	return counta < countb

}

func CompareTxMemPoolEntryByAncestorFee(src, dst interface{}) bool {
	a := src.(*TxMempoolEntry)
	b := dst.(*TxMempoolEntry)
	aFees := uint64(a.ModFeesWithAncestors)
	aSize := a.sizeWithAncestors

	bFees := uint64(b.ModFeesWithAncestors)
	bSize := b.sizeWithAncestors

	// Avoid division by rewriting (a/b > c/d) as (a*d > c*b).
	f1 := aFees * bSize
	f2 := aSize * bFees

	if f1 == f2 {
		return a.TxRef.Hash.ToBigInt().Cmp(b.TxRef.Hash.ToBigInt()) < 0
	}

	return f1 > f2
}

func CompareTxMemPoolEntryByEntryTime(src, dst interface{}) bool {
	a := src.(*TxMempoolEntry)
	b := dst.(*TxMempoolEntry)
	return a.Time < b.Time
}

// CompareTxMempoolEntryByScore Sort by score of entry ((fee+delta)/size) in descending order
func CompareTxMempoolEntryByScore(src, dst interface{}) bool {
	a := src.(*TxMempoolEntry)
	b := dst.(*TxMempoolEntry)
	f1 := float64(a.GetModifiedFee()) * float64(b.TxSize)
	f2 := float64(b.GetModifiedFee()) * float64(a.TxSize)
	if f1 == f2 {
		return a.TxRef.Hash.ToBigInt().Cmp(b.TxRef.Hash.ToBigInt()) > 0
	}
	return f1 > f2
}

func IncrementalDynamicUsageTxMempoolEntry(s *set.Set) int64 {
	size := int64(MallocUsage(int(unsafe.Sizeof(s))))
	return size
}
func NewTxMempoolEntry(txRef *model.Tx, fee btcutil.Amount, time int64,
	entryPriority float64, entryHeight uint, inChainInputValue btcutil.Amount, spendCoinbase bool,
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
	valueIn := btcutil.Amount(txRef.GetValueOut()) + fee

	if inChainInputValue > valueIn {
		panic("error inChainInputValue > valueIn ")
	}
	txMempoolEntry.FeeDelta = 0
	txMempoolEntry.CountWithAncestors = 1
	txMempoolEntry.sizeWithAncestors = uint64(txMempoolEntry.TxSize)
	txMempoolEntry.ModFeesWithAncestors = fee
	txMempoolEntry.SigOpCountWithAncestors = sigOpsCount

	return &txMempoolEntry
}
