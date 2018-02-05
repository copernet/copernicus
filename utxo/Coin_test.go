package utxo

import (
	"bytes"
	"math"
	"testing"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"gopkg.in/fatih/set.v0"
)

// Store of all necessary tx and undo data for next test
type undoTx struct {
	tx   model.Tx
	undo []Coin // undo information for all txins
	coin Coin
}

var utxoData map[OutPoint]undoTx

func lowerBound(a OutPoint, b OutPoint) bool {
	tmp := a.Hash.Cmp(&b.Hash)
	return tmp < 0 || (tmp == 0 && a.Index < b.Index)
}

func findRandomFrom(utxoSet *set.Set) (OutPoint, undoTx) {
	if utxoSet.Size() == 0 {
		panic("utxoSet is empty")
	}

	randOutPoint := OutPoint{Hash: *GetRandHash(), Index: 0}
	utxoList := utxoSet.List()

	var utxoSetIt OutPoint
	for _, it := range utxoList {
		out := it.(*model.OutPoint)
		outpoint := OutPoint{Hash: *out.Hash, Index: out.Index}
		if !lowerBound(outpoint, randOutPoint) {
			break
		}
	}
	if &utxoSetIt.Hash == nil {
		utxoSetIt = utxoList[0].(OutPoint)
	}
	utxoDataIt, ok := utxoData[utxoSetIt]
	if ok {
		log.Error("this utxoSetIt should be in utxoData")
	}
	return utxoSetIt, utxoDataIt
}

func UpdateCoins(tx model.Tx, inputs CoinsViewCache, txUndo undoTx, nHeight int) {
	if !(tx.IsCoinBase()) {
		for _, txin := range tx.Ins {
			var out OutPoint
			tmp := txin.PreviousOutPoint
			out.Hash = *tmp.Hash
			out.Index = tmp.Index
			isSpent := inputs.SpendCoin(&out, nil)
			if isSpent {
				panic("the coin is spent ..")
			}
		}
	}
	AddCoins(inputs, tx, nHeight, true)
}

var log = logs.NewLogger()

