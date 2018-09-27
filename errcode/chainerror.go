package errcode

import "fmt"

type ChainErr int

const (
	ErrorBlockHeaderNoValid ChainErr = ChainErrorBase + iota
	ErrorBlockHeaderNoParent
	ErrorBlockSize
	ErrorPowCheckErr
	ErrorBadTxnMrklRoot
	ErrorbadTxnsDuplicate
	ErrorBadCoinBaseMissing
	ErrorBadBlkLength
	ErrorBadBlkTxSize
	ErrorBadBlkTx
	ErrorBlockAlreadyExists
)

var ChainErrString = map[ChainErr]string{
	ErrorBlockHeaderNoValid:  "The block header is not valid",
	ErrorBlockHeaderNoParent: "Can not find this block header's father ",
	ErrorPowCheckErr:         "ErrorPowCheckErr",
	ErrorBlockAlreadyExists:  "block already exists",
}

func (chainerr ChainErr) String() string {
	if s, ok := ChainErrString[chainerr]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", chainerr)
}
