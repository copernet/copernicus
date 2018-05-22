package errcode

import "fmt"

type TxErr int

const (
	TxErrNoPreviousOut TxErr = TxErrorBase + iota
	TxErrNullPreOut
	TxErrIsCoinBase
	TxErrNotCoinBase
	TxErrEmptyInputs
	TxErrTotalMoneyTooLarge
	TxErrTooManySigOps
	TxErrDupIns
	TxErrBadVersion
	TxErrOverSize
	ScriptErrDustOut
	TxErrNotFinal
	TxErrTxCommitment
	TxErrMempoolAlreadyExist
	TxErrPreOutAlreadySpent
	TxErrInputsNotAvailable
	TxErrOutAlreadHave
)

var txErrorToString = map[TxErr]string {
	TxErrNoPreviousOut: "There is no previousout",
}

func (te TxErr) String() string {
	if s, ok := txErrorToString[te]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)",te)
}