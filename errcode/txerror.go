package errcode

import "fmt"

type TxErr int

const (
	//standard reject code defined in bip-0061 from 0x01-0x43

	TxErrRejectMalformed       TxErr = 0x01
	TxErrRejectInvalid         TxErr = 0x10
	TxErrRejectObsolete        TxErr = 0x11
	TxErrRejectDuplicate       TxErr = 0x12
	TxErrRejectNonstandard     TxErr = 0x40
	TxErrRejectDust            TxErr = 0x41
	TxErrRejectInsufficientFee TxErr = 0x42
	TxErrRejectCheckPoint      TxErr = 0x43

	TxErrRejectAlreadyKnown TxErr = 0x101
	TxErrRejectConflict     TxErr = 0x102

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
	TxErrOutAlreadHave
	TxErrInputsMoneyTooLarge
	TxErrInputsMoneyBigThanOut
	ScriptCheckInputsBug
	TxErrSignRawTransaction
	TxErrInvalidIndexOfIn
	TxErrPubKeyType

	ErrorNotInTxMap /* Error test */
)

var txErrorToString = map[TxErr]string{
	TxErrRejectMalformed:       "TxErrRejectMalformed",
	TxErrRejectInvalid:         "TxErrRejectInvalid",
	TxErrRejectObsolete:        "TxErrRejectObsolete",
	TxErrRejectDuplicate:       "TxErrRejectDuplicate",
	TxErrRejectNonstandard:     "TxErrRejectNonstandard",
	TxErrRejectDust:            "TxErrRejectDust",
	TxErrRejectInsufficientFee: "TxErrRejectInsufficientFee",
	TxErrRejectCheckPoint:      "TxErrRejectCheckPoint",
	TxErrRejectAlreadyKnown:    "TxErrRejectAlreadyKnown",
	TxErrRejectConflict:        "TxErrRejectConflict",

	TxErrNoPreviousOut:         "There is no previousout",
	TxErrNullPreOut:            "TxErrNullPreOut",
	TxErrNotCoinBase:           "TxErrNotCoinBase",
	TxErrEmptyInputs:           "TxErrEmptyInputs",
	TxErrTotalMoneyTooLarge:    "TxErrTotalMoneyTooLarge",
	TxErrTooManySigOps:         "TxErrTooManySigOps",
	TxErrDupIns:                "TxErrDupIns",
	TxErrBadVersion:            "TxErrBadVersion",
	TxErrOverSize:              "TxErrOverSize",
	ScriptErrDustOut:           "ScriptErrDustOut",
	TxErrNotFinal:              "TxErrNotFinal",
	TxErrTxCommitment:          "TxErrTxCommitment",
	TxErrMempoolAlreadyExist:   "TxErrMempoolAlreadyExist",
	TxErrOutAlreadHave:         "TxErrOutAlreadHave",
	TxErrInputsMoneyTooLarge:   "TxErrInputsMoneyTooLarge",
	TxErrInputsMoneyBigThanOut: "TxErrInputsMoneyBigThanOut",
	ScriptCheckInputsBug:       "ScriptCheckInputsBug",
	TxErrSignRawTransaction:    "TxErrSignRawTransaction",
	TxErrInvalidIndexOfIn:      "TxErrInvalidIndexOfIn",
	TxErrPubKeyType:            "TxErrPubKeyType",
}

func (te TxErr) String() string {
	if s, ok := txErrorToString[te]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", te)
}
