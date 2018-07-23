package errcode

import (
	"fmt"
)

type MemPoolErr int

const MemPoolBase MemPoolErr = 1000

const (
	MissParent MemPoolErr = MemPoolBase + iota
	RejectTx
	AlreadHaveTx
	Nomature
	ManyUnspendDepend
	TooMinFeeRate
)

var merrToString = map[MemPoolErr]string{
	MissParent:        "miss input transaction",
	RejectTx:          "the transaction reject by the rule",
	AlreadHaveTx:      "the transaction already in mempool",
	Nomature:          "non-BIP68-final",
	ManyUnspendDepend: "the transaction depend many unspend transaction",
	TooMinFeeRate:     "the transaction's feerate is too minimal",
}

func (me MemPoolErr) String() string {
	if s, ok := merrToString[me]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", me)
}
