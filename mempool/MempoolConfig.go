package mempool

import (
	"copernicus/protocol"
)

type MempoolConfig struct {
	MempoolPolicy MempoolPolicy
	BitcoinParams *protocol.BitcoinParams
//	FetchUTXOViewfunc func(transaction model.Transaction)()
	
}
