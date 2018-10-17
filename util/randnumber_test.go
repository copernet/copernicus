package util

import "testing"

func TestGetRandHash(t *testing.T) {
	hash := GetRandHash()
	if hash.EncodeSize() != 32 {
		t.Errorf("generate rand hash failed.")
	}
}

func TestGetRand(t *testing.T) {
	GetRand(10)

	value := GetRand(0)
	if value != 0 {
		t.Errorf("the value:%d should equal 0", value)
	}
}

func TestInsecureRand32(t *testing.T) {
	InsecureRand32()
}

func TestInsecureRand64(t *testing.T) {
	InsecureRand64()
}

func TestGetRandInt(t *testing.T) {
	GetRandInt(10)
}