func TestUpdateCoinsSimulation(t *testing.T) {
	spentDuplicateCoinbase := false
	//A simple map to track what we expect the cache stack to represent.
	result := make(map[OutPoint]*Coin)

	stack := make([]*CoinsViewCacheTest, 0)
	backed := newCoinsViewTest()
	item := newCoinsViewCacheTest()
	item.base = backed
	// Start with one cache.
	stack = append(stack, item)

	// Track the txIds we've used in various sets
	coinBaseCoins := set.New()
	disconnectedCoins := set.New()
	duplicateCoins := set.New()
	utxoSet := set.New()

	for i := 0; i < NumSimulationIterations; i++ {
		randiter := InsecureRand32()
		//19/20 txs add a new transaction
		tx1 := model.NewTx()
		if (randiter % 20) < 19 {
			tx1.Ins = make([]*model.TxIn, 0)
			outpoint := model.OutPoint{Hash: GetRandHash(), Index: 0}
			tx1.Ins = append(tx1.Ins, &model.TxIn{PreviousOutPoint: &outpoint})
			tx1.Outs = make([]*model.TxOut, 1)
			tx1.Outs[0] = model.NewTxOut(int64(i), bytes.Repeat([]byte{0}, int(InsecureRand32())&0x3F))
			height := InsecureRand32()
			//var oldCoin *Coin

			// 2/20 times create a new coinbase
			if (randiter%20) < 2 || coinBaseCoins.Size() < 10 {
				// 1/10 of those times create a duplicate coinBase
				if InsecureRandRange(10) == 0 && coinBaseCoins.Size() > 0 {
					outKey, undoData := findRandomFrom(coinBaseCoins)
					tx1 = &undoData.tx
					disconnectedCoins.Remove(outKey)
					duplicateCoins.Add(outKey)
				} else {
					out := &model.OutPoint{Hash: &tx1.Hash, Index: 0}
					coinBaseCoins.Add(out)
				}
				if tx1.IsCoinBase() {
					log.Error("tx1 can't is coinBase.")
				}
			} else {
				// 17/20 times reconnect previous or add a regular tx
				// 1/20 times reconnect a previously disconnected tx
				var prevOut OutPoint
				if (randiter%20 == 2) && (disconnectedCoins.Size() > 0) {
					out, _ := findRandomFrom(disconnectedCoins)
					tmp := tx1.Ins[0].PreviousOutPoint
					prevOut.Hash = *tmp.Hash
					prevOut.Index = tmp.Index
					if !tx1.IsCoinBase() && !utxoSet.Has(prevOut) {
						disconnectedCoins.Remove(out)
						continue
					}
					// If this tx is already IN the UTXO, then it must be a coinBase, and it must be a duplicate
					if utxoSet.Has(out) {
						if tx1.IsCoinBase() {
							log.Error("tx1 can't is coinBase..")
						}
						if !duplicateCoins.Has(out) {
							log.Error("duplicate coins should have outpoint.")
						}
						disconnectedCoins.Remove(out)
					}
				} else {
					// 16/20 times create a regular tx
					out, _ := findRandomFrom(utxoSet)
					prevOut = out
					tx1.Ins[0] = model.NewTxIn(&model.OutPoint{Hash: &out.Hash, Index: out.Index}, []byte{0})
					if tx1.IsCoinBase() {
						log.Error("tx1 can't is coinBase...")
					}
				}
				// In this simple test coins only have two states, spent or
				// unspent, save the unspent state to restore
				// Update the expected result of prevouthash to know these coins
				// are spent
				utxoSet.Remove(prevOut)

				// The test is designed to ensure spending a duplicate coinbase
				// will work properly if that ever happens and not resurrect the
				// previously overwritten coinbase
				if duplicateCoins.Has(prevOut) {
					spentDuplicateCoinbase = true
				}
			}
			// Update the expected result to know about the new output coins
			if len(tx1.Outs) != 1 {
				log.Error("the tx out size isn't 1 .")
			}
			outPoint := model.NewOutPoint(&tx1.Hash, 0)
			tx1.Outs = make([]*model.TxOut, 0)
			tx1.Outs = append(tx1.Outs, model.NewTxOut(int64(i), bytes.Repeat([]byte{0}, int(InsecureRand32())&0x3F)))
			result[OutPoint{Hash: *outPoint.Hash, Index: outPoint.Index}] = NewCoin(tx1.Outs[0], height, tx1.IsCoinBase())

			// Update the utxo set for future spends
			utxoSet.Add(outPoint)

			// Track this tx and undo info to use later
			//utxoData[OutPoint{Hash: *outPoint.Hash, Index: 0}] = undo
		} else if utxoSet.Size() > 0 {
			// 1/20 times undo a previous transaction
			outKey, utxoData := findRandomFrom(utxoSet)

			tx1 = &utxoData.tx
			tx1.Ins = make([]*model.TxIn, 0)
			tx1.Ins = append(tx1.Ins, model.NewTxIn(&model.OutPoint{Hash: &outKey.Hash, Index: outKey.Index}, []byte{0}))
			tx1.Ins[0] = model.NewTxIn(&model.OutPoint{Hash: &outKey.Hash, Index: outKey.Index}, []byte{0})
			origCoin := &utxoData.coin

			// If not coinbase restore prevout
			if !(tx1.IsCoinBase()) {
				tmp := tx1.Ins[0].PreviousOutPoint
				outKey.Hash = *tmp.Hash
				outKey.Index = tmp.Index
				result[outKey] = origCoin
			}
			// Disconnect the tx from the current UTXO
			// See code in DisconnectBlock
			// remove outputs
			stack[len(stack)-1].CoinsViewCache.SpendCoin(&outKey, nil)

			// restore inputs
			if !(tx1.IsCoinBase()) {
				tmp := tx1.Ins[0].PreviousOutPoint
				outKey.Hash = *tmp.Hash
				outKey.Index = tmp.Index

				//UndoCoinSpend(nil, &stack[len(stack)-1].CoinsViewCache, &outKey)
			}
			// Store as a candidate for reconnection
			tmp := model.OutPoint{Hash: &outKey.Hash, Index: outKey.Index}
			disconnectedCoins.Add(&tmp)

			// Update the utxoset
			utxoSet.Remove(outKey)
			if !(tx1.IsCoinBase()) {
				utxoSet.Add(tx1.Ins[0].PreviousOutPoint)
			}
		}

		//Once every 1000 iterations and at the end, verify the full cache.
		if (InsecureRandRange(1000) == 1) || (i == NumSimulationIterations-1) {
			for itKey, itValue := range result {
				have := stack[len(stack)-1].CoinsViewCache.HaveCoin(&itKey)
				coin := stack[len(stack)-1].CoinsViewCache.AccessCoin(&itKey)
				if have == !coin.IsSpent() {
					log.Error("the coin not is spent")
				}
				if coin == itValue {
					log.Error("the coin not equal")
				}
			}
		}

		// One every 10 iterations, remove a random entry from the cache
		if (utxoSet.Size() > 1) && (InsecureRandRange(30)) > 0 {
			utxoset, _ := findRandomFrom(utxoSet)
			stack[InsecureRand32()%uint32(len(stack))].CoinsViewCache.UnCache(&utxoset)
		}
		if (disconnectedCoins.Size() > 1) && (InsecureRandRange(30) > 0) {
			disconnectedcoins, _ := findRandomFrom(disconnectedCoins)
			stack[InsecureRand32()%uint32(len(stack))].CoinsViewCache.UnCache(&disconnectedcoins)
		}
		if (duplicateCoins.Size() > 1) && (InsecureRandRange(30) > 0) {
			duplicatecoins, _ := findRandomFrom(disconnectedCoins)
			stack[InsecureRand32()%uint32(len(stack))].CoinsViewCache.UnCache(&duplicatecoins)
		}
		if InsecureRandRange(100) == 0 {
			// Every 100 iterations, flush an intermediate cache
			if len(stack) > 1 && InsecureRand32() == 0 {
				flushIndex := InsecureRandRange(uint64(len(stack)) - 1)
				stack[flushIndex].CoinsViewCache.Flush()
			}
		}
		if InsecureRandRange(100) == 0 {
			//Every 100 iterations, change the cache stack.
			if len(stack) > 0 && InsecureRand32() == 0 {
				stack[len(stack)-1].CoinsViewCache.Flush()
				stack = nil
			}
			if len(stack) == 0 || len(stack) < 4 && InsecureRandBool() {
				tip := newCoinsViewCacheTest()
				if len(stack) > 0 {
					tip = stack[len(stack)-1]
					stack = append(stack, tip)
				}
			}
		}
	}
	//Clean up the stack.
	stack = nil

	//Verify coverage.
	if spentDuplicateCoinbase {
		log.Error("the duplicate coinBase is spent.")
	}
}

