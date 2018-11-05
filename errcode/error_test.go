package errcode

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestIsErrorCode(t *testing.T) {
	tests := []struct {
		errCode    fmt.Stringer
		want       bool
		descriptor string
	}{
		{ErrorBlockHeaderNoValid, true,
			"module: chain, errcode: " + strconv.Itoa(int(ErrorBlockHeaderNoValid)) + ": The block header is not valid"},
		{ErrorOutOfDiskSpace, true,
			"module: disk, errcode: " + strconv.Itoa(int(ErrorOutOfDiskSpace)) + ": ErrorOutOfDiskSpace"},
		{MissParent, true,
			"module: mempool, errcode: " + strconv.Itoa(int(MissParent)) + ": Miss input transaction"},
		{ModelValid, true,
			"module: rpc, errcode: " + strconv.Itoa(int(ModelValid)) + ": Valid"},
		{ScriptErrOK, true,
			"module: script, errcode: " + strconv.Itoa(int(ScriptErrOK)) + ": No error"},
		{TxErrNoPreviousOut, true,
			"module: transaction, errcode: " + strconv.Itoa(int(TxErrNoPreviousOut)) + ": Missing inputs"},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		err := New(test.errCode)
		result := IsErrorCode(err, test.errCode)
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, strconv.FormatBool(result), strconv.FormatBool(test.want))
		}
		if err.Error() != test.descriptor {
			t.Errorf("String #%d\n got: %s want: %s", i, err.Error(), test.descriptor)
		}
		fmt.Println(i, '\t', err.Error())
	}
}

func TestIsRejectCode(t *testing.T) {
	code, _, ok := IsRejectCode(New(RejectNonstandard))
	assert.Equal(t, code, RejectNonstandard)
	assert.True(t, ok)

	_, _, ok = IsRejectCode(New(ErrorOutOfDiskSpace))
	assert.False(t, ok)
}
