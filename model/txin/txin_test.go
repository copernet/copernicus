package txin

import (
	"bytes"
	"errors"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type TestWriter struct {
	Rets [3]bool
	Cnt  int
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	ret := tw.Rets[tw.Cnt]
	tw.Cnt += 1
	if ret {
		return 1, nil
	}
	return 0, errors.New("test writer error")
}

type TestReader struct {
	Cnt int
	Idx int
}

func (tr *TestReader) Read(p []byte) (n int, err error) {
	if tr.Cnt == tr.Idx {
		return 0, errors.New("test reader error")
	}
	tr.Cnt += 1
	return 1, nil
}

var preHash = util.Hash{
	0xc1, 0x60, 0x7e, 0x00, 0x31, 0xbc, 0xb1, 0x57,
	0xa3, 0xb2, 0xfd, 0x73, 0x0e, 0xcf, 0xac, 0xd1,
	0x6e, 0xda, 0x9d, 0x95, 0x7c, 0x5e, 0x03, 0xfa,
	0x34, 0x4e, 0x50, 0x21, 0xbb, 0x07, 0xcc, 0xbe,
}

var outPut = outpoint.NewOutPoint(preHash, 1)

var myScriptSig = []byte{0x16, 0x00, 0x14, 0xc3, 0xe2, 0x27, 0x9d,
	0x2a, 0xc7, 0x30, 0xbd, 0x33, 0xc4, 0x61, 0x74,
	0x4d, 0x8e, 0xd8, 0xe8, 0x11, 0xf8, 0x05, 0xdb}

var sigScript = script.NewScriptRaw(myScriptSig)

var sequence = uint32(script.SequenceFinal)

var testTxIn = NewTxIn(outPut, sigScript, sequence)

func TestNewTxIn(t *testing.T) {
	assert.Equal(t, testTxIn.scriptSig.GetData(), myScriptSig)
	assert.Equal(t, uint32(1), testTxIn.PreviousOutPoint.Index)
	assert.Equal(t, testTxIn.PreviousOutPoint.Hash[:], preHash[:])
}

func TestTxInSerialize(t *testing.T) {
	var buf bytes.Buffer

	err := testTxIn.Serialize(&buf)
	if err != nil {
		t.Error(err)
	}

	txInRead := &TxIn{}
	txInRead.PreviousOutPoint = &outpoint.OutPoint{}
	txInRead.PreviousOutPoint.Hash = util.Hash{}
	txInRead.scriptSig = script.NewEmptyScript()

	err = txInRead.Unserialize(&buf)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, txInRead.scriptSig.GetData(), testTxIn.scriptSig.GetData())
	assert.Equal(t, txInRead.PreviousOutPoint.Index, testTxIn.PreviousOutPoint.Index)
	assert.Equal(t, txInRead.PreviousOutPoint.Hash[:], testTxIn.PreviousOutPoint.Hash[:])
}

func TestTxIn_SerializeSize(t *testing.T) {
	sz := testTxIn.SerializeSize()
	assert.Equal(t, sz, uint32(64))
}

func TestTxIn_Encode_PreviousOutputPoint_false(t *testing.T) {

	w := TestWriter{Rets: [3]bool{false, false, false}, Cnt: 0}
	assert.NotNil(t, testTxIn.Encode(&w))
}

func TestTxIn_Encode_scriptSig_false(t *testing.T) {
	w := TestWriter{Rets: [3]bool{true, true, false}, Cnt: 0}
	assert.NotNil(t, testTxIn.Encode(&w))
}

func TestTxIn_Decode_PreviousOutputPoint_false(t *testing.T) {
	r := TestReader{Cnt: 0, Idx: 0}
	assert.NotNil(t, testTxIn.Decode(&r))
}

func TestTxIn_Decode_Codebase_true(t *testing.T) {
	var buf bytes.Buffer

	txin := testTxIn
	txin.PreviousOutPoint.Index = 0xffffffff
	err := txin.Serialize(&buf)
	if err != nil {
		t.Error(err)
	}

	txInRead := &TxIn{}
	txInRead.PreviousOutPoint = &outpoint.OutPoint{}
	txInRead.PreviousOutPoint.Hash = util.Hash{}
	txInRead.scriptSig = script.NewEmptyScript()

	err = txInRead.Unserialize(&buf)
	assert.NoError(t, err)
}

func TestTxIn_Decode_scriptSig_false(t *testing.T) {
	r := TestReader{Cnt: 0, Idx: 63}
	err := testTxIn.Decode(&r)
	assert.NotNil(t, err)
}

func TestTxIn_GetScriptSig(t *testing.T) {
	assert.Equal(t, sigScript, testTxIn.GetScriptSig())
}

func TestTxIn_SetScriptSig(t *testing.T) {
	txin := &TxIn{}
	txin.SetScriptSig(sigScript)
	assert.Equal(t, sigScript, txin.scriptSig)
}

func TestTxIn_CheckStandard(t *testing.T) {
	b, _ := testTxIn.CheckStandard()
	assert.True(t, b)
}

func TestTxIn_String(t *testing.T) {
	assert.Equal(t,
		"PreviousOutPoint: OutPoint (hash:becc07bb21504e34fa035e7c959dda6ed1accf0e73fdb2a357b1bc31007e60c1 index: 4294967295)  , script:160014c3e2279d2ac730bd33c461744d8ed8e811f805db , Sequence:4294967295 ",
		testTxIn.String())

	txin := testTxIn
	txin.SetScriptSig(nil)

	assert.Equal(t,
		"PreviousOutPoint: OutPoint (hash:becc07bb21504e34fa035e7c959dda6ed1accf0e73fdb2a357b1bc31007e60c1 index: 4294967295)  , script:  , Sequence:4294967295 ",
		txin.String())
}

func TestNewTxIn2(t *testing.T) {
	var txin = NewTxIn(nil, sigScript, sequence)
	assert.Equal(t, outpoint.NewOutPoint(util.Hash{}, math.MaxUint32), txin.PreviousOutPoint)
}
