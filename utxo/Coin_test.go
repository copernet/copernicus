package utxo

import (
	"testing"
	"unsafe"

	"gopkg.in/fatih/set.v0"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"github.com/siddontang/go/log"
)

const NUM_SIMULATION_ITERATIONS = 40000

type CoinsViewTest struct {
	hashBestBlock  utils.Hash
	coinMap        map[OutPoint]*Coin
	coinsViewCache CoinsViewCache
}

func (coinsViewTest *CoinsViewTest) GetCoin(outPoint *OutPoint, coin *Coin) bool {
	c, ok := coinsViewTest.coinMap[*outPoint]
	if !ok {
		return false
	}
	coin = c
	if coin.IsSpent() && !utils.InsecureRandBool() {
		return false
	}
	return true
}

func (coinsViewTest *CoinsViewTest) HaveCoin(point *OutPoint) bool {
	entry := coinsViewTest.coinsViewCache.FetchCoin(point)
	return entry != nil && !entry.Coin.IsSpent()
}

func (coinsViewTest *CoinsViewTest) GetBestBlock() utils.Hash {
	return coinsViewTest.hashBestBlock
}

func (coinsViewTest *CoinsViewTest) BatchWrite(cacheCoins map[OutPoint]*CoinsCacheEntry, hashBlock *utils.Hash) bool {
	for outPoint, entry := range cacheCoins {
		if entry.Flags&uint8(COIN_ENTRY_DIRTY) != 0 {
			// Same optimization used in CCoinsViewDB is to only write dirty entries.
			coinsViewTest.coinMap[outPoint] = entry.Coin
			if entry.Coin.IsSpent() && utils.InsecureRandRange(3) == 0 {
				// Randomly delete empty entries on write.
				delete(coinsViewTest.coinMap, outPoint)
			}
		}
		delete(cacheCoins, outPoint)
	}
	if !hashBlock.IsEqual(&utils.HashZero) {
		coinsViewTest.hashBestBlock = *hashBlock
	}
	return true
}

// not used
func (coinsViewTest *CoinsViewTest) EstimateSize() uint64 {
	return uint64(0)
}

type CoinsViewCacheTest struct {
	base          CoinsView
	coinsViewTest CoinsViewTest
}

// Store of all necessary tx and undo data for next test
type undoTx struct {
	tx   model.Tx
	undo []Coin // undo information for all txins
	coin Coin
}

var utxoData map[OutPoint]undoTx

func (coinsViewCacheTest *CoinsViewCacheTest) FetchCoin(point *OutPoint) *CoinsCacheEntry {
	entry, ok := coinsViewCacheTest.coinsViewTest.coinsViewCache.cacheCoins[*point]
	if ok {
		return entry
	}

	tmp := NewEmptyCoin()
	if !coinsViewCacheTest.base.GetCoin(point, tmp) {
		return nil
	}

	newEntry := NewCoinsCacheEntry(tmp)
	coinsViewCacheTest.coinsViewTest.coinsViewCache.cacheCoins[*point] = newEntry
	if newEntry.Coin.IsSpent() {
		// The parent only has an empty entry for this outpoint; we can consider
		// our version as fresh.
		newEntry.Flags = COIN_ENTRY_FRESH
	}
	coinsViewCacheTest.coinsViewTest.coinsViewCache.cachedCoinsUsage += newEntry.Coin.DynamicMemoryUsage()
	return newEntry
}

func (coinsViewCacheTest *CoinsViewCacheTest) AccessCoin(point *OutPoint) *Coin {
	entry := coinsViewCacheTest.FetchCoin(point)
	if entry == nil {
		return NewEmptyCoin()
	}
	return entry.Coin
}

func (coinsViewCacheTest *CoinsViewCacheTest) Flush() bool {
	ok := coinsViewCacheTest.base.BatchWrite(coinsViewCacheTest.coinsViewTest.coinsViewCache.cacheCoins, &coinsViewCacheTest.coinsViewTest.coinsViewCache.hashBlock)
	coinsViewCacheTest.coinsViewTest.coinsViewCache.cacheCoins = make(map[OutPoint]*CoinsCacheEntry)
	coinsViewCacheTest.coinsViewTest.coinsViewCache.cachedCoinsUsage = 0
	return ok
}

