package utxo

import (
	"fmt"

	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type CoinView struct {
	hashBlock        utils.Hash
	cacheCoins       map[*model.OutPoint]*CoinsCacheEntry
	cachedCoinsUsage int64
}

func (coinView *CoinView) AddCoin(outpoint *model.OutPoint, coin *Coin, possibleOverWrite bool) {
	if coin.IsSpent() {
		return
	}
	if coin.TxOut.Script.IsUnspendable() {
		return
	}
	fresh := false
	it := coinView.cacheCoins[outpoint]
	coinsCacheEntry := NewCoinsCacheEntry(coin)
	if it == nil {
		coinView.cacheCoins[outpoint] = coinsCacheEntry
		it = coinsCacheEntry
	}

	if !possibleOverWrite {
		if it.Coin.IsSpent() {
			fmt.Println("Adding new coin that replaces non-pruned entry")
		}
		fresh = it.Flags&COIN_ENTRY_DIRTY != COIN_ENTRY_DIRTY
	}
	it.Coin = coin
	if fresh {
		it.Flags |= COIN_ENTRY_DIRTY | COIN_ENTRY_FRESH
	} else {
		it.Flags |= COIN_ENTRY_DIRTY | 0
	}
	coinView.cachedCoinsUsage += it.Coin.DynamicMemoryUsage()

}

func AddCoins(cache *CoinView, tx *model.Tx, height int) {
	isCoinbase := tx.IsCoinBase()
	txid := tx.Hash
	for i, out := range tx.Outs {
		// Pass fCoinbase as the possible_overwrite flag to AddCoin, in order to
		// correctly deal with the pre-BIP30 occurrances of duplicate coinbase
		// transactions.
		outPoint := model.NewOutPoint(&txid, uint32(i))
		coin := NewCoin(out, uint32(height), isCoinbase)
		cache.AddCoin(outPoint, coin, isCoinbase)
	}

}
