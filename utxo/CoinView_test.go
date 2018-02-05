package utxo

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

const NumSimulationIterations = 40000

type CoinsViewCacheTest struct {
	CoinsViewCache
}

func newCoinsViewCacheTest() *CoinsViewCacheTest {
	return &CoinsViewCacheTest{
		CoinsViewCache: CoinsViewCache{
			cacheCoins: make(CacheCoins),
		},
	}
}

// backed store
type CoinsViewTest struct {
	hashBestBlock utils.Hash
	coinMap       map[OutPoint]*Coin
}

func newCoinsViewTest() *CoinsViewTest {
	return &CoinsViewTest{
		coinMap: make(map[OutPoint]*Coin),
	}
}

func (coinsViewTest *CoinsViewTest) GetCoin(outPoint *OutPoint, coin *Coin) bool {
	c, ok := coinsViewTest.coinMap[*outPoint]
	if !ok {
		return false
	}
	tmp := DeepCopyCoin(c)
	coin.TxOut = tmp.TxOut
	coin.HeightAndIsCoinBase = tmp.HeightAndIsCoinBase
	if coin.IsSpent() && InsecureRandBool() {
		return false
	}
	return true
}

func (coinsViewTest *CoinsViewTest) HaveCoin(point *OutPoint) bool {
	var coin *Coin
	return coinsViewTest.GetCoin(point, coin)
}

func (coinsViewTest *CoinsViewTest) GetBestBlock() utils.Hash {
	return coinsViewTest.hashBestBlock
}
func (coinsViewTest *CoinsViewTest) EstimateSize() uint64 {
	return 0
}

func (coinsViewTest *CoinsViewTest) BatchWrite(cacheCoins CacheCoins, hashBlock *utils.Hash) bool {
	for outPoint, entry := range cacheCoins {
		if entry.Flags&COIN_ENTRY_DIRTY != 0 {
			// Same optimization used in CCoinsViewDB is to only write dirty entries.
			tmp := DeepCopyCoin(entry.Coin)
			coinsViewTest.coinMap[outPoint] = &tmp
			if entry.Coin.IsSpent() && InsecureRand32()%3 == 0 {
				// Randomly delete empty entries on write.
				delete(coinsViewTest.coinMap, outPoint)
			}
		}
	}
	cacheCoins = make(CacheCoins)
	if !hashBlock.IsNull() {
		coinsViewTest.hashBestBlock = *hashBlock
	}

	return true
}

func (coinsViewCacheTest *CoinsViewCacheTest) SelfTest() {
	// Manually recompute the dynamic usage of the whole data, and compare it.
	var ret int64
	var count int
	for _, entry := range coinsViewCacheTest.cacheCoins {
		ret += entry.Coin.DynamicMemoryUsage()
		count++
	}
	if len(coinsViewCacheTest.cacheCoins) != count {
		panic("count error")
	}

	if coinsViewCacheTest.cachedCoinsUsage != ret {
		panic("calculate memory usage error")
	}
}

