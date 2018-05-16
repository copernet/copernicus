package errcode

import "fmt"

type TxErr int

const TxErrBase TxErr = 2000

const (
	ErrorNoPreviousOut TxErr = TxErrBase + iota
)

var txErrorToString = map[TxErr]string {
	ErrorNoPreviousOut: "There is no previousout",
}

func (te TxErr) String() string {
	if s, ok := txErrorToString[te]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)",te)
}