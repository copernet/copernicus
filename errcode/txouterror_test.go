package errcode

import (
	"strconv"
	"testing"
)

func TestTxOutErr_String(t *testing.T) {
	tests := []struct {
		in   TxOutErr
		want string
	}{
		{TxOutErrNegativeValue, "Tx out's value is negative"},
		{TxOutErrTooLargeValue, "TxOutErrTooLargeValue"},
		{ErrorNotInTxOutMap, "Unknown code (" + strconv.Itoa(int(ErrorNotInTxOutMap)) + ")"},
	}

	if len(tests)-1 != int(ErrorNotInTxOutMap)-int(TxOutErrNegativeValue) {
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