func IsEqualCoin(c1 *Coin, c2 *Coin) bool {
	if c1.IsSpent() && c2.IsSpent() {
		return true
	}
	return c1.HeightAndIsCoinBase == c2.HeightAndIsCoinBase && IsEqualTxOut(c1.TxOut, c2.TxOut)
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

// This is a large randomized insert/remove simulation test on a variable-size
// stack of caches on top of CCoinsViewTest.
//
// It will randomly create/update/delete Coin entries to a tip of caches, with
// txids picked from a limited list of random 256-bit hashes. Occasionally, a
// new tip is added to the stack of caches, or the tip is flushed and removed.
//
// During the process, booleans are kept to make sure that the randomized
// operation hits all branches.
func TestCoinsCacheSimulation(t *testing.T) {
	// Various coverage trackers.
	removedAllCaches := false
	reached4Caches := false
	addedAnEntry := false
	addedAnUnspendableEntry := false
	removedAnEntry := false
	updatedAnEntry := false
	foundAnEntry := false
	missedAnEntry := false
	unCachedAnEntry := false

	// A simple map to track what we expect the cache stack to represent.
	result := make(map[OutPoint]*Coin)

	// The cache stack.
	// A stack of CCoinsViewCaches on top.
	stack := make([]*CoinsViewCacheTest, 0)
	// A backed store instance
	backed := newCoinsViewTest()
	// A stack of CCoinsViewCaches on top.
	item := newCoinsViewCacheTest()
	item.base = backed
	// Start with one cache.
	stack = append(stack, item)

	// Use a limited set of random transaction ids, so we do test overwriting entries.
	var txids [NumSimulationIterations / 8]utils.Hash
	for i := 0; i < NumSimulationIterations/8; i++ {
		txids[i] = *GetRandHash()
	}

	for i := 0; i < NumSimulationIterations; i++ {
		// Do a random modification.
		{
			randomNum := InsecureRandRange(uint64(len(txids) - 1))
			// txid we're going to modify in this iteration.
			txid := txids[randomNum]
			coin, ok := result[OutPoint{Hash: txid, Index: 0}]

			if !ok {
				coin = NewEmptyCoin()
				result[OutPoint{Hash: txid, Index: 0}] = coin
			}

			randNum := InsecureRandRange(50)
			var entry *Coin
			if randNum == 0 {
				entry = AccessByTxid(&stack[len(stack)-1].CoinsViewCache, &txid)
			} else {
				entry = stack[len(stack)-1].AccessCoin(&OutPoint{Hash: txid, Index: 0})
			}

			if !IsEqualCoin(entry, coin) {
				t.Error("the coin should be equal to entry from cacheCoins or coinMap")
			}

			if InsecureRandRange(5) == 0 || coin.IsSpent() {
				var newTxOut model.TxOut
				newTxOut.Value = int64(InsecureRand32())
				if InsecureRandRange(16) == 0 && coin.IsSpent() {
					newTxOut.Script = model.NewScriptRaw(bytes.Repeat([]byte{byte(model.OP_RETURN)}, int(InsecureRandBits(6)+1)))
					if !newTxOut.Script.IsUnspendable() {
						t.Error("error IsUnspendable")
					}
					addedAnUnspendableEntry = true
				} else {
					// Random sizes so we can test memory usage accounting
					randomBytes := bytes.Repeat([]byte{0}, int(InsecureRandBits(6)+1))
					newTxOut.Script = model.NewScriptRaw(randomBytes)
					if coin.IsSpent() {
						addedAnEntry = true
					} else {
						updatedAnEntry = true
					}
					*result[OutPoint{Hash: txid, Index: 0}] = DeepCopyCoin(&Coin{TxOut: &newTxOut, HeightAndIsCoinBase: 2})
				}
				newCoin := Coin{TxOut: &newTxOut, HeightAndIsCoinBase: 2}
				newnewCoin := DeepCopyCoin(&newCoin)
				stack[len(stack)-1].AddCoin(&OutPoint{Hash: txid, Index: 0}, newnewCoin, !coin.IsSpent() || (InsecureRand32()&1 != 0))
			} else {
				removedAnEntry = true
				result[OutPoint{Hash: txid, Index: 0}].Clear()
				stack[len(stack)-1].SpendCoin(&OutPoint{Hash: txid, Index: 0}, nil)
			}
		}

		// One every 10 iterations, remove a random entry from the cache
		if InsecureRandRange(11) != 0 {
			cacheID := int(InsecureRand32()) % (len(stack))
			hashID := int(InsecureRand32()) % len(txids)
			out := OutPoint{Hash: txids[hashID], Index: 0}
			stack[cacheID].UnCache(&out)
			if !stack[cacheID].HaveCoinInCache(&out) {
				unCachedAnEntry = true
			}
		}

		// Once every 1000 iterations and at the end, verify the full cache.
		//if InsecureRandRange(2) == 1 || i == NumSimulationIterations-1 {
		if i == 200 || i == NumSimulationIterations-1 {
			for out, entry := range result {
				have := stack[len(stack)-1].HaveCoin(&out)
				coin := stack[len(stack)-1].AccessCoin(&out)
				if have == coin.IsSpent() {
					t.Error("the coin should be different from have in IsSpent")
				}

				if !IsEqualCoin(coin, entry) {
					t.Error("the coin should be equal to entry from cacheCoins or coinMap")
				}
				if coin.IsSpent() {
					missedAnEntry = true
				} else {
					if !stack[len(stack)-1].HaveCoinInCache(&out) {
						t.Error("error HaveCoinInCache")
					}
					foundAnEntry = true
				}
			}
			for _, test := range stack {
				test.SelfTest()
			}
		}

		// Every 100 iterations, flush an intermediate cache
		if InsecureRandRange(100) == 1000 {
			// Every 100 iterations, flush an intermediate cache
			if len(stack) > 1 && InsecureRandBool() {
				flushIndex := InsecureRandRange(uint64(len(stack) - 1))
				for out, item := range stack[0].cacheCoins {
					fmt.Println(out.Hash.ToString(), item.Coin.TxOut.Value, item.Coin.HeightAndIsCoinBase, item.Flags)
				}
				stack[flushIndex].Flush()
			}
		}

		if InsecureRandRange(100) == 0 {
			// Every 100 iterations, change the cache stack.
			length := len(stack)
			if length > 0 && InsecureRandBool() {
				//Remove the top cache
				stack[len(stack)-1].Flush()
				stack = stack[:length-1]
			}

			if len(stack) == 0 || len(stack) < 4 && InsecureRandBool() {
				//Add a new cache
				tip := newCoinsViewCacheTest()
				if len(stack) > 0 {
					tip.base = stack[len(stack)-1]
				} else {
					tip.base = backed
					removedAllCaches = true
				}

				stack = append(stack, tip)
				if len(stack) == 4 {
					reached4Caches = true
				}
			}
		}
	}

	// Clean up the stack.
	stack = nil

	// Verify coverage.
	if !removedAllCaches {
		t.Error("removedAllCaches should be true")
	}
	if !reached4Caches {
		t.Error("reached4Caches should be true")
	}
	if !addedAnEntry {
		t.Error("addedAnEntry should be true")
	}
	if !addedAnUnspendableEntry {
		t.Error("addedAnUnspendableEntry should be true")
	}
	if !removedAnEntry {
		t.Error("removedAnEntry should be true")
	}
	if !updatedAnEntry {
		t.Error("updatedAnEntry should be true")
	}
	if !foundAnEntry {
		t.Error("foundAnEntry should be true")
	}
	if !missedAnEntry {
		t.Error("missedAnEntry should be true")
	}
	if !unCachedAnEntry {
		t.Error("uncachedAnEntry should be true")
	}
}

var TestOutPoint = OutPoint{Hash: utils.HashZero, Index: math.MaxUint32}

const (
	TEST_PRUNED   btcutil.Amount = -1
	TEST_ABSENT   btcutil.Amount = -2
	TEST_FAIL     btcutil.Amount = -3
	TEST_VALUE1   btcutil.Amount = 100
	TEST_VALUE2   btcutil.Amount = 200
	TEST_VALUE3   btcutil.Amount = 300
	TEST_NO_ENTRY int            = -1
)

type SingleEntryCacheTest struct {
	root  CoinsView
	base  *CoinsViewCacheTest
	cache *CoinsViewCacheTest
}

func NewSingleEntryCacheTest(baseValue btcutil.Amount, cacheValue btcutil.Amount, cacheFlags int) *SingleEntryCacheTest {
	root := newCoinsViewTest()
	base := newCoinsViewCacheTest()
	base.base = root
	cache := newCoinsViewCacheTest()
	cache.base = base
	if baseValue == TEST_ABSENT {
		WriteCoinViewEntryTest(base, baseValue, TEST_NO_ENTRY)
	} else {
		WriteCoinViewEntryTest(base, baseValue, COIN_ENTRY_DIRTY)
	}
	cache.cachedCoinsUsage += InsertCoinMapEntryTest(cache.cacheCoins, cacheValue, cacheFlags)
	return &SingleEntryCacheTest{
		root:  root,
		base:  base,
		cache: cache,
	}
}

func WriteCoinViewEntryTest(view CoinsView, value btcutil.Amount, flags int) {
	cacheCoins := make(CacheCoins)
	InsertCoinMapEntryTest(cacheCoins, value, flags)
	view.BatchWrite(cacheCoins, &utils.Hash{})
}

func InsertCoinMapEntryTest(cacheCoins CacheCoins, value btcutil.Amount, flags int) int64 {
	if value == TEST_ABSENT {
		if flags != TEST_NO_ENTRY {
			panic("input flags should be NO_ENTRY")
		}
		return 0
	}
	if flags == TEST_NO_ENTRY {
		panic("input flags should not be NO_ENTRY")
	}
	coin := NewEmptyCoin()
	SetCoinValueTest(value, coin)
	coinsCacheEntry := NewCoinsCacheEntry(coin)
	coinsCacheEntry.Flags = uint8(flags)
	_, ok := cacheCoins[TestOutPoint]
	if ok {
		panic("add CoinsCacheEntry should success")
	}
	cacheCoins[TestOutPoint] = coinsCacheEntry
	return coinsCacheEntry.Coin.DynamicMemoryUsage()
}

func SetCoinValueTest(value btcutil.Amount, coin *Coin) {
	if value == TEST_ABSENT {
		panic("input value should not be equal to ABSENT")
	}
	coin.Clear()
	if !coin.IsSpent() {
		panic("coin should have spent after calling Clear() function")
	}
	if value != TEST_PRUNED {
		coin.TxOut = &model.TxOut{Value: int64(value)}
		coin.HeightAndIsCoinBase = (1 << 1) | 0
	}
}

func GetCoinMapEntryTest(cacheCoins CacheCoins) (btcutil.Amount, int) {
	entry, ok := cacheCoins[TestOutPoint]
	var resultValue btcutil.Amount
	var resultFlags int
	if !ok {
		resultValue = TEST_ABSENT
		resultFlags = TEST_NO_ENTRY
	} else {
		if entry.Coin.IsSpent() {
			resultValue = TEST_PRUNED
		} else {
			resultValue = btcutil.Amount(entry.Coin.TxOut.Value)
		}
		resultFlags = int(entry.Flags)
		if resultFlags == TEST_NO_ENTRY {
			panic("result_flags should not be equal to NO_ENTRY")
		}
	}
	return resultValue, resultFlags
}

func CheckSpendCoin(baseValue btcutil.Amount, cacheValue btcutil.Amount, expectedValue btcutil.Amount, cacheFlags int, expectedFlags int) {
	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, int(cacheFlags))
	singleEntryCacheTest.cache.SpendCoin(&TestOutPoint, nil)
	singleEntryCacheTest.cache.SelfTest()

	resultValue, resultFlags := GetCoinMapEntryTest(singleEntryCacheTest.cache.cacheCoins)
	if expectedValue != resultValue {
		panic("expectedValue should be equal to resultValue")
	}
	if expectedFlags != resultFlags {
		panic("expectedFlags should be equal to resultFlags")
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

	CheckSpendCoin(TEST_ABSENT, TEST_ABSENT, TEST_ABSENT, TEST_NO_ENTRY, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_ABSENT, TEST_PRUNED, TEST_PRUNED, 0, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_ABSENT, TEST_PRUNED, TEST_ABSENT, COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_ABSENT, TEST_PRUNED, TEST_PRUNED, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_ABSENT, TEST_PRUNED, TEST_ABSENT, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_ABSENT, TEST_VALUE2, TEST_PRUNED, 0, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_ABSENT, TEST_VALUE2, TEST_ABSENT, COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_ABSENT, TEST_VALUE2, TEST_PRUNED, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_ABSENT, TEST_VALUE2, TEST_ABSENT, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_PRUNED, TEST_ABSENT, TEST_ABSENT, TEST_NO_ENTRY, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_PRUNED, TEST_PRUNED, TEST_PRUNED, 0, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_PRUNED, TEST_PRUNED, TEST_ABSENT, COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_PRUNED, TEST_PRUNED, TEST_PRUNED, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_PRUNED, TEST_PRUNED, TEST_ABSENT, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_PRUNED, TEST_VALUE2, TEST_PRUNED, 0, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_PRUNED, TEST_VALUE2, TEST_ABSENT, COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_PRUNED, TEST_VALUE2, TEST_PRUNED, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_PRUNED, TEST_VALUE2, TEST_ABSENT, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_VALUE1, TEST_ABSENT, TEST_PRUNED, TEST_NO_ENTRY, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_VALUE1, TEST_PRUNED, TEST_PRUNED, 0, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_VALUE1, TEST_PRUNED, TEST_ABSENT, COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_VALUE1, TEST_PRUNED, TEST_PRUNED, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_VALUE1, TEST_PRUNED, TEST_ABSENT, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_VALUE1, TEST_VALUE2, TEST_PRUNED, 0, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_VALUE1, TEST_VALUE2, TEST_ABSENT, COIN_ENTRY_FRESH, TEST_NO_ENTRY)
	CheckSpendCoin(TEST_VALUE1, TEST_VALUE2, TEST_PRUNED, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY)
	CheckSpendCoin(TEST_VALUE1, TEST_VALUE2, TEST_ABSENT, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, TEST_NO_ENTRY)
}

func CheckAddCoinBase(baseValue btcutil.Amount, cacheValue btcutil.Amount, modifyValue btcutil.Amount, expectedValue btcutil.Amount, cacheFlags int, expectedFlags int, isCoinbase bool) {
	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, int(cacheFlags))

	var resultValue btcutil.Amount
	var resultFlags int
	defer func() {
		if r := recover(); r != nil {
			resultValue = TEST_FAIL
			resultFlags = TEST_NO_ENTRY
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

	txOut := model.NewTxOut(int64(modifyValue), []byte{})
	coin := NewCoin(txOut, 1, isCoinbase)
	singleEntryCacheTest.cache.AddCoin(&TestOutPoint, *coin, isCoinbase)
	singleEntryCacheTest.cache.SelfTest()
	resultValue, resultFlags = GetCoinMapEntryTest(singleEntryCacheTest.cache.cacheCoins)
}

func CheckAddCoin(cacheValue btcutil.Amount, modifyValue btcutil.Amount, expectedValue btcutil.Amount, cacheFlags int, expectedFlags int, isCoinbase bool) {
	for _, arg := range [3]btcutil.Amount{TEST_ABSENT, TEST_PRUNED, TEST_VALUE1} {
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
	CheckAddCoin(TEST_ABSENT, TEST_VALUE3, TEST_VALUE3, TEST_NO_ENTRY, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, false)
	CheckAddCoin(TEST_ABSENT, TEST_VALUE3, TEST_VALUE3, TEST_NO_ENTRY, COIN_ENTRY_DIRTY, true)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, 0, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, false)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, 0, COIN_ENTRY_DIRTY, true)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_FRESH, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, false)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_FRESH, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, true)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY, false)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY, true)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, false)
	CheckAddCoin(TEST_PRUNED, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, true)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_FAIL, 0, TEST_NO_ENTRY, false)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_VALUE3, 0, COIN_ENTRY_DIRTY, true)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_FAIL, COIN_ENTRY_FRESH, TEST_NO_ENTRY, false)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_FRESH, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, true)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_FAIL, COIN_ENTRY_DIRTY, TEST_NO_ENTRY, false)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_DIRTY, COIN_ENTRY_DIRTY, true)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_FAIL, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, TEST_NO_ENTRY, false)
	CheckAddCoin(TEST_VALUE2, TEST_VALUE3, TEST_VALUE3, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, COIN_ENTRY_DIRTY|COIN_ENTRY_FRESH, true)
}