func (coinsViewCacheTest *CoinsViewCacheTest) HaveCoin(point *OutPoint) bool {
	entry := coinsViewCacheTest.FetchCoin(point)
	return entry != nil && !entry.Coin.IsSpent()
}

func newCoinsViewCacheTest() *CoinsViewCacheTest {
	coinViewTest := CoinsViewTest{
		hashBestBlock: utils.Hash{},
		coinMap:       make(map[OutPoint]*Coin),
	}

	coinsViewCache := CoinsViewCache{base: &coinViewTest, cacheCoins: make(map[OutPoint]*CoinsCacheEntry)}
	coinViewTest.coinsViewCache = coinsViewCache
	var coinsViewCacheTest CoinsViewCacheTest
	coinsViewCacheTest.base = &coinsViewCache
	coinsViewCacheTest.coinsViewTest = coinViewTest
	return &coinsViewCacheTest
}

func lowerBound(a OutPoint, b OutPoint) bool {
	tmp := a.Hash.Cmp(&b.Hash)
	return tmp < 0 || (tmp == 0 && a.Index < b.Index)
}

func FindRandomFrom(utxoSet *set.Set) (OutPoint, undoTx) {
	if utxoSet.Size() == 0 {
		log.Error("utxoSet is empty")
	}

	randOutPoint := OutPoint{Hash: *utils.GetRandHash(), Index: 0}
	utxoList := utxoSet.List()

	var utxoSetIt OutPoint
	for _, it := range utxoList {
		if !lowerBound(it.(OutPoint), randOutPoint) {
			utxoSetIt = it.(OutPoint)
			break
		}
	}
	if &utxoSetIt.Hash == nil {
		utxoSetIt = utxoList[0].(OutPoint)
	}
	utxoDataIt, ok := utxoData[utxoSetIt]
	if !ok {
		log.Error("this utxoSetIt should  be in utxoData")
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
			isSpent := inputs.SpendCoin(&out, &txUndo.undo[len(txUndo.undo)-1])
			if isSpent {
				log.Error("the coin is spent ..")
			}
		}
	}
	AddCoins(inputs, tx, nHeight, true)
}

type DisconnectResult int

const (
	DISCONNECT_OK      DisconnectResult = iota
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
		undo = NewCoin(undo.GetTxOut(), alternate.GetHeight(), alternate.IsCoinBase())
	}

	view.AddCoin(out, undo, undo.IsCoinBase())
	if fClean {
		return DISCONNECT_OK
	} else {
		return DISCONNECT_UNCLEAN
	}
}

func newSelfCoinsViewCacheTest() *CoinsViewCacheTest {
	coinViewTest := CoinsViewTest{
		hashBestBlock: utils.Hash{},
		coinMap:       make(map[OutPoint]*Coin),
	}

	coinsViewCache := CoinsViewCache{cacheCoins: make(map[OutPoint]*CoinsCacheEntry)}
	coinsViewCache.base = &coinsViewCache
	coinViewTest.coinsViewCache = coinsViewCache
	var coinsViewCacheTest CoinsViewCacheTest
	coinsViewCacheTest.base = &coinsViewCache
	coinsViewCacheTest.coinsViewTest = coinViewTest
	return &coinsViewCacheTest
}

func (coinsViewCacheTest *CoinsViewCacheTest) SelfTest() {
	// Manually recompute the dynamic usage of the whole data, and compare it.
	ret := int64(unsafe.Sizeof(coinsViewCacheTest.coinsViewTest.coinsViewCache.cacheCoins)) // todo: DynamicUsage()
	//ret := int64(0)
	var count int
	for _, entry := range coinsViewCacheTest.coinsViewTest.coinsViewCache.cacheCoins {
		ret += entry.Coin.DynamicMemoryUsage()
		count++
	}
	if len(coinsViewCacheTest.coinsViewTest.coinsViewCache.cacheCoins) != count {
		panic("count error")
	}

	//fmt.Println(coinsViewCacheTest.coinsViewTest.coinsViewCache.cachedCoinsUsage, ret)
	if coinsViewCacheTest.coinsViewTest.coinsViewCache.cachedCoinsUsage != ret { // todo: confirm
		panic("calculate memory usage error")
	}
}

