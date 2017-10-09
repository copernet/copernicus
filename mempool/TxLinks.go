package mempool

import (
	"gopkg.in/fatih/set.v0"
)

type TxLinks struct {
	Parents  *set.Set
	Children *set.Set //The set element type : *TxMempoolEntry
}
