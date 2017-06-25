package mempool

import (
	"time"
	"btcutil"
)

type TxDesc struct {
	Tx      *btcutil.Tx
	Added   time.Time
	Height  int32
	Fee     int64
	TxIndex int
}
