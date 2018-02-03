package utxo

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/rand"
	"testing"
	"unsafe"

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
	coin = c
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
		if entry.Flags&uint8(COIN_ENTRY_DIRTY) != 0 {
			// Same optimization used in CCoinsViewDB is to only write dirty entries.
			coinsViewTest.coinMap[outPoint] = entry.Coin
			if entry.Coin.IsSpent() && InsecureRandRange(3) == 0 {
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

func (coinsViewCacheTest *CoinsViewCacheTest) SelfTest() {
	// Manually recompute the dynamic usage of the whole data, and compare it.
	ret := int64(unsafe.Sizeof(coinsViewCacheTest.cacheCoins))
	//ret := int64(0)
	var count int
	for _, entry := range coinsViewCacheTest.cacheCoins {
		ret += entry.Coin.DynamicMemoryUsage()
		count++
	}
	if len(coinsViewCacheTest.cacheCoins) != count {
		panic("count error")
	}

	//fmt.Println(coinsViewCacheTest.coinsViewTest.coinsViewCache.cachedCoinsUsage, ret)
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

func bytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

// new a insecure rand creator from random seed
func newInsecureRand() *rand.Rand {
	randomHash := GetRandHash()[:]
	source := rand.NewSource(bytesToInt64(randomHash))
	return rand.New(source)
}

// GetRandHash create a random Hash(utils.Hash)
func GetRandHash() *utils.Hash {
	seed := make([]byte, 32)
	rand.Read(seed)
	tmpStr := hex.EncodeToString(seed)
	return utils.HashFromString(tmpStr)
}

// InsecureRandRange create a random number in [0, limit)
func InsecureRandRange(limit int64) uint64 {
	if limit == 0 {
		return 0
	}
	r := newInsecureRand()
	return uint64(abs(r.Int63n(limit)).(int64))
}

// InsecureRand32 create a random number in [0 math.MaxUint32)
func InsecureRand32() uint32 {
	r := newInsecureRand()
	return uint32(abs(r.Int31n(math.MaxInt32)).(int32)) + uint32(abs(r.Int31n(math.MaxInt32)).(int32))
}

// InsecureRandBits create a random number following  specified bit count
func InsecureRandBits(bit uint8) uint64 {
	r := newInsecureRand()
	maxNum := int64(((1<<(bit-1))-1)*2 + 1)
	return uint64(abs(r.Int63n(maxNum)).(int64))
}

// InsecureRandBool create true or false randomly
func InsecureRandBool() bool {
	r := newInsecureRand()
	tmpInt := r.Intn(2)
	return tmpInt == 1
}

func abs(i interface{}) interface{} {
	switch i.(type) {
	case int64:
		tmp := i.(int64)
		if tmp < 0 {
			return -tmp
		}
		return tmp
	case int32:
		tmp := i.(int32)
		if tmp < 0 {
			return -tmp
		}
		return tmp
	}
	return nil
}

func TestRandom(t *testing.T) {
	trueCount := 0
	falseCount := 0

	for i := 0; i < 10000; i++ {
		NumUint64 := InsecureRandRange(1000000)
		if NumUint64 > 1000000 {
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
