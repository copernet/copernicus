package lmempool

import (
	"container/list"
	"fmt"
	"github.com/copernet/copernicus/model/wallet"
	"math"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util"
)

func AcceptTxToMemPool(txn *tx.Tx) error {
	pool := mempool.GetInstance()

	txEntry, err := ltx.CheckTxBeforeAcceptToMemPool(txn, pool)
	if err != nil {
		return err
	}
	pool.Lock()
	defer pool.Unlock()
	return addTxToMemPool(pool, txEntry)
}

func addTxToMemPool(pool *mempool.TxMempool, txe *mempool.TxEntry) error {
	ancestorNum := conf.Cfg.Mempool.LimitAncestorCount
	ancestorSize := conf.Cfg.Mempool.LimitAncestorSize
	descendantNum := conf.Cfg.Mempool.LimitDescendantCount
	descendantSize := conf.Cfg.Mempool.LimitDescendantSize

	ancestors, err := pool.CalculateMemPoolAncestors(txe.Tx, uint64(ancestorNum), uint64(ancestorSize*1000),
		uint64(descendantNum), uint64(descendantSize*1000), true)

	if err != nil {
		return err
	}

	err = pool.AddTx(txe, ancestors)
	if err != nil {
		log.Error("add tx failed:%s", err.Error())
		return err
	}

	// TODO: simple implementation just for testing, remove this after complete wallet
	if wallet.GetInstance().IsEnable() {
		wallet.GetInstance().HandleRelatedMempoolTx(txe.Tx)
	}
	return nil
}

func TryAcceptOrphansTxs(transaction *tx.Tx, chainHeight int32, checkLockPoint bool) (acceptTxs []*tx.Tx, rejectTxs []util.Hash) {
	vWorkQueue := make([]outpoint.OutPoint, 0)
	pool := mempool.GetInstance()

	// first collect this tx all outPoint.
	for i := 0; i < transaction.GetOutsCount(); i++ {
		o := outpoint.OutPoint{Hash: transaction.GetHash(), Index: uint32(i)}
		vWorkQueue = append(vWorkQueue, o)
	}

	setMisbehaving := make(map[int64]struct{})
	for len(vWorkQueue) > 0 {
		prevOut := vWorkQueue[0]
		vWorkQueue = vWorkQueue[1:]
		if orphans, ok := pool.OrphanTransactionsByPrev[prevOut]; ok {
			for _, iOrphanTx := range orphans {
				fromPeer := iOrphanTx.NodeID
				if _, ok := setMisbehaving[fromPeer]; ok {
					continue
				}

				err := AcceptTxToMemPool(iOrphanTx.Tx)
				if err == nil {
					acceptTxs = append(acceptTxs, iOrphanTx.Tx)
					for i := 0; i < iOrphanTx.Tx.GetOutsCount(); i++ {
						o := outpoint.OutPoint{Hash: iOrphanTx.Tx.GetHash(), Index: uint32(i)}
						vWorkQueue = append(vWorkQueue, o)
					}
					pool.EraseOrphanTx(iOrphanTx.Tx.GetHash(), false)
					break
				}

				if !errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut) {
					pool.EraseOrphanTx(iOrphanTx.Tx.GetHash(), true)
					if errcode.IsErrorCode(err, errcode.RejectTx) {
						rejectTxs = append(rejectTxs, iOrphanTx.Tx.GetHash())
					}
					break
				}
			}
		}
	}
	return
}

func RemoveTxSelf(txs []*tx.Tx) {
	pool := mempool.GetInstance()
	pool.RemoveTxSelf(txs)
}

