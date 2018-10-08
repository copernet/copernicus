package errcode

import (
	"testing"
)

func TestRPCErr_String(t *testing.T) {
	tests := []struct {
		in   RPCErr
		want string
	}{
		{ModelValid, "Valid"},
		{ModelInvalid, "Invalid"},
		{ModelError, "Error"},
		{ErrorNotExistInRPCMap, "Unknown error code!"},

	}

	if len(tests) - 1 != int(ErrorNotExistInRPCMap) - int(ModelValid) {
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