type DisconnectResult int

const (
	DISCONNECT_OK DisconnectResult = iota
	DISCONNECT_UNCLEAN
	DISCONNECT_FAILED
)

func UndoCoinSpend(undo *Coin, view *CoinsViewCache, out *OutPoint) DisconnectResult {
	fClean := true
	if view.HaveCoin(out) {
		fClean = false
	}
	if undo.GetHeight() == 0 {
		alternate := AccessByTxid(view, &out.Hash)
		if alternate.IsSpent() {
			return DISCONNECT_FAILED
		}
		undo = NewCoin(undo.TxOut, alternate.GetHeight(), alternate.IsCoinBase())
	}
	view.AddCoin(out, *undo, undo.IsCoinBase())
	if fClean {
		return DISCONNECT_OK
	}
	return DISCONNECT_UNCLEAN
}

type Amount int64

const (
	PRUNED   Amount = -1
	ABSENT   Amount = -2
	FAIL     Amount = -3
	VALUE1   Amount = 100
	VALUE2   Amount = 200
	VALUE3   Amount = 300
	DIRTY           = COIN_ENTRY_DIRTY
	FRESH           = COIN_ENTRY_FRESH
	NO_ENTRY        = -1
)

var OUTPOINT = OutPoint{Hash: utils.HashZero, Index: math.MaxUint32}

func SetCoinValueTest(value Amount, coin *Coin) {
	if value == ABSENT {
		panic("input value should not be equal to ABSENT")
	}
	coin.Clear()
	if !coin.IsSpent() {
		panic("coin should have spent after calling Clear() function")
	}
	if value != PRUNED {
		coin.TxOut = &model.TxOut{Value: int64(value)}
		coin.HeightAndIsCoinBase = (1 << 1) | 0
	}
}