// func RemoveForReorg(nMemPoolHeight int32, flag int) {
// 	newPool := mempool.NewTxMempool()
// 	oldPool := mempool.GetInstance()
// 	log.Debug("RemoveForReorg start")
// 	mempool.SetInstance(newPool)
// 	for _, txentry := range oldPool.GetAllTxEntry() {
// 		txn := txentry.Tx
// 		err := AcceptTxToMemPool(txn)
// 		if err == nil {
// 			accepttxn, _ := TryAcceptOrphansTxs(txn, nMemPoolHeight-1, true)
// 			log.Debug("RemoveForReorg move %v to mempool", append(accepttxn, txn))
// 			if !newPool.HaveTransaction(txn) {
// 				log.Error("the tx:%s not exist mempool", txn.GetHash().String())
// 				return
// 			}
// 		} else {
// 			if errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut) {
// 				newPool.AddOrphanTx(txn, 0)
// 			}
// 		}
// 	}
// 	newPool.CleanOrphan()
// 	CheckMempool(nMemPoolHeight - 1)
// 	log.Debug("RemoveForReorg end")
// }

func RemoveForReorg(nMemPoolHeight int32, flag int) {
	view := utxo.GetUtxoCacheInstance()
	pool := mempool.GetInstance()
	pool.Lock()
	defer pool.Unlock()

	// Remove transactions spending a coinbase which are now immature and
	// no-longer-final transactions
	txToRemove := make(map[*mempool.TxEntry]struct{})
	allEntry := pool.GetAllTxEntryWithoutLock()
	for _, entry := range allEntry {
		tmpTx := entry.Tx

		lp := entry.GetLockPointFromTxEntry()
		if ltx.ContextualCheckTransactionForCurrentBlock(tmpTx, flag) != nil ||
			!ltx.CheckSequenceLocks(lp.Height, lp.Time) {
			txToRemove[entry] = struct{}{}
		} else if entry.GetSpendsCoinbase() {
			for _, preout := range tmpTx.GetAllPreviousOut() {
				if _, ok := allEntry[preout.Hash]; ok {
					continue
				}

				coin := view.GetCoin(&preout)
				if pool.GetCheckFrequency() != 0 {
					if coin.IsSpent() {
						panic("the coin must be unspent")
					}
				}

				if coin.IsSpent() || (coin.IsCoinBase() &&
					nMemPoolHeight-coin.GetHeight() < consensus.CoinbaseMaturity) {
					txToRemove[entry] = struct{}{}
					break
				}
			}
		}
		entry.SetLockPointFromTxEntry(lp)
	}

	for it := range txToRemove {
		staged := pool.CalculateDescendants(it)
		staged[it] = struct{}{}
		pool.RemoveStaged(staged, false, mempool.REORG)
	}
}

func updateCoins(coinsMap *utxo.CoinsMap, trax *tx.Tx) {
	isCoinBase := trax.IsCoinBase()
	if !isCoinBase {
		for _, preout := range trax.GetAllPreviousOut() {
			if coinsMap.SpendCoin(&preout) == nil {
				panic(fmt.Sprintf("no spentable coin for prevout(%v)", preout))
			}
		}
	}
	for i := 0; i < trax.GetOutsCount(); i++ {
		a := outpoint.OutPoint{Hash: trax.GetHash(), Index: uint32(i)}
		coinsMap.AddCoin(&a, utxo.NewFreshCoin(trax.GetTxOut(i), 1000000, isCoinBase), isCoinBase)
	}
}

func haveInputs(coinsMap *utxo.CoinsMap, trax *tx.Tx) bool {
	if trax.IsCoinBase() {
		return true
	}
	for _, preout := range trax.GetAllPreviousOut() {
		if coinsMap.FetchCoin(&preout) == nil {
			return false
		}
	}
	return true
}

