package util

import "testing"

var a, b, c = 1, 2, 3

func TestMaxI(t *testing.T) {
	num := MaxI(int64(a), int64(b))
	if num != int64(b) {
		t.Errorf("test MaxI failed, num value is: %d", num)
	}

	num1 := MaxI(int64(c), int64(b))
	if num1 != int64(c) {
		t.Errorf("test MaxI failed, num value is: %d", num)
	}
}

func TestMaxU(t *testing.T) {
	num := MaxU(uint64(a), uint64(b))
	if num != uint64(b) {
		t.Errorf("test MaxU failed, num value is: %d", num)
	}

	num1 := MaxU(uint64(c), uint64(b))
	if num1 != uint64(c) {
		t.Errorf("test MaxU failed, num value is: %d", num)
	}
}

func TestMaxI32(t *testing.T) {
	num := MaxI32(int32(a), int32(b))
	if num != int32(b) {
		t.Errorf("test MaxI32 failed, num value is: %d", num)
	}

	num1 := MaxI32(int32(c), int32(b))
	if num1 != int32(c) {
		t.Errorf("test MaxI32 failed, num value is: %d", num)
	}
}

func TestMaxU32(t *testing.T) {
	num := MaxU32(uint32(a), uint32(b))
	if num != uint32(b) {
		t.Errorf("test MaxU32 failed, num value is: %d", num)
	}

	num1 := MaxU32(uint32(c), uint32(b))
	if num1 != uint32(c) {
		t.Errorf("test MaxU32 failed, num value is: %d", num)
	}
}

func TestMinI(t *testing.T) {
	num := MinI(int64(a), int64(b))
	if num != int64(a) {
		t.Errorf("test MinI failed, num value is: %d", num)
	}

	num1 := MinI(int64(c), int64(b))
	if num1 != int64(b) {
		t.Errorf("test MinI failed, num value is: %d", num)
	}
}

func TestMinU(t *testing.T) {
	num := MinU(uint64(a), uint64(b))
	if num != uint64(a) {
		t.Errorf("test MinU failed, num value is: %d", num)
	}

	num1 := MinU(uint64(c), uint64(b))
	if num1 != uint64(b) {
		t.Errorf("test MinU failed, num value is: %d", num)
	}
}

func TestMinI32(t *testing.T) {
	num := MinI32(int32(a), int32(b))
	if num != int32(a) {
		t.Errorf("test MinI32 failed, num value is: %d", num)
	}

	num1 := MinI32(int32(c), int32(b))
	if num1 != int32(b) {
		t.Errorf("test MinI32 failed, num value is: %d", num)
	}
}

func TestMinU32(t *testing.T) {
	num := MinU32(uint32(a), uint32(b))
	if num != uint32(a) {
		t.Errorf("test MinU32 failed, num value is: %d", num)
	}

	num1 := MinU32(uint32(c), uint32(b))
	if num1 != uint32(b) {
		t.Errorf("test MinU32 failed, num value is: %d", num)
	}
}