// new a insecure rand creator from crypto/rand seed
func newInsecureRand() []byte {
	randByte := make([]byte, 32)
	_, err := rand.Read(randByte)
	if err != nil {
		panic("init rand number creator failed...")
	}
	return randByte
}

// GetRandHash create a random Hash(utils.Hash)
func GetRandHash() *utils.Hash {
	tmpStr := hex.EncodeToString(newInsecureRand())
	return utils.HashFromString(tmpStr)
}

// InsecureRandRange create a random number in [0, limit]
func InsecureRandRange(limit uint64) uint64 {
	if limit == 0 {
		fmt.Println("param 0 will be insignificant")
		return 0
	}
	r := newInsecureRand()
	return binary.LittleEndian.Uint64(r) % (limit + 1)
}

// InsecureRand32 create a random number in [0 math.MaxUint32]
func InsecureRand32() uint32 {
	r := newInsecureRand()
	return binary.LittleEndian.Uint32(r)
}

// InsecureRandBits create a random number following  specified bit count
func InsecureRandBits(bit uint8) uint64 {
	r := newInsecureRand()
	maxNum := uint64(((1<<(bit-1))-1)*2 + 1 + 1)
	return binary.LittleEndian.Uint64(r) % maxNum
}

// InsecureRandBool create true or false randomly
func InsecureRandBool() bool {
	r := newInsecureRand()
	remainder := binary.LittleEndian.Uint16(r) % 2
	return remainder == 1
}

func TestRandomFunction(t *testing.T) {
	trueCount := 0
	falseCount := 0

	for i := 0; i < 10000; i++ {
		NumUint64 := InsecureRandRange(100)
		if NumUint64 > 100 {
			t.Error("InsecureRandRange() create a random number bigger than 10000")
		}

		NumUint32 := InsecureRand32()
		if NumUint32 > math.MaxUint32 {
			t.Error("InsecureRand32() creates a random number bigger than math.MaxUint32")
		}

		NumFromRandBit := InsecureRandBits(6)
		if NumFromRandBit > (((1<<(6-1))-1)*2 + 1) {
			t.Error("InsecureRandBits() creates a random numner bigger than bit-specific MaxNumber")
		}

		BoolFromRandFunc := InsecureRandBool()
		if BoolFromRandFunc {
			trueCount++
		} else {
			falseCount++
		}
	}

	if trueCount == 0 || falseCount == 0 {
		t.Error("InsecureRandBool() maybe needed to check")
	}
}
