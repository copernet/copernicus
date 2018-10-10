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
	ErrorOpenBlockDataDir
	ErrorDeleteBlockFile
	// ErrorBadBlkLength
	// ErrorBadBlkTxSize
	// ErrorBadBlkTx

	// ErrorNotExistsInDiskMap used in error test
	ErrorNotExistsInDiskMap
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
	ErrorOpenBlockDataDir:                  "ErrorOpenBlockDataDir",
	ErrorDeleteBlockFile:                   "ErrorDeleteBlockFile",
}

func (de DiskErr) String() string {
	if s, ok := DiskErrString[de]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", de)
}
