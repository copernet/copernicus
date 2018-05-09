package utxo

import (

	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/utxo"
	"fmt"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

func AddCoins(cache utxo.CoinsCache, tx tx.Tx, height int) {
	isCoinbase := tx.IsCoinBase()
	txid := tx.Hash
	for i, out := range tx.GetOuts() {
		// Pass fCoinbase as the possible_overwrite flag to AddCoin, in order to
		// correctly deal with the pre-BIP30 occurrences of duplicate coinbase
		// transactions.
		point := outpoint.OutPoint{Hash: txid, Index: uint32(i)}
		coin := utxo.NewCoin(out, uint32(height), isCoinbase)
		fmt.Printf("coin======%#v \n",coin.GetTxOut().GetValue())
		cache.AddCoin(&point, *coin, isCoinbase)
	}
}

func AccessByTxid(coinsCache *utxo.CoinsCache, hash *util.Hash) *utxo.Coin {
	out := outpoint.OutPoint{ *hash,  0}
	for int(out.Index) < 11000 { // todo modify to be precise
		alternate,_ := coinsCache.GetCoin(&out)
		if !alternate.IsSpent() {
			return alternate
		}
		out.Index++
	}
	return nil
}
