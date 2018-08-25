package txout

import (
	"testing"

	"github.com/copernet/copernicus/util/amount"
)

const (
	// amounts 0.00000001 .. 0.00100000
	numMultiplesUnit = 100000

	// amounts 0.01 .. 100.00
	numMultiplesCent = 10000

	// amounts 1 .. 10000
	numMultiples1BCH = 10000

	// amounts 50 .. 21000000
	numMultiples50BCH = 420000
)

func testEncode(in uint64) bool {
	return amount.Amount(in) == DecompressAmount(CompressAmount(amount.Amount(in)))
}

func testDecode(in uint64) bool {
	return in == CompressAmount(DecompressAmount(in))
}

func testPair(dec, enc uint64) bool {
	return CompressAmount(amount.Amount(dec)) == enc &&
		DecompressAmount(enc) == amount.Amount(dec)
}

func TestCompressAmount(t *testing.T) {
	if !testPair(0, 0x0) {
		t.Errorf("testPair(%d, %d) failed", 0, 0x0)
	}
	if !testPair(1, 0x1) {
		t.Errorf("testPair(%d, %d) failed", 1, 0x1)
	}
	if !testPair(uint64(amount.CENT), 0x7) {
		t.Errorf("testPair(%d, %d) failed", amount.CENT, 0x7)
	}
	if !testPair(uint64(amount.COIN), 0x9) {
		t.Errorf("testPair(%d, %d) failed", amount.COIN, 0x9)
	}
	if !testPair(50*uint64(amount.COIN), 0x32) {
		t.Errorf("testPair(%d, %d) failed", 50*amount.COIN, 0x32)
	}
	if !testPair(21000000*uint64(amount.COIN), 0x1406f40) {
		t.Errorf("testPair(%d, %d) failed", 21000000*amount.COIN, 0x1406f40)
	}

	for i := 1; i <= numMultiplesUnit; i++ {
		if !testEncode(uint64(i)) {
			t.Errorf("testEncode(%d) failed", i)
		}
	}
	for i := int64(1); i <= numMultiplesCent; i++ {
		if !testEncode(uint64(i * amount.CENT)) {
			t.Errorf("testEncode(%d) failed", i*amount.CENT)
		}
	}
	for i := int64(1); i <= numMultiples1BCH; i++ {
		if !testEncode(uint64(i * amount.COIN)) {
			t.Errorf("testEncode(%d) failed", i*amount.COIN)
		}
	}
	for i := int64(1); i <= numMultiples50BCH; i++ {
		if !testEncode(uint64(i * 50 * amount.COIN)) {
			t.Errorf("testEncode(%d) failed", i*50*amount.COIN)
		}
	}
	for i := 0; i < 100000; i++ {
		if !testDecode(uint64(i)) {
			t.Errorf("testDecode(%d) failed", i)
		}
	}
}
