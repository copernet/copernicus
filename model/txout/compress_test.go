package txout

import (
	"bytes"
	"errors"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/stretchr/testify/assert"
	"testing"
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

type DummyWriter struct {
	Cnt int
	Idx int
}

func (tw *DummyWriter) Write(p []byte) (n int, err error) {
	if tw.Cnt == tw.Idx {
		return 0, errors.New("test writer error")
	}
	tw.Cnt++
	return 1, nil
}

type DummyReader struct {
	Cnt int
	Idx int
}

func (tr *DummyReader) Read(p []byte) (n int, err error) {
	if tr.Cnt == tr.Idx {
		return 0, errors.New("test reader error")
	}
	tr.Cnt++
	return 1, nil
}

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
	if !testPair(uint64(util.COIN), 0x9) {
		t.Errorf("testPair(%d, %d) failed", util.COIN, 0x9)
	}
	if !testPair(50*uint64(util.COIN), 0x32) {
		t.Errorf("testPair(%d, %d) failed", 50*util.COIN, 0x32)
	}
	if !testPair(21000000*uint64(util.COIN), 0x1406f40) {
		t.Errorf("testPair(%d, %d) failed", 21000000*util.COIN, 0x1406f40)
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
		if !testEncode(uint64(i * util.COIN)) {
			t.Errorf("testEncode(%d) failed", i*util.COIN)
		}
	}
	for i := int64(1); i <= numMultiples50BCH; i++ {
		if !testEncode(uint64(i * 50 * util.COIN)) {
			t.Errorf("testEncode(%d) failed", i*50*util.COIN)
		}
	}
	for i := 0; i < 100000; i++ {
		if !testDecode(uint64(i)) {
			t.Errorf("testDecode(%d) failed", i)
		}
	}
}

func Test_scriptCompressor_newScriptCompressor(t *testing.T) {
	assert.Nil(t, newScriptCompressor(nil))

	var s *script.Script
	assert.Equal(t, script.NewEmptyScript(), *newScriptCompressor(&s).sp)
}

func Test_scriptCompressor_isToKeyID(t *testing.T) {
	var s *script.Script
	sc := newScriptCompressor(&s)
	assert.Nil(t, sc.isToKeyID())
}

func Test_scriptCompressor_isToScriptID(t *testing.T) {
	var s *script.Script
	sc := newScriptCompressor(&s)
	assert.Nil(t, sc.isToScriptID())

	bs := make([]byte, 23)
	bs[0] = opcodes.OP_HASH160
	bs[1] = 20
	bs[22] = opcodes.OP_EQUAL
	s = script.NewScriptRaw(bs)
	sc = newScriptCompressor(&s)
	assert.NotNil(t, sc.isToScriptID())
}

func Test_scriptCompressor_isToPubKey(t *testing.T) {
	var s *script.Script
	sc := newScriptCompressor(&s)
	assert.Nil(t, sc.isToPubKey())

	bs := make([]byte, 35)
	bs[0] = 33
	bs[34] = opcodes.OP_CHECKSIG
	bs[1] = 0x2
	bs[1] = 0x3
	s = script.NewScriptRaw(bs)
	sc = newScriptCompressor(&s)
	assert.NotNil(t, sc.isToPubKey())

	bs = make([]byte, 67)
	bs[0] = 65
	bs[66] = opcodes.OP_CHECKSIG
	bs[1] = 0x4
	s = script.NewScriptRaw(bs)
	sc = newScriptCompressor(&s)
	//assert.NotNil(t, sc.isToPubKey())
}

func Test_scriptCompressor_Compress(t *testing.T) {
	bs := make([]byte, 35)
	bs[0] = 33
	bs[34] = opcodes.OP_CHECKSIG
	bs[1] = 0x2
	s := script.NewScriptRaw(bs)
	sc := newScriptCompressor(&s)
	assert.NotNil(t, sc.Compress())

	bs = make([]byte, 23)
	bs[0] = opcodes.OP_HASH160
	bs[1] = 20
	bs[22] = opcodes.OP_EQUAL
	s = script.NewScriptRaw(bs)
	sc = newScriptCompressor(&s)
	assert.NotNil(t, sc.Compress())

	bs[1] = 4
	s = script.NewScriptRaw(bs)
	sc = newScriptCompressor(&s)
	assert.Nil(t, sc.Compress())
}

func Test_getSpecialSize(t *testing.T) {
	assert.Equal(t, 32, getSpecialSize(2))
	assert.Equal(t, 0, getSpecialSize(6))
}

