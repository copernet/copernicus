package mempool

import (
	"github.com/btcboost/copernicus/errcode"
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/astaxie/beego/logs"
	"fmt"
	"math"
	"container/list"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/conf"
)

// AccpetTxToMemPool add one check corret transaction to mempool.
func AccpetTxToMemPool(tx *tx.Tx ) error {

	//first : check transaction context And itself.
	if err := ltx.CheckRegularTransaction(tx); err != nil {
		return err
	}

	//second : check whether enter mempool.
	pool := mempool.GetInstance()
	pool.Lock()
	defer pool.Unlock()
	utxoTip := utxo.GetUtxoCacheInstance()
	mpHeight := 0
	allPreout := tx.GetAllPreviousOut()
	coins := make([]*utxo.Coin, len(allPreout))
	var txfee int64
	var inputValue int64
	for i, preout := range allPreout {
		if coin := utxoTip.GetCoin(&preout); coin != nil {
			coins[i] = coin
			inputValue += int64(coin.GetAmount())
		} else {
			if coin := pool.GetCoin(&preout); coin != nil {
				coins[i] = coin
				inputValue += int64(coin.GetAmount())
			} else {
				panic("the transaction in mempool, not found its parent " +
					"transaction in local node and utxo")
			}
		}
	}
	txfee = inputValue - tx.GetValueOut()
	ancestors, lp, err := isAcceptTx(tx, txfee)
	if err != nil {
		return err
	}

	//three : add transaction to mempool.
	txentry := mempool.NewTxentry(tx, txfee, 0, mpHeight, *lp, 0, false)
	pool.AddTx(txentry, ancestors)

	return nil
}

func ProcessOrphan(transaction *tx.Tx) []*tx.Tx {
	vWorkQueue := make([]outpoint.OutPoint, 0)
	acceptTx := make([]*tx.Tx, 0)
	pool := mempool.GetInstance()

	// first collect this tx all outPoint.
	for i := 0; i < transaction.GetOutsCount(); i++ {

		o := outpoint.OutPoint{Hash: transaction.GetHash(), Index: uint32(i)}
		vWorkQueue = append(vWorkQueue, o)
	}

	//todo !!! modify this transaction send node time .
	//pfrom->nLastTXTime = GetTime();
	setMisbehaving := make(map[int64]struct{}, 0)
	for len(vWorkQueue) > 0 {
		prevOut := vWorkQueue[0]
		vWorkQueue = vWorkQueue[1:]
		if orphans, ok := pool.OrphanTransactionsByPrev[prevOut]; !ok {
			continue
		} else {
			for _, iOrphanTx := range orphans {
				fromPeer := iOrphanTx.NodeID
				if _, ok := setMisbehaving[fromPeer]; ok {
					continue
				}

				err2 := AccpetTxToMemPool(iOrphanTx.Tx)
				if err2 == nil {
					acceptTx = append(acceptTx, iOrphanTx.Tx)
					for i := 0; i < iOrphanTx.Tx.GetOutsCount(); i++ {
						o := outpoint.OutPoint{Hash: iOrphanTx.Tx.GetHash(), Index: uint32(i)}
						vWorkQueue = append(vWorkQueue, o)
					}
					pool.EraseOrphanTx(iOrphanTx.Tx.GetHash(), false)
					break
				}

				if !errcode.IsErrorCode(err2, errcode.TxErrNoPreviousOut) {
					pool.EraseOrphanTx(iOrphanTx.Tx.GetHash(), true)
					if errcode.IsErrorCode(err2, errcode.RejectTx) {
						pool.RecentRejects[iOrphanTx.Tx.GetHash()] = struct{}{}
					}
					break
				}
			}
		}
	}

	return acceptTx
}

