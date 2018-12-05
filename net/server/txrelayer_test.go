package server

import (
	"github.com/copernet/copernicus/model/tx"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_tx_relayer_can_cache_txn(t *testing.T) {
	txrelayer := NewTxRelayer()

	txn := tx.NewTx(0, 1)
	hash := txn.GetHash()

	txrelayer.Cache(&hash, txn)

	cachedTxn := txrelayer.TxToRelay(&hash)

	assert.Equal(t, cachedTxn, txn)
}

func Test_tx_relayer_can_auto_limit_cache_size___to_remove_too_old_txns(t *testing.T) {
	txrelayer := NewTxRelayer()

	txn := tx.NewTx(0, 1)

	hash := txn.GetHash()
	txrelayer.Cache(&hash, txn)

	//given 15min passed
	_16minsAgo := time.Now().Add(time.Minute * -16)
	txrelayer.lastCompactTime = _16minsAgo
	txrelayer.cache[hash].cacheTime = _16minsAgo

	//relay txn2 will trigger the limit cache logic
	txn2 := tx.NewTx(0, 2)
	hash2 := txn2.GetHash()
	txrelayer.Cache(&hash2, txn2)

	//too old txn should be removed during cache limit
	cachedTxn := txrelayer.TxToRelay(&hash)
	assert.Nil(t, cachedTxn)
}
