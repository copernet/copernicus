package mining

import (
	"testing"
)

func TestGetSubVersionEB(t *testing.T) {
	str1 := getSubVersionEB(0)
	str2 := getSubVersionEB(2e6)
	str3 := getSubVersionEB(2e7)
	str4 := getSubVersionEB(2e8)

	if str1 != "0.0" {
		t.Error("convert error when value equal to zero")
	}

	if str2 != "0.2" {
		t.Error("convert error when value less than 1")
	}

	if str3 != "2.0" {
		t.Error("convert error when value between 1 and 10")
	}

	if str4 != "20.0" {
		t.Error("convert error when value more than 10")
	}
}
