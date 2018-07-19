package errcode

import "fmt"

type TxOutErr int

const (
	TxOutErrNegativeValue TxOutErr = TxOutErrorBase + iota
	TxOutErrTooLargeValue
)

var txOutErrorToString = map[TxOutErr]string{
	TxOutErrNegativeValue: "Tx out's value is negative",
}

func (te TxOutErr) String() string {
	if s, ok := txOutErrorToString[te]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", te)
}
