package errcode

import (
	"fmt"
)

type DiskErr int

const (
	ErrorOutOfDiskSpace DiskErr = DiskErrorBase + iota
	ErrorNotFindUndoFile
	ErrorFailedToWriteToCoinDatabase
	ErrorFailedToWriteToBlockIndexDatabase
	SystemErrorWhileFlushing
	ErrorOpenUndoFileFailed
	FailedToReadBlock
	DisconnectTipUndoFailed
	// ErrorBadBlkLength
	// ErrorBadBlkTxSize
	// ErrorBadBlkTx
	ErrorNotExistsInDiskMap    // errorTest
)

var DiskErrString = map[DiskErr]string{
	ErrorOutOfDiskSpace:                    "ErrorOutOfDiskSpace",
	ErrorNotFindUndoFile:                   "ErrorNotFindUndoFile",
	ErrorFailedToWriteToCoinDatabase:       "ErrorFailedToWriteToCoinDatabase",
	ErrorFailedToWriteToBlockIndexDatabase: "ErrorFailedToWriteToBlockIndexDatabase",
	SystemErrorWhileFlushing:               "SystemErrorWhileFlushing",
	ErrorOpenUndoFileFailed:                "ErrorOpenUndoFileFailed",
	FailedToReadBlock:                      "FailedToReadBlock",
	DisconnectTipUndoFailed:                "DisconnectTipUndoFailed",
}

func (de DiskErr) String() string {
	if s, ok := DiskErrString[de]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", de)
}
