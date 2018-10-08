package errcode

import (
	"fmt"
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
			"module: chain, global errcode: " + strconv.Itoa(int(ErrorBlockHeaderNoValid)) + ",  errdesc: The block header is not valid"},
		{ErrorOutOfDiskSpace, true,
			"module: disk, global errcode: " + strconv.Itoa(int(ErrorOutOfDiskSpace)) + ",  errdesc: ErrorOutOfDiskSpace"},
		{MissParent, true,
			"module: mempool, global errcode: " + strconv.Itoa(int(MissParent)) + ",  errdesc: Miss input transaction"},
		{ModelValid, true,
			"module: rpc, global errcode: " + strconv.Itoa(int(ModelValid)) + ",  errdesc: Valid"},
		{ScriptErrOK, true,
			"module: script, global errcode: " + strconv.Itoa(int(ScriptErrOK)) + ",  errdesc: No error"},
		{TxErrNoPreviousOut, true,
			"module: transaction, global errcode: " + strconv.Itoa(int(TxErrNoPreviousOut)) + ",  errdesc: There is no previousout"},
		{TxOutErrNegativeValue, true,
			"module: transaction, global errcode: " + strconv.Itoa(int(TxOutErrNegativeValue)) + ",  errdesc: Tx out's value is negative"},
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
