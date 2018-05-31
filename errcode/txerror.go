package errcode

import "fmt"

type TxErr int

const (
	TxErrRejectMalformed       TxErr = 0x01
	TxErrRejectInvalid         TxErr = 0x10
	TxErrRejectObsolete        TxErr = 0x11
	TxErrRejectDuplicate       TxErr = 0x12
	TxErrRejectNonstandard     TxErr = 0x40
	TxErrRejectDust            TxErr = 0x41
	TxErrRejectInsufficientFee TxErr = 0x42
	TxErrRejectCheckPoint      TxErr = 0x43

	TxErrNoPreviousOut TxErr = TxErrorBase + iota
	TxErrNullPreOut
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
	TxErrInputsMoneyTooLarge
	TxErrInputsMoneyBigThanOut
	ScriptCheckInputsBug
	TxErrSignRawTransaction
	TxErrInvalidIndexOfIn
	TxErrPubKeyType
)

var txErrorToString = map[TxErr]string{
	TxErrNoPreviousOut: "There is no previousout",
}

func (te TxErr) String() string {
	if s, ok := txErrorToString[te]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", te)
}
