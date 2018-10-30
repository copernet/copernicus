package errcode

import (
	"strconv"
	"testing"
)

func TestRejectCode_String(t *testing.T) {
	tests := []struct {
		in   RejectCode
		want string
	}{
		{RejectMalformed, "REJECT_MALFORMED"},
		{RejectInvalid, "REJECT_INVALID"},
		{RejectObsolete, "REJECT_OBSOLETE"},
		{RejectDuplicate, "REJECT_DUPLICATE"},
		{RejectNonstandard, "REJECT_NONSTANDARD"},
		{RejectInsufficientFee, "REJECT_INSUFFICIENTFEE"},
		{RejectCheckpoint, "REJECT_CHECKPOINT"},
		{2, "Unknown RejectCode (" + strconv.Itoa(int(2)) + ")"},
	}

	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

func TestInternalRejectCode_String(t *testing.T) {
	tests := []struct {
		in   InternalRejectCode
		want string
	}{
		{RejectHighFee, "RejectHighFee"},
		{RejectAlreadyKnown, "RejectAlreadyKnown"},
		{RejectConflict, "RejectConflict"},
		{1, "Unknown InternalRejectCode (" + strconv.Itoa(int(1)) + ")"},
	}

	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}
