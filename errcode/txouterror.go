package errcode

import "fmt"

type TxOutErr int

const (
	TxOutErrNegativeValue TxOutErr = TxOutErrorBase + iota
	TxOutErrTooLargeValue

	// Error test
	ErrorNotInTxOutMap
)

var txOutErrorToString = map[TxOutErr]string{
	TxOutErrNegativeValue: "Tx out's value is negative",
	TxOutErrTooLargeValue: "TxOutErrTooLargeValue",
}

func (te TxOutErr) String() string {
	if s, ok := txOutErrorToString[te]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", te)
}
