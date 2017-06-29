package mempool

import (
	"github.com/btccom/copernicus/msg"
)

type TxPoolConfig struct {
	MempoolPolicy PoolPolicy
	BitcoinParams *msg.BitcoinParams
	//	FetchUTXOViewfunc func(transaction model.Transaction)()

}
