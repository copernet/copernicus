package errcode

import (
	"strconv"
	"testing"
)

func TestTxErr_String(t *testing.T) {
	tests := []struct {
		in   TxErr
		want string
	}{
		{TxErrRejectMalformed, "TxErrRejectMalformed"},
		{TxErrRejectInvalid, "TxErrRejectInvalid"},
		{TxErrRejectObsolete, "TxErrRejectObsolete"},
		{TxErrRejectDuplicate, "TxErrRejectDuplicate"},
		{TxErrRejectNonstandard, "TxErrRejectNonstandard"},
		{TxErrRejectDust, "TxErrRejectDust"},
		{TxErrRejectInsufficientFee, "TxErrRejectInsufficientFee"},
		{TxErrRejectCheckPoint, "TxErrRejectCheckPoint"},
		{TxErrRejectAlreadyKnown, "TxErrRejectAlreadyKnown"},
		{TxErrRejectConflict, "TxErrRejectConflict"},

		{TxErrNoPreviousOut, "There is no previousout"},
		{TxErrNullPreOut, "TxErrNullPreOut"},
		{TxErrNotCoinBase, "TxErrNotCoinBase"},
		{TxErrEmptyInputs, "TxErrEmptyInputs"},
		{TxErrTotalMoneyTooLarge, "TxErrTotalMoneyTooLarge"},
		{TxErrTooManySigOps, "TxErrTooManySigOps"},
		{TxErrDupIns, "TxErrDupIns"},
		{TxErrBadVersion, "TxErrBadVersion"},
		{TxErrOverSize, "TxErrOverSize"},
		{ScriptErrDustOut, "ScriptErrDustOut"},
		{TxErrNotFinal, "TxErrNotFinal"},
		{TxErrTxCommitment, "TxErrTxCommitment"},
		{TxErrMempoolAlreadyExist, "TxErrMempoolAlreadyExist"},
		{TxErrOutAlreadHave, "TxErrOutAlreadHave"},
		{TxErrInputsMoneyTooLarge, "TxErrInputsMoneyTooLarge"},
		{TxErrInputsMoneyBigThanOut, "TxErrInputsMoneyBigThanOut"},
		{ScriptCheckInputsBug, "ScriptCheckInputsBug"},
		{TxErrSignRawTransaction, "TxErrSignRawTransaction"},
		{TxErrInvalidIndexOfIn, "TxErrInvalidIndexOfIn"},
		{TxErrPubKeyType, "TxErrPubKeyType"},
		{ErrorNotInTxMap, "Unknown code (" + strconv.Itoa(int(ErrorNotInTxMap)) + ")"},
	}

	if len(tests)-1 != int(ErrorNotInTxMap)-int(TxErrNoPreviousOut)+10 {
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
