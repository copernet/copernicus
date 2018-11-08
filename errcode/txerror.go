package errcode

import "fmt"

type TxErr int

const (
	TxErrRejectCheckPoint TxErr = 0x43

	TxErrNoPreviousOut TxErr = TxErrorBase + iota
	ScriptCheckInputsBug
	TxErrSignRawTransaction
	TxErrInvalidIndexOfIn
	TxErrPubKeyType
)

var txErrorToString = map[TxErr]string{
	TxErrRejectCheckPoint:   "TxErrRejectCheckPoint",
	TxErrNoPreviousOut:      "Missing inputs",
	ScriptCheckInputsBug:    "ScriptCheckInputsBug",
	TxErrSignRawTransaction: "TxErrSignRawTransaction",
	TxErrInvalidIndexOfIn:   "TxErrInvalidIndexOfIn",
	TxErrPubKeyType:         "TxErrPubKeyType",
}

func (te TxErr) String() string {
	if s, ok := txErrorToString[te]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", te)
}