// CheckMempool check If sanity-checking is turned on, check makes sure the pool is consistent
// (does not contain two transactions that spend the same inputs, all inputs
// are in the mapNextTx array). If sanity-checking is turned off, check does
// nothing.
func CheckMempool(bestHeight int32) {
	spentHeight := bestHeight + 1
	view := utxo.GetUtxoCacheInstance()
	pool := mempool.GetInstance()
	pool.RLock()
	defer pool.RUnlock()

	if pool.GetCheckFrequency() == 0 {
		return
	}
	if util.GetRand(math.MaxUint32) >= pool.GetCheckFrequency() {
		return
	}
	// activaChain := chain.GetInstance()
	mempoolDuplicate := utxo.NewEmptyCoinsMap()
	allEntry := pool.GetAllTxEntryWithoutLock()
	spentOut := pool.GetAllSpentOutWithoutLock()
	//bestHash, _ := view.GetBestBlock()
	//bestHeigh := activaChain.FindBlockIndex(bestHash).Height + 1
	log.Debug("mempool", fmt.Sprintf("checking mempool with %d transaction and %d inputs ...", len(allEntry), len(spentOut)))
	checkTotal := uint64(0)

	waitingOnDependants := list.New()

	// foreach every txentry in mempool, and check these txentry correctness.
	for _, entry := range allEntry {

		checkTotal += uint64(entry.TxSize)
		fDependsWait := false
		setParentCheck := make(map[util.Hash]struct{})

		for _, preout := range entry.Tx.GetAllPreviousOut() {
			if entry, ok := allEntry[preout.Hash]; ok {
				tx2 := entry.Tx
				if !(tx2.GetOutsCount() > int(preout.Index) && !tx2.GetTxOut(int(preout.Index)).IsNull()) {
					panic("the tx introduced input dose not exist, or the input amount is nil ")
				}

				fDependsWait = true
				setParentCheck[tx2.GetHash()] = struct{}{}
			} else {
				if !view.HaveCoin(&preout) {
					panic("the tx introduced input dose not exist mempool And UTXO set !!!")
				}
			}

			if e := pool.HasSPentOutWithoutLock(&preout); e == nil {
				panic("the introduced tx is not in mempool")
			}
		}
		if len(setParentCheck) != len(entry.ParentTx) {
			panic("the two parent set should be equal")
		}

		// Verify ancestor state is correct.
		nNoLimit := uint64(math.MaxUint64)
		setAncestors, err := pool.CalculateMemPoolAncestors(entry.Tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, false)
		if err != nil {
			return
		}
		nCountCheck := int64(len(setAncestors)) + 1
		nSizeCheck := int64(entry.TxSize)
		nSigOpCheck := int64(entry.SigOpCount)
		nFeesCheck := entry.TxFee
		for ancestorIt := range setAncestors {
			nSizeCheck += int64(ancestorIt.TxSize)
			nSigOpCheck += int64(ancestorIt.SigOpCount)
			nFeesCheck += ancestorIt.TxFee
		}
		if entry.SumTxCountWithAncestors != nCountCheck {
			log.Error(
				"the txentry's ancestors number is incorrect: entry.SumTxCountWithAncestors(%d) nCountCheck(%d)",
				entry.SumTxCountWithAncestors, nCountCheck)
			entry.SumTxCountWithAncestors = nCountCheck
		}
		if entry.SumTxSizeWitAncestors != nSizeCheck {
			log.Error("the txentry's ancestors size is incorrect: entry.SumTxSizeWitAncestors(%d) nSizeCheck(%d)",
				entry.SumTxSizeWitAncestors, nSizeCheck)
			entry.SumTxSizeWitAncestors = nSizeCheck
		}
		if entry.SumTxSigOpCountWithAncestors != nSigOpCheck {
			log.Error("the txentry's ancestors sigopcount is incorrect: entry.SumTxSigOpCountWithAncestors(%d), nSigOpCheck(%d)",
				entry.SumTxSigOpCountWithAncestors, nSigOpCheck)
			entry.SumTxSigOpCountWithAncestors = nSigOpCheck
		}
		if entry.SumTxFeeWithAncestors != nFeesCheck {
			log.Error("the txentry's ancestors feew is incorrect: entry.SumTxFeeWithAncestors(%d), nFeesCheck(%d)",
				entry.SumTxFeeWithAncestors, nFeesCheck)
			entry.SumTxFeeWithAncestors = nFeesCheck
		}

		setChildrenCheck := make(map[*mempool.TxEntry]struct{})
		childSize := 0
		for i := 0; i < entry.Tx.GetOutsCount(); i++ {
			o := outpoint.OutPoint{Hash: entry.Tx.GetHash(), Index: uint32(i)}
			if e := pool.HasSPentOutWithoutLock(&o); e != nil {
				if _, ok := allEntry[e.Tx.GetHash()]; !ok {
					panic("the transaction should be in mempool ...")
				}
				if _, ok := setChildrenCheck[e]; !ok {
					setChildrenCheck[e] = struct{}{}
					childSize += e.TxSize
				}
			}
		}

		if len(setChildrenCheck) != len(entry.ChildTx) {
			panic("the transaction children set is different ...")
		}
		if entry.SumTxSizeWithDescendants < int64(childSize+entry.TxSize) {
			panic("the transaction descendant's fee is less its children fee ...")
		}

		// Also check to make sure size is greater than sum with immediate
		// children. Just a sanity check, not definitive that this calc is
		// correct...
		if fDependsWait {
			waitingOnDependants.PushBack(entry)
		} else {
			fCheckResult := entry.Tx.IsCoinBase()
			if !fCheckResult {
				for _, e := range entry.Tx.GetIns() {
					mempoolDuplicate.FetchCoin(e.PreviousOutPoint)
				}
				fCheckResult = ltx.CheckInputsMoney(entry.Tx, mempoolDuplicate,
					spentHeight) == nil
			}

			if !fCheckResult {
				panic("the txentry check failed with utxo set1...")
			}
			updateCoins(mempoolDuplicate, entry.Tx)
		}
	}

	stepsSinceLastRemove := 0
	for waitingOnDependants.Len() > 0 {
		it := waitingOnDependants.Front()
		entry := it.Value.(*mempool.TxEntry)
		waitingOnDependants.Remove(it)

		if !haveInputs(mempoolDuplicate, entry.Tx) {
			waitingOnDependants.PushBack(entry)
			stepsSinceLastRemove++
			if stepsSinceLastRemove >= waitingOnDependants.Len() {
				panic(fmt.Sprintf(
					"stepsSinceLastRemove(%d) should be less then length of waitingOnDependants(%d)",
					stepsSinceLastRemove, waitingOnDependants.Len()))
			}
		} else {
			fCheckResult := entry.Tx.IsCoinBase()
			if !fCheckResult {
				for _, e := range entry.Tx.GetIns() {
					mempoolDuplicate.FetchCoin(e.PreviousOutPoint)
				}
				fCheckResult = ltx.CheckInputsMoney(entry.Tx, mempoolDuplicate,
					spentHeight) == nil
			}
			if !fCheckResult {
				panic("the txentry check failed with utxo set2...")
			}
			updateCoins(mempoolDuplicate, entry.Tx)
			stepsSinceLastRemove = 0
		}
	}

	for _, entry := range spentOut {
		txid := entry.Tx.GetHash()
		if e, ok := allEntry[txid]; !ok {
			panic("the transaction not exist in mempool. . .")
		} else {
			if e.Tx != entry.Tx {
				panic("mempool store the transaction is different with it's two struct . . .")
			}
		}
	}

	if pool.GetPoolAllTxSize(false) != checkTotal {
		panic("mempool have all transaction size state is incorrect ...")
	}
}

func FindTxInMempool(hash util.Hash) *mempool.TxEntry {
	pool := mempool.GetInstance()
	return pool.FindTx(hash)
}

func FindOrphanTxInMemPool(hash util.Hash) *tx.Tx {
	pool := mempool.GetInstance()
	pool.RLock()
	defer pool.RUnlock()
	if orphan, ok := pool.OrphanTransactions[hash]; ok {
		return orphan.Tx
	}

	return nil
}