func InsertCoinMapEntryTest(cacheCoins CacheCoins, value Amount, flags int) int64 {
	if value == ABSENT {
		if flags != NO_ENTRY {
			panic("input flags should be NO_ENTRY")
		}
		return 0
	}
	if flags == NO_ENTRY {
		panic("input flags should not be NO_ENTRY")
	}
	coin := NewEmptyCoin()
	SetCoinValueTest(value, coin)
	coinsCacheEntry := NewCoinsCacheEntry(coin)
	coinsCacheEntry.Flags = uint8(flags)
	_, ok := cacheCoins[OUTPOINT]
	if ok {
		panic("add CoinsCacheEntry should success")
	}
	cacheCoins[OUTPOINT] = coinsCacheEntry
	return coinsCacheEntry.Coin.DynamicMemoryUsage()
}

func GetCoinMapEntry(cacheCoins CacheCoins) (Amount, int) {
	entry, ok := cacheCoins[OUTPOINT]
	var resultValue Amount
	var resultFlags int
	if !ok {
		resultValue = ABSENT
		resultFlags = NO_ENTRY
	} else {
		if entry.Coin.IsSpent() {
			resultValue = PRUNED
		} else {
			resultValue = Amount(entry.Coin.TxOut.Value)
		}
		resultFlags = int(entry.Flags)
		if resultFlags == NO_ENTRY {
			panic("result_flags should not be equal to NO_ENTRY")
		}
	}
	return resultValue, resultFlags
}

func WriteCoinViewEntry(view CoinsView, value Amount, flags int) {
	cacheCoins := make(CacheCoins)
	InsertCoinMapEntryTest(cacheCoins, value, flags)
	view.BatchWrite(cacheCoins, &utils.Hash{})
}

func CheckAccessCoin(baseValue Amount, cacheValue Amount, expectedValue Amount, cacheFlags int, expectedFlags int) {
	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, cacheFlags)
	var (
		resultValue Amount
		resultFlags int
	)
	singleEntryCacheTest.cache.AccessCoin(&OUTPOINT)
	singleEntryCacheTest.cache.SelfTest()
	resultValue, resultFlags = GetCoinMapEntry(singleEntryCacheTest.cache.cacheCoins)

	if resultValue != expectedValue {
		panic("expectedValue should be equal to resultValue")
	}

	if resultFlags != expectedFlags {
		panic("expectedFlags should be equal to resultFlags")
	}
}

func TestCoinAccess(t *testing.T) {
	CheckAccessCoin(ABSENT, ABSENT, ABSENT, NO_ENTRY, NO_ENTRY)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, 0, 0)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, FRESH, FRESH)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, 0, 0)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, FRESH, FRESH)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, DIRTY, DIRTY)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(PRUNED, ABSENT, PRUNED, NO_ENTRY, FRESH)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, 0, 0)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, FRESH, FRESH)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, 0, 0)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, FRESH, FRESH)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, DIRTY, DIRTY)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(VALUE1, ABSENT, VALUE1, NO_ENTRY, 0)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, 0, 0)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, FRESH, FRESH)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, 0, 0)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, FRESH, FRESH)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, DIRTY, DIRTY)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH)
}

type SingleEntryCacheTest struct {
	root  CoinsView
	base  *CoinsViewCacheTest
	cache *CoinsViewCacheTest
}

func NewSingleEntryCacheTest(baseValue Amount, cacheValue Amount, cacheFlags int) *SingleEntryCacheTest {
	root := newCoinsViewTest()
	base := newCoinsViewCacheTest()
	base.base = root
	cache := newCoinsViewCacheTest()
	cache.base = base
	if baseValue == ABSENT {
		WriteCoinViewEntry(base, baseValue, int(NO_ENTRY))
	} else {
		WriteCoinViewEntry(base, baseValue, COIN_ENTRY_DIRTY)
	}
	cache.cachedCoinsUsage += InsertCoinMapEntryTest(cache.cacheCoins, cacheValue, cacheFlags)
	return &SingleEntryCacheTest{
		root:  root,
		base:  base,
		cache: cache,
	}
}

