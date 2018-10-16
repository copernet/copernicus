package errcode

import "fmt"

type TxErr int

const (
	TxErrRejectNonstandard     TxErr = 0x40
	TxErrRejectDust            TxErr = 0x41
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
	TxErrRejectNonstandard:     "TxErrRejectNonstandard",
	TxErrRejectDust:            "TxErrRejectDust",
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
