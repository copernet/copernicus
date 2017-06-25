package mempool

import (
	"copernicus/msg"
)

type MempoolConfig struct {
	MempoolPolicy MempoolPolicy
	BitcoinParams *msg.BitcoinParams
//	FetchUTXOViewfunc func(transaction model.Transaction)()
	
}
