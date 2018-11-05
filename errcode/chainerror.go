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
	ErrorBlockNotStartWithCoinBase
	ErrorNotExistsInChainMap // errorTest
)

var ChainErrString = map[ChainErr]string{
	ErrorBlockHeaderNoValid:        "The block header is not valid",
	ErrorBlockHeaderNoParent:       "Can not find this block header's father",
	ErrorBlockSize:                 "ErrorBlockSize",
	ErrorPowCheckErr:               "ErrorPowCheckErr",
	ErrorBadTxnMrklRoot:            "ErrorBadTxnMrklRoot",
	ErrorbadTxnsDuplicate:          "ErrorbadTxnsDuplicate",
	ErrorBadCoinBaseMissing:        "ErrorBadCoinBaseMissing",
	ErrorBadBlkLength:              "ErrorBadBlkLength",
	ErrorBadBlkTxSize:              "ErrorBadBlkTxSize",
	ErrorBadBlkTx:                  "ErrorBadBlkTx",
	ErrorBlockAlreadyExists:        "block already exists",
	ErrorBlockNotStartWithCoinBase: "block does not start with a coinbase",
}

func (chainerr ChainErr) String() string {
	if s, ok := ChainErrString[chainerr]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", chainerr)
}
