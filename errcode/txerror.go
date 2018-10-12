package errcode

import "fmt"

type TxErr int

const (
	TxErrRejectObsolete        TxErr = 0x11
	TxErrRejectDuplicate       TxErr = 0x12
	TxErrRejectNonstandard     TxErr = 0x40
	TxErrRejectDust            TxErr = 0x41
	TxErrRejectInsufficientFee TxErr = 0x42
	TxErrRejectCheckPoint      TxErr = 0x43

	TxErrRejectAlreadyKnown TxErr = 0x101
	TxErrRejectConflict     TxErr = 0x102

	TxErrNoPreviousOut TxErr = TxErrorBase + iota
	ScriptCheckInputsBug
	TxErrSignRawTransaction
	TxErrInvalidIndexOfIn
	TxErrPubKeyType
)

var txErrorToString = map[TxErr]string{
	TxErrRejectObsolete:        "TxErrRejectObsolete",
	TxErrRejectDuplicate:       "TxErrRejectDuplicate",
	TxErrRejectNonstandard:     "TxErrRejectNonstandard",
	TxErrRejectDust:            "TxErrRejectDust",
	TxErrRejectInsufficientFee: "TxErrRejectInsufficientFee",
	TxErrRejectCheckPoint:      "TxErrRejectCheckPoint",
	TxErrRejectAlreadyKnown:    "TxErrRejectAlreadyKnown",
	TxErrRejectConflict:        "TxErrRejectConflict",
	TxErrNoPreviousOut:         "There is no previousout",
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
