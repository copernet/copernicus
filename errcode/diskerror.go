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
)

var DiskErrString = map[DiskErr]string{}

func (de DiskErr) String() string {
	if s, ok := DiskErrString[de]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", de)
}
