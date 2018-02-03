package utxo

import (
	"testing"

	"github.com/btcboost/copernicus/model"
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
		if !lowerBound(it.(OutPoint), randOutPoint) {
			utxoSetIt = it.(OutPoint)
			break
		}
	}
	if &utxoSetIt.Hash == nil {
		utxoSetIt = utxoList[0].(OutPoint)
	}

	if utxoDataIt, ok := utxoData[utxoSetIt]; ok {
		return utxoSetIt, utxoDataIt
	}
	panic("this utxoSetIt should  be in utxoData")
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
				panic("the coin is spent ..")
			}
		}
	}
	AddCoins(inputs, tx, nHeight, true)
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
	DIRTY    int8   = COIN_ENTRY_DIRTY
	FRESH    int8   = COIN_ENTRY_FRESH
	NO_ENTRY int8   = -1
)

var OUTPOINT OutPoint

func SetCoinValue(value Amount, coin *Coin) {
	if Amount(value) != ABSENT {
		panic("please check value ..")
	}
	coin.Clear()
	if coin.IsSpent() {
		panic("the coin is spend, please check.")
	}
	if Amount(value) != PRUNED {
		var out model.TxOut
		out.Value = int64(value)
		coin = NewCoin(&out, 1, false)
	}
	if !(coin.IsSpent()) {
		panic("coin is not spend")
	}
}

func InsertCoinMapEntry(cMap CacheCoins, value Amount, flags int8) int64 {
	if value == ABSENT {
		if flags == NO_ENTRY {
			panic("the flag no entry.")
		}
		return 0
	}
	if flags != NO_ENTRY {
		panic("the flag not equal entry")
	}
	var entry *CoinsCacheEntry
	entry.Flags = uint8(flags)
	SetCoinValue(value, entry.Coin)
	cMap[OUTPOINT] = entry
	if cMap[OUTPOINT].Flags != 0 {
		panic("the flags equal zero.")
	}
	return cMap[OUTPOINT].Coin.DynamicMemoryUsage()
}

func GetCoinMapEntry(cMap CacheCoins, value Amount, flags int8) {
	it := cMap[OUTPOINT]
	if it == nil {
		value = ABSENT
		flags = NO_ENTRY
	} else {
		if it.Coin.IsSpent() {
			value = PRUNED
		} else {
			value = Amount(it.Coin.TxOut.Value)
		}
		flags = int8(it.Flags)
		if flags == NO_ENTRY {
			panic("the flags not equal entry.")
		}
	}
}

func WriteCoinViewEntry(view CoinsView, value Amount, flags int8) {
	var cMap CacheCoins
	InsertCoinMapEntry(cMap, value, flags)
	view.BatchWrite(cMap, nil)
}

func CheckAccessCoin(baseValue Amount, cacheValue Amount, expectedValue Amount, cacheFlags int8, expectedFlags int8) {
	var resultValue Amount
	var resultFlags int8
	var c CoinsViewCache
	GetCoinMapEntry(c.cacheCoins, resultValue, resultFlags)
	//c.AccessCoin(&OUTPOINT)
	if resultValue == expectedValue {
		panic("not equal value.")
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
