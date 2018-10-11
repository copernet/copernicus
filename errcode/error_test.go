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
			"module: chain, global errcode: " + strconv.Itoa(int(ErrorBlockHeaderNoValid)) + ",  desc: The block header is not valid"},
		{ErrorOutOfDiskSpace, true,
			"module: disk, global errcode: " + strconv.Itoa(int(ErrorOutOfDiskSpace)) + ",  desc: ErrorOutOfDiskSpace"},
		{MissParent, true,
			"module: mempool, global errcode: " + strconv.Itoa(int(MissParent)) + ",  desc: Miss input transaction"},
		{ModelValid, true,
			"module: rpc, global errcode: " + strconv.Itoa(int(ModelValid)) + ",  desc: Valid"},
		{ScriptErrOK, true,
			"module: script, global errcode: " + strconv.Itoa(int(ScriptErrOK)) + ",  desc: No error"},
		{TxErrNoPreviousOut, true,
			"module: transaction, global errcode: " + strconv.Itoa(int(TxErrNoPreviousOut)) + ",  desc: There is no previousout"},
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
	code, ok := HasRejectCode(New(RejectNonstandard))
	assert.Equal(t, code, RejectNonstandard)
	assert.True(t, ok)

	_, ok = HasRejectCode(New(ErrorOutOfDiskSpace))
	assert.False(t, ok)
}
