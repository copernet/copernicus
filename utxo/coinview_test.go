package utxo

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"gopkg.in/fatih/set.v0"
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

// Store of all necessary tx and undo data for next test
type undoTx struct {
	tx   core.Tx
	undo []Coin // undo information for all txins
	coin Coin
}

// backed store
type CoinsViewTest struct {
	hashBestBlock utils.Hash
	coinMap       map[core.OutPoint]*Coin
}

func newCoinsViewTest() *CoinsViewTest {
	return &CoinsViewTest{
		coinMap: make(map[core.OutPoint]*Coin),
	}
}


func (coinsViewTest *CoinsViewTest) GetCoin(outPoint *core.OutPoint) (*Coin, error){
	c, ok := coinsViewTest.coinMap[*outPoint]
	if !ok {
		return nil, nil
	}

	if c.IsSpent() && InsecureRandBool() {
		return nil, nil
	}
	return c, nil
}

func (coinsViewTest *CoinsViewTest) HaveCoin(point *core.OutPoint) bool {
	coin, err := coinsViewTest.GetCoin(point)
	if (coin!=nil && err==nil){
		return true
	}
	return false
}

func (coinsViewTest *CoinsViewTest) GetBestBlock() utils.Hash {
	return coinsViewTest.hashBestBlock
}
func (coinsViewTest *CoinsViewTest) EstimateSize() uint64 {
	return 0
}

func (coinsViewTest *CoinsViewTest) BatchWrite(cacheCoins *CacheCoins, hashBlock *utils.Hash) error {
	for outPoint, entry := range *cacheCoins {
		if entry.dirty {
			// Same optimization used in CCoinsViewDB is to only write dirty entries.
			coinsViewTest.coinMap[outPoint] = entry.Coin
			if entry.Coin.IsSpent() && InsecureRand32()%3 == 0 {
				// Randomly delete empty entries on write.
				delete(coinsViewTest.coinMap, outPoint)
			}
		}
	}
	cacheCoins = new(CacheCoins)
	if !hashBlock.IsNull() {
		coinsViewTest.hashBestBlock = *hashBlock
	}

	return nil
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
	return c1.IsCoinBase() == c2.IsCoinBase()&&c1.GetHeight() == c2.GetHeight() && IsEqualTxOut(c1.GetTxOut(), c2.GetTxOut())
}