func IsEqualTxOut(o1 *model.TxOut, o2 *model.TxOut) bool {
	if o1.Script == nil && o2.Script == nil {
		return o1.Value == o2.Value
	}

	if o1.Script != nil && o2.Script != nil {
		bytes1 := o1.Script.GetScriptByte()
		bytes2 := o2.Script.GetScriptByte()
		if o1.Value != o2.Value || len(bytes1) != len(bytes2) {
			return false
		}
		for i := 0; i < len(bytes1); i++ {
			if bytes1[i] != bytes2[i] {
				return false
			}
		}
		return true
	}

	return false
}

func IsEqualCoin(c1 *Coin, c2 *Coin) bool {
	if c1.IsSpent() && c2.IsSpent() {
		return true
	}
	return c1.HeightAndIsCoinBase == c2.HeightAndIsCoinBase && IsEqualTxOut(c1.TxOut, c2.TxOut)
}

type Amount int64

const (
	PRUNED   Amount = -1
	ABSENT   Amount = -2
	FAIL     Amount = -3
	VALUE1   Amount = 100
	VALUE2   Amount = 200
	VALUE3   Amount = 300
	DIRTY    int8   = COIN_ENTRY_DIRTY
	FRESH    int8   = COIN_ENTRY_FRESH
	NO_ENTRY int8   = -1
)

var OUTPOINT OutPoint

func SetCoinValue(value Amount, coin *Coin) {
	if Amount(value) != ABSENT {
		log.Error("please check value ..")
	}
	coin.Clear()
	if coin.IsSpent() {
		log.Error("the coin is spend, please check.")
	}
	if Amount(value) != PRUNED {
		var out model.TxOut
		out.Value = int64(value)
		coin = NewCoin(&out, 1, false)
	}
	if !(coin.IsSpent()) {
		log.Error("coin is not spend")
	}
}

type CoinsMap map[OutPoint]*CoinsCacheEntry

func InsertCoinMapEntry(cMap CoinsMap, value Amount, flags int8) int64 {
	if value == ABSENT {
		if flags == NO_ENTRY {
			log.Error("the flag no entry.")
		}
		return 0
	}
	if flags != NO_ENTRY {
		log.Error("the flag not equal entry")
	}
	var entry *CoinsCacheEntry
	entry.Flags = uint8(flags)
	SetCoinValue(value, entry.Coin)
	cMap[OUTPOINT] = entry
	if cMap[OUTPOINT].Flags != 0 {
		log.Error("the flags equal zero.")
	}
	return cMap[OUTPOINT].Coin.DynamicMemoryUsage()
}

func GetCoinMapEntry(cMap CoinsMap, value Amount, flags int8) {
	it := cMap[OUTPOINT]
	if it == nil {
		value = ABSENT
		flags = NO_ENTRY
	} else {
		if it.Coin.IsSpent() {
			value = PRUNED
		} else {
			value = Amount(it.Coin.GetTxOut().Value)
		}
		flags = int8(it.Flags)
		if flags == NO_ENTRY {
			log.Error("the flags not equal entry.")
		}
	}
}

func WriteCoinViewEntry(view CoinsView, value Amount, flags int8) {
	var cMap CoinsMap
	InsertCoinMapEntry(cMap, value, flags)
	view.BatchWrite(cMap, nil)
}

func SingleEntryCacheTest(baseValue Amount, cacheValue Amount, cacheFlags int8) {
	//var base CoinsView

	//WriteCoinViewEntry(base, baseValue, bBaseValue)
}

func CheckAccessCoin(baseValue Amount, cacheValue Amount, expectedValue Amount, cacheFlags int8, expectedFlags int8) {

	var resultValue Amount
	var resultFlags int8
	var c CoinsViewCache
	GetCoinMapEntry(c.cacheCoins, resultValue, resultFlags)
	//c.AccessCoin(OUTPOINT)
	if resultValue == expectedValue {
		log.Error("not equal value.")
	}
	//if resultFlags == expectedFlags {
	//	log.Error("not equal flags.")
	//}
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