func isAcceptTx(tx *tx.Tx, txfee int64 ) (map[*mempool.TxEntry]struct{}, *mempool.LockPoints, error) {
	pool := mempool.GetInstance()
	allEntry := pool.GetAllTxEntryWithoutLock()
	if _, ok := allEntry[tx.GetHash()]; ok {
		return nil, nil, errcode.New(errcode.AlreadHaveTx)
	}

	lp := ltx.CalculateSequenceLocks(tx, consensus.LocktimeVerifySequence|consensus.LocktimeMedianTimePast)
	if !ltx.CheckSequenceLocks(lp){
		return nil, lp, errcode.New(errcode.Nomature)
	}

	ancestorNum := conf.Cfg.Mempool.LimitAncestorCount
	ancestorSize := conf.Cfg.Mempool.LimitAncestorSize
	descendantNum := conf.Cfg.Mempool.LimitDescendantCount
	descendantSize := conf.Cfg.Mempool.LimitDescendantSize
	ancestors, err := pool.CalculateMemPoolAncestors(tx, uint64(ancestorNum), uint64(ancestorSize*1000),
		uint64(descendantNum), uint64(descendantSize*1000), true)
	if err != nil {
		return nil, lp, err
	}

	txsize := int64(tx.EncodeSize())
	minfeeRate := pool.GetMinFee(conf.Cfg.Mempool.MaxPoolSize)
	rejectFee := minfeeRate.GetFee(int(txsize))
	// compare the transaction feeRate with enter mempool min txfeeRate
	if txfee < rejectFee{
		return nil, lp, errcode.New(errcode.TooMinFeeRate)
	}

	return ancestors, lp, nil
}

func RemoveTxSelf(txs []*tx.Tx)  {
	pool := mempool.GetInstance()
	pool.RemoveTxSelf(txs)
}

func RemoveTxRecursive(origTx *tx.Tx, reason mempool.PoolRemovalReason)  {
	pool := mempool.GetInstance()
	pool.RemoveTxRecursive(origTx, reason)
}

