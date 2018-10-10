package errcode

import (
	"fmt"
)

type MemPoolErr int

const (
	MissParent MemPoolErr = MempoolErrorBase + iota
	RejectTx
	AlreadHaveTx
	Nomature
	ManyUnspendDepend
	ErrorNotExistsInMemMap
)

var merrToString = map[MemPoolErr]string{
	MissParent:        "Miss input transaction",
	RejectTx:          "The transaction reject by the rule",
	AlreadHaveTx:      "The transaction already in mempool",
	Nomature:          "Non-BIP68-final",
	ManyUnspendDepend: "The transaction depend many unspend transaction",
}

func (me MemPoolErr) String() string {
	if s, ok := merrToString[me]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", me)
}