func Test_scriptCompressor_Decompress(t *testing.T) {
	bs := make([]byte, 23)
	bs[0] = opcodes.OP_HASH160
	bs[1] = 20
	bs[22] = opcodes.OP_EQUAL
	s := script.NewScriptRaw(bs)
	sc := newScriptCompressor(&s)
	b := sc.Compress()
	assert.True(t, sc.Decompress(1, b))

	bs = make([]byte, 35)
	bs[0] = 33
	bs[34] = opcodes.OP_CHECKSIG
	bs[1] = 0x2
	s = script.NewScriptRaw(bs)
	sc = newScriptCompressor(&s)
	b = sc.Compress()
	assert.True(t, sc.Decompress(2, b))

	//assert.True(t, sc.Decompress(4, b))

	assert.False(t, sc.Decompress(6, b))
}

func Test_scriptCompressor_Serialize(t *testing.T) {
	var s *script.Script
	sc := newScriptCompressor(&s)
	w := DummyWriter{Cnt: 0, Idx: 0}
	assert.NotNil(t, sc.Serialize(&w))

	w = DummyWriter{Cnt: 0, Idx: 1}
	assert.NotNil(t, sc.Serialize(&w))

	w = DummyWriter{Cnt: 0, Idx: 2}
	assert.Nil(t, sc.Serialize(&w))
}

func Test_scriptCompressor_Unserialize(t *testing.T) {
	var s *script.Script
	sc := newScriptCompressor(&s)
	r := DummyReader{Cnt: 0, Idx: 0}
	assert.NotNil(t, sc.Unserialize(&r))

	r = DummyReader{Cnt: 0, Idx: 1}
	assert.NotNil(t, sc.Unserialize(&r))

	r = DummyReader{Cnt: 0, Idx: 2}
	assert.NotNil(t, sc.Unserialize(&r))

	var buf bytes.Buffer
	bs := make([]byte, 23)
	bs[0] = opcodes.OP_HASH160
	bs[1] = 20
	bs[22] = opcodes.OP_EQUAL
	s = script.NewScriptRaw(bs)
	sc = newScriptCompressor(&s)
	sc.Serialize(&buf)
	sc.Unserialize(&buf)
}

func getTestScript() *script.Script {
	return script.NewScriptRaw(
		[]byte{
			0x76, // OP_DUP
			0xa9, // OP_HASH160
			0x14, // OP_DATA_20
			0x03, 0xef, 0xb6, 0xc9,
			0xd3, 0x87, 0xb9, 0x7b,
			0x59, 0x8a, 0x26, 0x64,
			0x22, 0x5f, 0xe7, 0xb7,
			0x9a, 0x0a, 0xe0, 0x55,
			0x88, // OP_EQUALVERIFY
			0xac, // OP_CHECKSIG
		},
	)
}

func Test_TxoutCompressor_NewTxoutCompressor(t *testing.T) {
	txoutCompressor := NewTxoutCompressor(NewTxOut(0x33a478, getTestScript()))
	assert.NotNil(t, txoutCompressor)

	txoutCompressor = NewTxoutCompressor(nil)
	assert.Nil(t, txoutCompressor)
}

func Test_TxoutCompressor_Serialize(t *testing.T) {
	w := DummyWriter{Cnt: 0, Idx: 0}
	txoutCompressor := NewTxoutCompressor(nil)
	assert.Equal(t, ErrCompress, txoutCompressor.Serialize(&w))

	txoutCompressor = NewTxoutCompressor(NewTxOut(0x33a478, getTestScript()))
	var buf bytes.Buffer
	assert.NoError(t, txoutCompressor.Serialize(&buf))

	w = DummyWriter{Cnt: 0, Idx: 0}
	assert.NotNil(t, txoutCompressor.Serialize(&w))
}

func Test_TxoutCompressor_Unserialize(t *testing.T) {
	r := DummyReader{Cnt: 0, Idx: 0}
	txoutCompressor := NewTxoutCompressor(nil)
	assert.Equal(t, ErrCompress, txoutCompressor.Unserialize(&r))

	txoutCompressor = NewTxoutCompressor(NewTxOut(0x33a478, getTestScript()))
	assert.NotNil(t, txoutCompressor.Unserialize(&r))

	var buf bytes.Buffer
	txoutCompressor.Serialize(&buf)
	assert.NoError(t, txoutCompressor.Unserialize(&buf))
}
