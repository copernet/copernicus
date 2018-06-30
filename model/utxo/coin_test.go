package utxo

import (
	"math"
	"testing"

	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/script"
)

// test whether get the expected item by OutPoint struct with a pointer
// in it or not
func TestGetCoinByPointerOrValue(t *testing.T) {
	type OutPoint struct {
		Hash  *util.Hash
		Index int
	}

	map1 := make(map[outpoint.OutPoint]*Coin)
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}
	// store one item
	map1[outpoint1] = &Coin{}
	hash11 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	if _, ok := map1[outpoint.OutPoint{Hash: *hash11, Index: 0}]; !ok {
		t.Error("the key without pointer should point to a exist value")
	}

	map2 := make(map[OutPoint]*Coin)
	hash2 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint2 := OutPoint{Hash: hash2, Index: 0}
	//store one item
	map2[outpoint2] = &Coin{}
	hash22 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	if _, ok := map2[OutPoint{Hash: hash22, Index: 0}]; ok {
		t.Error("there should not be a item as the different pointer value in the struct")
	}
}

const (
	PRUNED  amount.Amount = -1
	ABSENT  amount.Amount = -2
	FAIL    amount.Amount = -3
	VALUE1  amount.Amount = 100
	VALUE2  amount.Amount = 200
	VALUE3  amount.Amount = 300
	DIRTY                 = 1
	FRESH                 = 2
	NoEntry               = -1
)

var OUTPOINT = outpoint.OutPoint{Hash: util.HashZero, Index: math.MaxUint32}

func GetCoinMapEntry(cache CoinsLruCache) (amount.Amount, int) {
	entry := cache.dirtyCoins
	var resultValue amount.Amount
	var resultFlags int
	if entry == nil {
		resultValue = ABSENT
		resultFlags = NoEntry
	} else {
		for _, coin := range entry {

			if coin.IsSpent() {
				resultValue = PRUNED
			} else {
				resultValue = amount.Amount(coin.txOut.GetValue())
			}
			resultFlags = coin.dirty
			if resultFlags == NoEntry {
				panic("result_flags should not be equal to NO_ENTRY")
			}
		}
	}
	return resultValue, resultFlags
}

func CheckSpendCoin(baseValue amount.Amount, cacheValue amount.Amount, expectedValue amount.Amount, cacheFlags int, expectedFlags int) {
	//singleEntryCacheTest := GetUtxoLruCacheInstance() //baseValue, cacheValue, cacheFlags
	coinMap := NewEmptyCoinsMap()
	coinMap.SpendCoin(&OUTPOINT)
	coinMap.SpendGlobalCoin(&OUTPOINT)

	resultValue := GetCoinMapEntry()
	if expectedValue != resultValue {
		panic("expectedValue should be equal to resultValue")
	}
}

