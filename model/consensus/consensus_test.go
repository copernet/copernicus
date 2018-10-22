package consensus

import (
	"testing"
)

func TestGetMaxBlockSigOpsCount(t *testing.T) {

	tests := []struct {
		in  uint64
		exp uint64
	}{
		{0, 0},
		{900000, 0},
		{8000000, 160000},
		{16000000, 320000},
		{32000000, 640000},
		{128000000, 2560000},

		{7000000, 140000},
		{17000000, 340000},
		{33000000, 660000},
		{129000000, 2580000},
	}

	for _, test := range tests {
		actual, _ := GetMaxBlockSigOpsCount(test.in)
		if actual != test.exp {
			t.Errorf("Test GetMaxBlockSigOpsCount err! Expected %d, Actual is %d", test.exp, actual)
		}
	}

}

func TestParam_DifficultyAdjustmentInterval(t *testing.T) {
	param := Param{
		TargetTimePerBlock: 60 * 10,
		TargetTimespan:     60 * 60 * 24 * 14,
	}

	exp := int64(6 * 24 * 14)
	actual := param.DifficultyAdjustmentInterval()

	if actual != exp {
		t.Errorf("Test Param_DifficultyAdjustmentInterval err! Expected %d, Actual is %d", exp, actual)
	}
}