func RemoveUnFinalTx(chain *chain.Chain, view utxo.CacheView, nMemPoolHeight int32, flag int) {
	pool := mempool.GetInstance()
	pool.Lock()
	defer pool.Unlock()

	// Remove transactions spending a coinbase which are now immature and
	// no-longer-final transactions
	txToRemove := make(map[*mempool.TxEntry]struct{})
	allEntry := pool.GetAllTxEntryWithoutLock()
	for _, entry := range allEntry {
		lp := entry.GetLockPointFromTxEntry()
		validLP := entry.CheckLockPointValidity(chain)
		//state := NewValidationState()

		tx := entry.Tx
		allPreout := tx.GetAllPreviousOut()
		coins := make([]*utxo.Coin, len(allPreout))
		for i, preout := range allPreout {
			if coin := view.GetCoin(&preout); coin != nil {
				coins[i] = coin
			} else {
				if coin := pool.GetCoin(&preout); coin != nil {
					coins[i] = coin
				} else {
					panic("the transaction in mempool, not found its parent " +
						"transaction in local node and utxo")
				}
			}
		}
		tlp := ltx.CalculateSequenceLocks(tx, uint(flag))
		if ltx.ContextualCheckTransactionForCurrentBlock(tx, flag) != nil ||
			!ltx.CheckSequenceLocks(tlp) {
			txToRemove[entry] = struct{}{}
		} else if entry.GetSpendsCoinbase() {
			for _, preout := range tx.GetAllPreviousOut() {
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
					nMemPoolHeight - coin.GetHeight() < consensus.CoinbaseMaturity) {
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
		pool.CalculateDescendants(it, allRemoves)
	}
	pool.RemoveStaged(allRemoves, false, mempool.REORG)
}

// Check If sanity-checking is turned on, check makes sure the pool is consistent
// (does not contain two transactions that spend the same inputs, all inputs
// are in the mapNextTx array). If sanity-checking is turned off, check does
// nothing.
func CheckMempool(view utxo.CacheView) {
	pool := mempool.GetInstance()
	pool.Lock()
	defer pool.Unlock()

	if pool.GetCheckFrequency() == 0 {
		return
	}
	if float64(util.GetRand(math.MaxUint32)) >= pool.GetCheckFrequency() {
		return
	}
	activaChain := chain.GetInstance()
	mempoolDuplicate := utxo.NewEmptyCoinsMap()
	allEntry := pool.GetAllTxEntryWithoutLock()
	spentOut := pool.GetAllSpentOutWithoutLock()
	bestHeigh := activaChain.FindBlockIndex(view.GetBestBlock()).Height + 1
	logs.SetLogger("mempool", fmt.Sprintf("checking mempool with %d transaction and %d inputs ...", len(allEntry), len(spentOut)))
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
				if !(tx2.GetOutsCount() > int(preout.Index)) {
					if !tx2.GetTxOut(int(preout.Index)).IsNull() {
						panic("the tx introduced input dose not exist, or the input amount is nil ")
					}
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
		setAncestors, err := pool.CalculateMemPoolAncestors(entry.Tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, true)
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
			panic("the txentry's ancestors number is incorrect .")
		}
		if entry.SumSizeWitAncestors != nSizeCheck {
			panic("the txentry's ancestors size is incorrect .")
		}
		if entry.SumSigOpCountWithAncestors != nSigOpCheck {
			panic("the txentry's ancestors sigopcount is incorrect .")
		}
		if entry.SumFeeWithAncestors != nFeesCheck {
			panic("the txentry's ancestors fee is incorrect .")
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
		if entry.SumSizeWithDescendants < int64(childSize+entry.TxSize) {
			panic("the transaction descendant's fee is less its children fee ...")
		}

		// Also check to make sure size is greater than sum with immediate
		// children. Just a sanity check, not definitive that this calc is
		// correct...
		if fDependsWait {
			waitingOnDependants.PushBack(entry)
		} else {
			//todo !!! need to fix error in here by yyx.
			//fCheckResult := entry.Tx.IsCoinBase() || ltx.CheckInputsMoney(entry.Tx, ) == nil
			//if !fCheckResult {
			//	panic("the txentry check failed with utxo set...")
			//}
			//ltx.CheckInputsMoney(entry.Tx, view, bestHeigh)
			_ = bestHeigh

			for _, preout := range entry.Tx.GetAllPreviousOut() {
				mempoolDuplicate.SpendCoin(&preout)
			}
			isCoinBase := entry.Tx.IsCoinBase()
			for i := 0; i < entry.Tx.GetOutsCount(); i++ {
				a := outpoint.OutPoint{entry.Tx.GetHash(), uint32(i)}
				mempoolDuplicate.AddCoin(&a, utxo.NewCoin(entry.Tx.GetTxOut(i), 1000000, isCoinBase))
			}
		}
	}

	stepsSinceLastRemove := 0
	for waitingOnDependants.Len() > 0 {
		it := waitingOnDependants.Front()
		entry := it.Value.(*mempool.TxEntry)
		waitingOnDependants.Remove(it)
		spend := false
		for _, preOut := range entry.Tx.GetAllPreviousOut() {
			co := mempoolDuplicate.GetCoin(&preOut)
			if !(co != nil && !co.IsSpent()) {
				waitingOnDependants.PushBack(entry)
				stepsSinceLastRemove++
				if !(stepsSinceLastRemove < waitingOnDependants.Len()) {
					panic("the waitingOnDependants list have incorrect number ...")
				}
				spend = true
				break
			}
		}

		if spend {
			fCheckResult := entry.Tx.IsCoinBase()

			for _, preOut := range entry.Tx.GetAllPreviousOut() {
				co := mempoolDuplicate.GetCoin(&preOut)
				if !(co != nil && !co.IsSpent()) {
					fCheckResult = false
					break
				}
			}
			if !fCheckResult {
				panic("this transaction all parent have spent...")
			}
			for _, preout := range entry.Tx.GetAllPreviousOut() {
				mempoolDuplicate.SpendCoin(&preout)
			}
			isCoinBase := entry.Tx.IsCoinBase()
			for i := 0; i < entry.Tx.GetOutsCount(); i++ {
				a := outpoint.OutPoint{entry.Tx.GetHash(), uint32(i)}
				mempoolDuplicate.AddCoin(&a, utxo.NewCoin(entry.Tx.GetTxOut(i), 1000000, isCoinBase))
			}
			stepsSinceLastRemove = 0
		}
	}

	for _, entry := range spentOut {
		txid := entry.Tx.GetHash()
		if e, ok := allEntry[txid]; !ok {
			panic("the transaction not exsit mempool. . .")
		} else {
			if e.Tx != entry.Tx {
				panic("mempool store the transaction is different with it's two struct . . .")
			}
		}
	}

	if pool.GetPoolAllTxSize() != checkTotal {
		panic("mempool have all transaction size state is incorrect ...")
	}
}

func FindTxInMempool(hash util.Hash) *mempool.TxEntry {
	pool := mempool.GetInstance()
	return pool.FindTx(hash)
}