func IsEqualTxOut(o1 *core.TxOut, o2 *core.TxOut) bool {
	if o1.GetScriptPubKey() == nil && o2.GetScriptPubKey() == nil {
		return o1.GetValue() == o2.GetValue()
	}

	if o1.GetScriptPubKey() != nil && o2.GetScriptPubKey() != nil {
		bytes1 := o1.GetScriptPubKey().GetScriptByte()
		bytes2 := o2.GetScriptPubKey().GetScriptByte()
		if o1.GetValue() != o2.GetValue() || len(bytes1) != len(bytes2) {
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

// test whether get the expected item by OutPoint struct with a pointer
// in it or not
func TestGetCoinByPointerOrValue(t *testing.T) {
	type OutPoint struct {
		Hash  *utils.Hash
		Index int
	}

	map1 := make(map[core.OutPoint]*Coin)
	hash1 := utils.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := core.OutPoint{Hash: *hash1, Index: 0}
	// store one item
	map1[outpoint1] = &Coin{}
	hash11 := utils.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	if _, ok := map1[core.OutPoint{Hash: *hash11, Index: 0}]; !ok {
		t.Error("the key without pointer should point to a exist value")
	}

	map2 := make(map[OutPoint]*Coin)
	hash2 := utils.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint2 := OutPoint{Hash: hash2, Index: 0}
	//store one item
	map2[outpoint2] = &Coin{}
	hash22 := utils.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	if _, ok := map2[OutPoint{Hash: hash22, Index: 0}]; ok {
		t.Error("there should not be a item as the different pointer value in the struct")
	}
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
	result := make(map[core.OutPoint]*Coin)

	// The cache stack.
	// A stack of CCoinsViewCaches on top.
	stack := make([]*CoinsViewCacheTest, 0)
	// A backed store instance
	backed := newCoinsViewTest()
	// A stack of CCoinsViewCaches on top.
	item := newCoinsViewCacheTest()
	item.Base = backed
	// Start with one cache.
	stack = append(stack, item)

	// Use a limited set of random transaction ids, so we do test overwriting entries.
	var txids [NumSimulationIterations / 8]utils.Hash
	for i := 0; i < NumSimulationIterations/8; i++ {
		txids[i] = *GetRandHash()
	}

	for i := 0; i < NumSimulationIterations; i++ {
		{
			// Do a random modification.
			randomNum := InsecureRandRange(uint64(len(txids) - 1))
			// txid we're going to modify in this iteration.
			txid := txids[randomNum]
			coin, ok := result[core.OutPoint{Hash: txid, Index: 0}]

			if !ok {
				coin = NewEmptyCoin()
				result[core.OutPoint{Hash: txid, Index: 0}] = coin
			}

			randNum := InsecureRandRange(50)
			var entry *Coin
			if randNum == 0 {
				entry = AccessByTxid(&stack[len(stack)-1].CoinsViewCache, &txid)
			} else {
				entry,_ = stack[len(stack)-1].GetCoin(&core.OutPoint{Hash: txid, Index: 0})
			}

			if !IsEqualCoin(entry, coin) {
				t.Error("the coin should be equal to entry from cacheCoins or coinMap")
			}

			if InsecureRandRange(5) == 0 || coin.IsSpent() {
				var newTxOut core.TxOut
				newTxOut.SetValue(int64(InsecureRand32()))
				if InsecureRandRange(16) == 0 && coin.IsSpent() {
					newTxOut.SetScriptPubKey(core.NewScriptRaw(bytes.Repeat([]byte{byte(core.OP_RETURN)}, int(InsecureRandBits(6)+1))))
					if !newTxOut.GetScriptPubKey().IsUnspendable() {
						t.Error("error IsUnspendable")
					}
					addedAnUnspendableEntry = true
				} else {
					// Random sizes so we can test memory usage accounting
					randomBytes := bytes.Repeat([]byte{0}, int(InsecureRandBits(6)+1))
					newTxOut.SetScriptPubKey(core.NewScriptRaw(randomBytes))
					if coin.IsSpent() {
						addedAnEntry = true
					} else {
						updatedAnEntry = true
					}
					result[core.OutPoint{Hash: txid, Index: 0}] = coin
				}
				newCoin := coin
				stack[len(stack)-1].AddCoin(&core.OutPoint{Hash: txid, Index: 0}, *newCoin, !coin.IsSpent() || (InsecureRand32()&1 != 0))
			} else {
				removedAnEntry = true
				result[core.OutPoint{Hash: txid, Index: 0}].Clear()
				stack[len(stack)-1].SpendCoin(&core.OutPoint{Hash: txid, Index: 0}, nil)
			}
		}

		// One every 10 iterations, remove a random entry from the cache
		if InsecureRandRange(11) != 0 {
			cacheID := int(InsecureRand32()) % (len(stack))
			hashID := int(InsecureRand32()) % len(txids)
			out := core.OutPoint{Hash: txids[hashID], Index: 0}
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
				coin,_ := stack[len(stack)-1].GetCoin(&out)
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
					fmt.Println(out.Hash.ToString(), item.Coin.txOut.GetValue(), item.Coin.GetHeight(),item.Coin.IsCoinBase(), item.dirty,item.fresh)
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
					//todo tip.Base = stack[len(stack)-1]
				} else {
					tip.Base = backed
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

// Store of all necessary tx and undo data for next test
var utxoData = make(map[core.OutPoint]undoTx)

func newUndoTx(tx core.Tx, coins []Coin, coin Coin) undoTx {
	if coins == nil {
		return undoTx{
			tx:   tx,
			undo: make([]Coin, 0),
			coin: coin,
		}
	}

	return undoTx{
		tx:   tx,
		undo: coins,
		coin: coin,
	}
}

func lowerBound(a *core.OutPoint, b *core.OutPoint) bool {
	tmp := a.Hash.Cmp(&b.Hash)
	return tmp >= 0 || (tmp == 0 && a.Index >= b.Index)
}

func findRandomFrom(utxoSet *set.Set) (core.OutPoint, undoTx) {
	if utxoSet.Size() == 0 {
		panic("utxoSet is empty")
	}

	randOutPoint := core.OutPoint{Hash: *GetRandHash(), Index: 0}
	utxoList := utxoSet.List()

	var outPoint core.OutPoint
	var found bool
	for _, it := range utxoList {
		out := it.(core.OutPoint) // pointer
		outpoint := core.NewOutPoint(out.Hash, out.Index)
		if lowerBound(outpoint, &randOutPoint) {
			found = true
			break
		}
	}
	if found { // warning: do not use utxoSetIt == utils.HashZero
		outPoint = utxoList[0].(core.OutPoint)
	}
	utxoDataIt, ok := utxoData[outPoint]
	if !ok {
		panic("this utxoSetIt should be in utxoData")
	}
	return outPoint, utxoDataIt
}

// This test is similar to the previous test except the emphasis is on testing
// the functionality of UpdateCoins random txs are created and UpdateCoins is
// used to update the cache stack. In particular it is tested that spending a
// duplicate coinbase tx has the expected effect (the other duplicate is
// overwitten at all cache levels)
func TestUpdateCoinsSimulation(t *testing.T) {
	spentDuplicateCoinbase := false
	//A simple map to track what we expect the cache stack to represent.
	result := make(map[core.OutPoint]*Coin)

	stack := make([]*CoinsViewCacheTest, 0)

	// The cache stack.
	// A CCoinsViewTest at the bottom.
	backed := newCoinsViewTest()
	// A stack of CCoinsViewCaches on top.
	item := newCoinsViewCacheTest()
	item.Base = backed
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
		if (randiter % 20) < 19 {
			tx1 := core.NewTx()
			tx1.Ins = make([]*core.TxIn, 1)
			tx1.Ins[0] = core.NewTxIn(nil, []byte{})
			tx1.Outs = make([]*core.TxOut, 1)
			out := core.NewTxOut()
			out.SetValue(int64(i))
			s:=core.Script{}
			s.SetByteCodes(bytes.Repeat([]byte{0}, int(InsecureRand32())&0x3F))
			s.GetSigOpCount()
			out.SetScriptPubKey(&core.Script{})
			tx1.Outs[0] = out
			height := InsecureRand32()
			var oldCoin = NewEmptyCoin()

			// 2/20 times create a new coinbase
			if (randiter%20) < 2 || coinBaseCoins.Size() < 10 {
				// 1/10 of those times create a duplicate coinBase
				if InsecureRand32()%10 == 0 && coinBaseCoins.Size() != 0 {
					outKey, undoData := findRandomFrom(coinBaseCoins)
					tx1 = &undoData.tx
					disconnectedCoins.Remove(outKey)
					duplicateCoins.Add(outKey)
				} else {
					out := core.OutPoint{Hash: tx1.Hash, Index: 0} // repair add pointer
					coinBaseCoins.Add(out)
				}
				if !tx1.IsCoinBase() {
					t.Error("the tx1 should be coinBase")
				}
			} else {
				// 17/20 times reconnect previous or add a regular tx
				var prevOut core.OutPoint
				// 1/20 times reconnect a previously disconnected tx
				if (randiter%20 == 2) && (disconnectedCoins.Size() > 0) {
					out, undoData := findRandomFrom(disconnectedCoins)
					tx1 = &undoData.tx
					prevOut = *tx1.Ins[0].PreviousOutPoint
					if !tx1.IsCoinBase() && !utxoSet.Has(prevOut) {
						disconnectedCoins.Remove(out)
						continue
					}
					// If this tx is already IN the UTXO, then it must be a coinBase, and it must be a duplicate
					if utxoSet.Has(out) {
						if !tx1.IsCoinBase() {
							t.Error("the tx1 should be coinBase")
						}
						if !duplicateCoins.Has(out) {
							t.Error("duplicate coins should have this specific outpoint")
						}
						disconnectedCoins.Remove(out)
					}
				} else {
					// 16/20 times create a regular tx
					out, _ := findRandomFrom(utxoSet)
					prevOut = out

					// Construct the tx to spend the coins of prevouthash
					tx1.Ins[0].PreviousOutPoint = &prevOut
					tx1.Ins[0].PreviousOutPoint.Index = 0
					if tx1.IsCoinBase() {
						t.Error("the tx1 should not be coinBase")
					}
				}
				// In this simple test coins only have two states, spent or
				// unspent, save the unspent state to restore
				var ok bool
				oldCoin, ok = result[prevOut]
				if !ok {
					result[prevOut] = NewEmptyCoin()
					oldCoin = result[prevOut]
				}

				// Update the expected result of prevouthash to know these coins
				// are spent
				result = make(map[core.OutPoint]*Coin)

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
				t.Error("the volume of the tx1 should not be one")
			}
			OutPoint := core.NewOutPoint(tx1.Hash, 0)
			//tx1.Outs = make([]*core.TxOut, 0)
			//tx1.Outs = append(tx1.Outs, core.NewTxOut(int64(i), bytes.Repeat([]byte{0}, int(InsecureRand32())&0x3F)))
			result[*OutPoint] = NewCoin(tx1.Outs[0], height, tx1.IsCoinBase())

			// Call UpdateCoins on the top cache
			ud := make([]Coin, 0)
			updateCoins(*tx1, stack[len(stack)-1].CoinsViewCache, ud, int(height))

			// Update the utxo set for future spends
			utxoSet.Add(*OutPoint)

			// Track this tx and undo info to use later
			_, ok := utxoData[*OutPoint]
			if !ok {
				utxoData[*OutPoint] = newUndoTx(*tx1, ud, *oldCoin)
			}
		} else if utxoSet.Size() > 0 {
			// 1/20 times undo a previous transaction
			outKey, utxoData := findRandomFrom(utxoSet)

			tx2 := utxoData.tx
			ud2 := make([]Coin, 0)
			copy(ud2, utxoData.undo)
			origCoin := &utxoData.coin

			// Update the expected result
			// Remove new outputs
			delete(result, outKey)

			// If not coinbase restore prevout
			if !(tx2.IsCoinBase()) {
				result[*tx2.Ins[0].PreviousOutPoint] = origCoin
			}

			// Disconnect the tx from the current UTXO
			// See code in DisconnectBlock
			// remove outputs
			stack[len(stack)-1].SpendCoin(&outKey, nil)

			// restore inputs
			if !(tx2.IsCoinBase()) {
				out := tx2.Ins[0].PreviousOutPoint
				UndoCoinSpend(&ud2[0], &stack[len(stack)-1].CoinsViewCache, out)
			}
			// Store as a candidate for reconnection
			disconnectedCoins.Add(outKey)

			// Update the utxoset
			utxoSet.Remove(outKey)
			if !(tx2.IsCoinBase()) {
				utxoSet.Add(*tx2.Ins[0].PreviousOutPoint)
			}
		}

		//Once every 1000 iterations and at the end, verify the full cache.
		if (InsecureRandRange(1000) == 1) || (i == NumSimulationIterations-1) {
			for itKey, itValue := range result {
				have := stack[len(stack)-1].HaveCoin(&itKey)
				coin,_ := stack[len(stack)-1].GetCoin(&itKey)
				if have == coin.IsSpent() {
					t.Error("this should have been spent")
				}

				if !IsEqualCoin(coin, itValue) {
					t.Error("the coin should be equal to the value from Accession() function passed the specific key")
				}
			}
		}

		// One every 10 iterations, remove a random entry from the cache
		if (utxoSet.Size() > 1) && (InsecureRand32()%30) != 0 {
			outPoint, _ := findRandomFrom(utxoSet)
			stack[InsecureRand32()%uint32(len(stack))].UnCache(&outPoint)
		}
		if (disconnectedCoins.Size() > 1) && (InsecureRand32()%30) != 0 {
			outPoint, _ := findRandomFrom(disconnectedCoins)
			stack[InsecureRand32()%uint32(len(stack))].UnCache(&outPoint)
		}
		if (duplicateCoins.Size() > 1) && (InsecureRand32()%30) != 0 {
			outPoint, _ := findRandomFrom(disconnectedCoins)
			stack[InsecureRand32()%uint32(len(stack))].UnCache(&outPoint)
		}
		if InsecureRand32()%100 == 0 {
			// Every 100 iterations, flush an intermediate cache
			if len(stack) > 1 && InsecureRand32()%2 == 0 {
				flushIndex := InsecureRand32() % uint32(len(stack)-1)
				stack[flushIndex].Flush()
			}
		}
		if InsecureRand32()%100 == 0 {
			//Every 100 iterations, change the cache stack.
			if len(stack) > 0 && InsecureRand32()%2 == 0 {
				stack[len(stack)-1].Flush()
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 || len(stack) < 4 && InsecureRand32()%2 != 0 {
				//Add a new cache
				tip := newCoinsViewCacheTest()

				if len(stack) > 0 {
					//todo tip.Base = stack[len(stack)-1]
				} else {
					tip.Base = backed
				}

				stack = append(stack, tip)
			}
		}
	}
	//Clean up the stack.
	stack = nil

	//Verify coverage.
	if spentDuplicateCoinbase {
		t.Error("one of duplicated coinbase coins should have been spent.")
	}
}

type DisconnectResult int

const (
	DisconnectOK DisconnectResult = iota
	DisconnectUnclean
	DisconnectFailed
)

func UndoCoinSpend(undo *Coin, view *CoinsViewCache, out *core.OutPoint) DisconnectResult {
	fClean := true
	if view.HaveCoin(out) {
		// Overwriting transaction output.
		fClean = false
	}
	if undo.GetHeight() == 0 {
		// Missing undo metadata (height and coinbase). Older versions included
		// this information only in undo records for the last spend of a
		// transactions' outputs. This implies that it must be present for some
		// other output of the same tx.
		alternate := AccessByTxid(view, &out.Hash)
		if alternate.IsSpent() {
			// Adding output for transaction without known metadata
			return DisconnectFailed
		}

		// This is somewhat ugly, but hopefully utility is limited. This is only
		// useful when working from legacy on disck data. In any case, putting
		// the correct information in there doesn't hurt.
		undo = NewCoin(undo.txOut, alternate.GetHeight(), alternate.IsCoinBase())
	}
	view.AddCoin(out, *undo, undo.IsCoinBase())
	if fClean {
		return DisconnectOK
	}
	return DisconnectUnclean
}

func updateCoins(tx core.Tx, inputs CoinsViewCache, ud []Coin, nHeight int) {
	// Mark inputs spent.
	if !(tx.IsCoinBase()) {
		for _, txin := range tx.Ins {
			ud = append(ud, *NewEmptyCoin())

			isSpent := inputs.SpendCoin(txin.PreviousOutPoint, &ud[len(ud)-1])
			if isSpent {
				panic("the coin is spent ..")
			}
		}
	}
	// Add outputs.
	AddCoins(inputs, tx, nHeight)
}

var OUTPOINT = core.OutPoint{Hash: utils.HashZero, Index: math.MaxUint32}

const (
	PRUNED  utils.Amount = -1
	ABSENT  utils.Amount = -2
	FAIL    utils.Amount = -3
	VALUE1  utils.Amount = 100
	VALUE2  utils.Amount = 200
	VALUE3  utils.Amount = 300
	DIRTY                = 1
	FRESH                = 2
	NoEntry              = -1
)

var (
	FLAGS       = []int{0, FRESH, DIRTY, DIRTY | FRESH}
	CleanFlags  = []int{0, FRESH}
	AbsentFlags = []int{NoEntry}
)

type SingleEntryCacheTest struct {
	root  coinsView
	base  *CoinsViewCacheTest
	cache *CoinsViewCacheTest
}

func NewSingleEntryCacheTest(baseValue utils.Amount, cacheValue utils.Amount, cacheFlags int) *SingleEntryCacheTest {
	root := newCoinsViewTest()
	base := newCoinsViewCacheTest()
	base.Base = root
	cache := newCoinsViewCacheTest()
	cache.Base = base
	if baseValue == ABSENT {
		WriteCoinViewEntry(base, baseValue, NoEntry)
	} else {
		WriteCoinViewEntry(base, baseValue, DIRTY)
	}
	cache.cachedCoinsUsage += InsertCoinMapEntry(cache.cacheCoins, cacheValue, cacheFlags)
	return &SingleEntryCacheTest{
		root:  root,
		base:  base,
		cache: cache,
	}
}

func WriteCoinViewEntry(view coinsView, value utils.Amount, flags int) {
	cacheCoins := make(CacheCoins)
	InsertCoinMapEntry(cacheCoins, value, flags)
	view.BatchWrite(&cacheCoins, &utils.Hash{})
}

func InsertCoinMapEntry(cacheCoins CacheCoins, value utils.Amount, flags int) int64 {
	if value == ABSENT {
		if flags != NoEntry {
			panic("input flags should be NO_ENTRY")
		}
		return 0
	}
	if flags == NoEntry {
		panic("input flags should not be NO_ENTRY")
	}
	coin := NewEmptyCoin()
	SetCoinValue(value, coin)
	coinsCacheEntry := NewCoinsCacheEntry(coin)
	coinsCacheEntry.dirty = flags&1==1
	coinsCacheEntry.fresh = flags&2==2

	_, ok := cacheCoins[OUTPOINT]
	if ok {
		panic("add CoinsCacheEntry should success")
	}
	cacheCoins[OUTPOINT] = coinsCacheEntry
	return coinsCacheEntry.Coin.DynamicMemoryUsage()
}

func SetCoinValue(value utils.Amount, coin *Coin) {
	if value == ABSENT {
		panic("input value should not be equal to ABSENT")
	}
	coin.Clear()
	if !coin.IsSpent() {
		panic("coin should have spent after calling Clear() function")
	}
	if value != PRUNED {
		coin.txOut = core.NewTxOut()
		coin.txOut.SetValue(int64(value))
		coin.isCoinBase=false
		coin.height=1
	}
}

func GetCoinMapEntry(cacheCoins CacheCoins) (utils.Amount, int) {
	entry, ok := cacheCoins[OUTPOINT]
	var resultValue utils.Amount
	var resultFlags int
	if !ok {
		resultValue = ABSENT
		resultFlags = NoEntry
	} else {
		if entry.Coin.IsSpent() {
			resultValue = PRUNED
		} else {
			resultValue = utils.Amount(entry.Coin.txOut.GetValue())
		}
		//resultFlags = int(entry.Flags)
		//if resultFlags == NoEntry {
		//	panic("result_flags should not be equal to NO_ENTRY")
		//}
	}
	return resultValue, resultFlags
}

func CheckAccessCoin(baseValue utils.Amount, cacheValue utils.Amount, expectedValue utils.Amount, cacheFlags int, expectedFlags int) {
	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, cacheFlags)
	var (
		resultValue utils.Amount
		resultFlags int
	)
	singleEntryCacheTest.cache.GetCoin(&OUTPOINT)
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

func CheckSpendCoin(baseValue utils.Amount, cacheValue utils.Amount, expectedValue utils.Amount, cacheFlags int, expectedFlags int) {
	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, int(cacheFlags))
	singleEntryCacheTest.cache.SpendCoin(&OUTPOINT, nil)
	singleEntryCacheTest.cache.SelfTest()

	resultValue, resultFlags := GetCoinMapEntry(singleEntryCacheTest.cache.cacheCoins)
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

func CheckAddCoinBase(baseValue utils.Amount, cacheValue utils.Amount, modifyValue utils.Amount,
	expectedValue utils.Amount, cacheFlags int, expectedFlags int, isCoinbase bool) {

	singleEntryCacheTest := NewSingleEntryCacheTest(baseValue, cacheValue, int(cacheFlags))

	var resultValue utils.Amount
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

	txOut := core.NewTxOut()
	txOut.SetValue(int64(modifyValue))
	coin := NewCoin(txOut, 1, isCoinbase)
	singleEntryCacheTest.cache.AddCoin(&OUTPOINT, *coin, isCoinbase)
	singleEntryCacheTest.cache.SelfTest()
	resultValue, resultFlags = GetCoinMapEntry(singleEntryCacheTest.cache.cacheCoins)
}

func CheckAddCoin(cacheValue utils.Amount, modifyValue utils.Amount, expectedValue utils.Amount, cacheFlags int, expectedFlags int, isCoinbase bool) {
	for _, arg := range [3]utils.Amount{ABSENT, PRUNED, VALUE1} {
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

func CheckWriteCoin(parentValue utils.Amount, childValue utils.Amount, expectedValue utils.Amount, parentFlags int, childFlags int, expectedFlags int) {
	singleEntryCacheTest := NewSingleEntryCacheTest(ABSENT, parentValue, parentFlags)
	var (
		resultValue utils.Amount
		resultFlags int
	)
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
	CheckWriteCoin(ABSENT, ABSENT, ABSENT, NoEntry, NoEntry, NoEntry)
	CheckWriteCoin(ABSENT, PRUNED, PRUNED, NoEntry, DIRTY, DIRTY)
	CheckWriteCoin(ABSENT, PRUNED, ABSENT, NoEntry, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(ABSENT, VALUE2, VALUE2, NoEntry, DIRTY, DIRTY)
	CheckWriteCoin(ABSENT, VALUE2, VALUE2, NoEntry, DIRTY|FRESH, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, 0, NoEntry, 0)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, FRESH, NoEntry, FRESH)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, DIRTY, NoEntry, DIRTY)
	CheckWriteCoin(PRUNED, ABSENT, PRUNED, DIRTY|FRESH, NoEntry, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, 0, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, 0, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, FRESH, DIRTY, NoEntry)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, FRESH, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, PRUNED, DIRTY, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, DIRTY|FRESH, DIRTY, NoEntry)
	CheckWriteCoin(PRUNED, PRUNED, ABSENT, DIRTY|FRESH, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, 0, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, 0, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, FRESH, DIRTY|FRESH, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY, DIRTY|FRESH, DIRTY)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY|FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(PRUNED, VALUE2, VALUE2, DIRTY|FRESH, DIRTY|FRESH, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, 0, NoEntry, 0)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, FRESH, NoEntry, FRESH)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, DIRTY, NoEntry, DIRTY)
	CheckWriteCoin(VALUE1, ABSENT, VALUE1, DIRTY|FRESH, NoEntry, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, PRUNED, PRUNED, 0, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, 0, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(VALUE1, PRUNED, ABSENT, FRESH, DIRTY, NoEntry)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, FRESH, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(VALUE1, PRUNED, PRUNED, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, DIRTY, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(VALUE1, PRUNED, ABSENT, DIRTY|FRESH, DIRTY, NoEntry)
	CheckWriteCoin(VALUE1, PRUNED, FAIL, DIRTY|FRESH, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, 0, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, 0, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, FRESH, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, DIRTY, DIRTY, DIRTY)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, DIRTY, DIRTY|FRESH, NoEntry)
	CheckWriteCoin(VALUE1, VALUE2, VALUE2, DIRTY|FRESH, DIRTY, DIRTY|FRESH)
	CheckWriteCoin(VALUE1, VALUE2, FAIL, DIRTY|FRESH, DIRTY|FRESH, NoEntry)

	// The checks above omit cases where the child flags are not DIRTY, since
	// they would be too repetitive (the parent cache is never updated in these
	// cases). The loop below covers these cases and makes sure the parent cache
	// is always left unchanged.
	for parentValue := range [...]utils.Amount{ABSENT, PRUNED, VALUE1} {
		for childValue := range [...]utils.Amount{ABSENT, PRUNED, VALUE2} {

			var parentFlags []int
			if parentValue == int(ABSENT) {
				parentFlags = AbsentFlags
			} else {
				parentFlags = FLAGS
			}

			for _, parentFlag := range parentFlags {

				var childFlags []int
				if childValue == int(ABSENT) {
					childFlags = AbsentFlags
				} else {
					childFlags = CleanFlags
				}

				for _, childFlag := range childFlags {
					CheckWriteCoin(utils.Amount(parentValue), utils.Amount(childValue), utils.Amount(parentValue), parentFlag, childFlag, parentFlag)
				}
			}
		}
	}
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