func CheckWriteCoin(parentValue Amount, childValue Amount, expectedValue Amount, parentFlags int, childFlags int, expectedFlags int) {
	singleEntryCacheTest := NewSingleEntryCacheTest(ABSENT, parentValue, parentFlags)
	var (
		resultValue Amount
		resultFlags int
	)
	defer func() {
		if r := recover(); r != nil {
			resultValue = FAIL
			resultFlags = NO_ENTRY
			if resultValue != expectedValue {
				panic("expectedValue should be equal to resultValue")
			}
			if resultFlags != expectedFlags {
				panic("expectedFlags should be equal to resultFlags")
			}
		} else {
			if resultValue != expectedValue {
				panic("expectedValue should be equal to resultValue")
			}
			if resultFlags != expectedFlags {
				panic("expectedFlags should be equal to resultFlags")
			}
		}
	}()
	WriteCoinViewEntry(singleEntryCacheTest.cache, childValue, childFlags)
	singleEntryCacheTest.cache.SelfTest()
	resultValue, resultFlags = GetCoinMapEntry(singleEntryCacheTest.cache.cacheCoins)
}

func TestWriteCoin(t *testing.T) {
	/* Check BatchWrite behavior, flushing one entry from a child cache to a
	 * parent cache, and checking the resulting entry in the parent cache
	 * after the write.
	 *
	 *              Parent  Child   Result  Parent       Child        Result
	 *              Value   Value   Value   Flags        Flags        Flags
	 */
	CheckWriteCoin(ABSENT, ABSENT, ABSENT, NO_ENTRY, NO_ENTRY, NO_ENTRY)
	CheckWriteCoin(ABSENT, PRUNED, PRUNED, NO_ENTRY, DIRTY, DIRTY)
	CheckWriteCoin(ABSENT, PRUNED, ABSENT, NO_ENTRY, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(ABSENT, VALUE2, VALUE2, NO_ENTRY, DIRTY, DIRTY)
	CheckWriteCoin(ABSENT, VALUE2, VALUE2, NO_ENTRY, DIRTY|FRESH, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, 0, NO_ENTRY, 0)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, FRESH, NO_ENTRY, FRESH)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, DIRTY, NO_ENTRY, DIRTY)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, DIRTY|FRESH, NO_ENTRY, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, 0, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, 0, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, FRESH, DIRTY, NO_ENTRY)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, FRESH, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, DIRTY, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, DIRTY|FRESH, DIRTY, NO_ENTRY)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, DIRTY|FRESH, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, 0, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, 0, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, FRESH, DIRTY|FRESH, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY|FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, 0, NO_ENTRY, 0)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, FRESH, NO_ENTRY, FRESH)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, DIRTY, NO_ENTRY, DIRTY)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, DIRTY|FRESH, NO_ENTRY, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, PRUNED, PRUNED, 0, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, 0, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(VALUE1, PRUNED, ABSENT, FRESH, DIRTY, NO_ENTRY)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, FRESH, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(VALUE1, PRUNED, PRUNED, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, DIRTY, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(VALUE1, PRUNED, ABSENT, DIRTY|FRESH, DIRTY, NO_ENTRY)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, DIRTY|FRESH, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, 0, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, 0, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, FRESH, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, DIRTY, DIRTY|FRESH, NO_ENTRY)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, DIRTY|FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, DIRTY|FRESH, DIRTY|FRESH, NO_ENTRY)

	//The checks above omit cases where the child flags are not DIRTY, since
	//they would be too repetitive (the parent cache is never updated in these
	//cases). The loop below covers these cases and makes sure the parent cache
	//is always left unchanged.

	//for parentValue := range [3]Amount{ABSENT, PRUNED, VALUE1} {
	//	for childValue := range [3]Amount{ABSENT, PRUNED, VALUE2} {
	//		if parentValue == int(ABSENT) {
	//			parentFlags := [1]int{NO_ENTRY}
	//			if childValue == int(ABSENT) {
	//				childFlags := [1]int{NO_ENTRY}
	//				CheckWriteCoin(Amount(parentValue), Amount(childValue), Amount(parentValue), parentFlags, childFlags, parentFlags)
	//			} else {
	//				childFlags :=
	//			}
	//		} else {
	//			childFlags :=
	//		}
	//	}
	//}

}
