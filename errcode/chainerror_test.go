package errcode

import (
	"strconv"
	"testing"
)

func TestChainErr_String(t *testing.T) {
	tests := []struct {
		in   ChainErr
		want string
	}{
		{ErrorBlockHeaderNoValid, "The block header is not valid"},
		{ErrorBlockHeaderNoParent, "Can not find this block header's father"},
		{ErrorBlockSize, "ErrorBlockSize"},
		{ErrorPowCheckErr, "ErrorPowCheckErr"},
		{ErrorBadTxnMrklRoot, "ErrorBadTxnMrklRoot"},
		{ErrorbadTxnsDuplicate, "ErrorbadTxnsDuplicate"},
		{ErrorBadCoinBaseMissing, "ErrorBadCoinBaseMissing"},
		{ErrorBadBlkLength, "ErrorBadBlkLength"},
		{ErrorBadBlkTxSize, "ErrorBadBlkTxSize"},
		{ErrorBadBlkTx, "ErrorBadBlkTx"},
		{ErrorBlockAlreadyExists, "block already exists"},
		{ErrorNotExistsInChainMap, "Unknown code ("+strconv.Itoa(int(ErrorNotExistsInChainMap))+")"},
	}

	if len(tests) - 1 != int(ErrorNotExistsInChainMap) - int(ErrorBlockHeaderNoValid) {
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
