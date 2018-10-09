package errcode

import (
	"strconv"
	"testing"
)

func TestMemPoolErr_String(t *testing.T) {
	tests := []struct {
		in   MemPoolErr
		want string
	}{
		{MissParent, "Miss input transaction"},
		{RejectTx, "The transaction reject by the rule"},
		{AlreadHaveTx, "The transaction already in mempool"},
		{Nomature, "Non-BIP68-final"},
		{ManyUnspendDepend, "The transaction depend many unspend transaction"},
		{TooMinFeeRate, "The transaction's feerate is too minimal"},
		{ErrorNotExistsInMemMap, "Unknown code (" + strconv.Itoa(int(ErrorNotExistsInMemMap)) + ")"},
	}

	if len(tests)-1 != int(ErrorNotExistsInMemMap)-int(MissParent) {
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
