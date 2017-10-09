package mempool

import (
	"gopkg.in/fatih/set.v0"
)

type TxLinks struct {
	Parents  *set.Set //The set element type : TxMempoolEntry
	Children *set.Set //The set element type : *TxMempoolEntry
}
