package algorithm

import "testing"

func TestMinUint32(t *testing.T) {
	var a, b uint32
	a = 1
	b = 1
	result := MinUint32(a, b)
	if result != 1 {
		t.Errorf("MinUint32 is %d , not 1", result)
	}

	a = 2
	b = 3
	result = MinUint32(a, b)
	if result != 2 {
		t.Errorf("MinUint32 is %d , not 2", result)
	}

	a = 4
	b = 3
	result = MinUint32(a, b)
	if result != 3 {
		t.Errorf("MinUint32 got %d , not 3", result)
	}
}
