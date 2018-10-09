package errcode

import (
	"strconv"
	"testing"
)

func TestDiskErr_String(t *testing.T) {
	tests := []struct {
		in   DiskErr
		want string
	}{
		{ErrorOutOfDiskSpace, "ErrorOutOfDiskSpace"},
		{ErrorNotFindUndoFile, "ErrorNotFindUndoFile"},
		{ErrorFailedToWriteToCoinDatabase, "ErrorFailedToWriteToCoinDatabase"},
		{ErrorFailedToWriteToBlockIndexDatabase, "ErrorFailedToWriteToBlockIndexDatabase"},
		{SystemErrorWhileFlushing, "SystemErrorWhileFlushing"},
		{ErrorOpenUndoFileFailed, "ErrorOpenUndoFileFailed"},
		{FailedToReadBlock, "FailedToReadBlock"},
		{DisconnectTipUndoFailed, "DisconnectTipUndoFailed"},
		{ErrorNotExistsInDiskMap, "Unknown code (" + strconv.Itoa(int(ErrorNotExistsInDiskMap)) + ")"},
	}

	if len(tests)-1 != int(ErrorNotExistsInDiskMap)-int(ErrorOutOfDiskSpace) {
		t.Errorf("It appears an error code was added without adding an " +
			"associated stringer test")
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}