func TestCoinSpeed(t *testing.T) {
	/**
	 * Check SpendCoin behavior, requesting a coin from a cache view layered on
	 * top of a base view, spending, and then checking the resulting entry in
	 * the cache after the modification.
	 *
	 *              Base    	Cache   	Result  		Cache        Result
	 *              Value   	Value   	Value   		Flags        Flags
	 */

	CheckSpendCoin(ABSENT, ABSENT, ABSENT, NoEntry, NoEntry)
	CheckSpendCoin(ABSENT, PRUNED, PRUNED, 0, DIRTY)
	CheckSpendCoin(ABSENT, PRUNED, ABSENT, FRESH, NoEntry)
	CheckSpendCoin(ABSENT, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckSpendCoin(ABSENT, PRUNED, ABSENT, DIRTY|FRESH, NoEntry)
	CheckSpendCoin(ABSENT, VALUE2, PRUNED, 0, DIRTY)
	CheckSpendCoin(ABSENT, VALUE2, ABSENT, FRESH, NoEntry)
	CheckSpendCoin(ABSENT, VALUE2, PRUNED, DIRTY, DIRTY)
	CheckSpendCoin(ABSENT, VALUE2, ABSENT, DIRTY|FRESH, NoEntry)
	CheckSpendCoin(PRUNED, ABSENT, ABSENT, NoEntry, NoEntry)
	CheckSpendCoin(PRUNED, PRUNED, PRUNED, 0, DIRTY)
	CheckSpendCoin(PRUNED, PRUNED, ABSENT, FRESH, NoEntry)
	CheckSpendCoin(PRUNED, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckSpendCoin(PRUNED, PRUNED, ABSENT, DIRTY|FRESH, NoEntry)
	CheckSpendCoin(PRUNED, VALUE2, PRUNED, 0, DIRTY)
	CheckSpendCoin(PRUNED, VALUE2, ABSENT, FRESH, NoEntry)
	CheckSpendCoin(PRUNED, VALUE2, PRUNED, DIRTY, DIRTY)
	CheckSpendCoin(PRUNED, VALUE2, ABSENT, DIRTY|FRESH, NoEntry)
	CheckSpendCoin(VALUE1, ABSENT, PRUNED, NoEntry, DIRTY)
	CheckSpendCoin(VALUE1, PRUNED, PRUNED, 0, DIRTY)
	CheckSpendCoin(VALUE1, PRUNED, ABSENT, FRESH, NoEntry)
	CheckSpendCoin(VALUE1, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckSpendCoin(VALUE1, PRUNED, ABSENT, DIRTY|FRESH, NoEntry)
	CheckSpendCoin(VALUE1, VALUE2, PRUNED, 0, DIRTY)
	CheckSpendCoin(VALUE1, VALUE2, ABSENT, FRESH, NoEntry)
	CheckSpendCoin(VALUE1, VALUE2, PRUNED, DIRTY, DIRTY)
	CheckSpendCoin(VALUE1, VALUE2, ABSENT, DIRTY|FRESH, NoEntry)
}

func CheckAddCoinBase(baseValue amount.Amount, cacheValue amount.Amount, modifyValue amount.Amount,
	expectedValue amount.Amount, cacheFlags int, expectedFlags int, isCoinbase bool) {

	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, int(cacheFlags))

	var resultValue amount.Amount
	var resultFlags int
	defer func() {
		if r := recover(); r != nil {
			resultValue = FAIL
			resultFlags = NoEntry
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

	txOut := txout.NewTxOut(modifyValue, script.NewEmptyScript())
	coin := NewCoin(txOut, 1, isCoinbase)
	singleEntryCacheTest.cache.AddCoin(&OUTPOINT, *coin, isCoinbase)
	singleEntryCacheTest.cache.SelfTest()
	resultValue, resultFlags = GetCoinMapEntry(singleEntryCacheTest.cache.CacheCoins)
}

func CheckAddCoin(cacheValue amount.Amount, modifyValue amount.Amount, expectedValue amount.Amount, cacheFlags int, expectedFlags int, isCoinbase bool) {
	for _, arg := range [3]amount.Amount{ABSENT, PRUNED, VALUE1} {
		CheckAddCoinBase(arg, cacheValue, modifyValue, expectedValue, cacheFlags, expectedFlags, isCoinbase)
	}
}

func TestCoinAdd(t *testing.T) {
	/**
	 * Check AddCoin behavior, requesting a new coin from a cache view, writing
	 * a modification to the coin, and then checking the resulting entry in the
	 * cache after the modification. Verify behavior with the with the AddCoin
	 * potential_overwrite argument set to false, and to true.
	 *
	 * Cache   Write   Result  Cache        Result       potential_overwrite
	 * Value   Value   Value   Flags        Flags
	 */
	CheckAddCoin(ABSENT, VALUE3, VALUE3, NoEntry, DIRTY|FRESH, false)
	CheckAddCoin(ABSENT, VALUE3, VALUE3, NoEntry, DIRTY, true)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, 0, DIRTY|FRESH, false)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, 0, DIRTY, true)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, FRESH, DIRTY|FRESH, false)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, FRESH, DIRTY|FRESH, true)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, DIRTY, DIRTY, false)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, DIRTY, DIRTY, true)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, DIRTY|FRESH, DIRTY|FRESH, false)
	CheckAddCoin(PRUNED, VALUE3, VALUE3, DIRTY|FRESH, DIRTY|FRESH, true)
	CheckAddCoin(VALUE2, VALUE3, FAIL, 0, NoEntry, false)
	CheckAddCoin(VALUE2, VALUE3, VALUE3, 0, DIRTY, true)
	CheckAddCoin(VALUE2, VALUE3, FAIL, FRESH, NoEntry, false)
	CheckAddCoin(VALUE2, VALUE3, VALUE3, FRESH, DIRTY|FRESH, true)
	CheckAddCoin(VALUE2, VALUE3, FAIL, DIRTY, NoEntry, false)
	CheckAddCoin(VALUE2, VALUE3, VALUE3, DIRTY, DIRTY, true)
	CheckAddCoin(VALUE2, VALUE3, FAIL, DIRTY|FRESH, NoEntry, false)
	CheckAddCoin(VALUE2, VALUE3, VALUE3, DIRTY|FRESH, DIRTY|FRESH, true)
}

func CheckAccessCoin(baseValue amount.Amount, cacheValue amount.Amount, expectedValue amount.Amount, cacheFlags int, expectedFlags int) {
	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, cacheFlags)
	var (
		resultValue amount.Amount
		resultFlags int
	)
	singleEntryCacheTest.cache.AccessCoin(&OUTPOINT)
	singleEntryCacheTest.cache.SelfTest()
	resultValue, resultFlags = GetCoinMapEntry(singleEntryCacheTest.cache.CacheCoins)

	if resultValue != expectedValue {
		panic("expectedValue should be equal to resultValue")
	}

	if resultFlags != expectedFlags {
		panic("expectedFlags should be equal to resultFlags")
	}
}

func TestCoinAccess(t *testing.T) {
	CheckAccessCoin(ABSENT, ABSENT, ABSENT, NoEntry, NoEntry)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, 0, 0)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, FRESH, FRESH)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckAccessCoin(ABSENT, PRUNED, PRUNED, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, 0, 0)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, FRESH, FRESH)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, DIRTY, DIRTY)
	CheckAccessCoin(ABSENT, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(PRUNED, ABSENT, PRUNED, NoEntry, FRESH)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, 0, 0)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, FRESH, FRESH)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckAccessCoin(PRUNED, PRUNED, PRUNED, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, 0, 0)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, FRESH, FRESH)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, DIRTY, DIRTY)
	CheckAccessCoin(PRUNED, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(VALUE1, ABSENT, VALUE1, NoEntry, 0)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, 0, 0)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, FRESH, FRESH)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, DIRTY, DIRTY)
	CheckAccessCoin(VALUE1, PRUNED, PRUNED, DIRTY|FRESH, DIRTY|FRESH)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, 0, 0)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, FRESH, FRESH)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, DIRTY, DIRTY)
	CheckAccessCoin(VALUE1, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH)
}
