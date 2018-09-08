package txout

import (
	"bytes"

	"github.com/copernet/copernicus/model/script"
	"testing"

	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"os"
)

var myscript = []byte{0x14, 0x69, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
	0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31} //21 bytes

var script1 = script.NewScriptRaw(myscript)

var testTxout = NewTxOut(9, script1)

var minRelayTxFee = util.NewFeeRate(9766)

func TestNewTxOut(t *testing.T) {

	if testTxout.value != 9 {
		t.Error("The value should be 9 instead of ", testTxout.value)
	}
	if !bytes.Equal(testTxout.GetScriptPubKey().GetData(), myscript) {
		t.Error("this data should be equal")
	}
}

func TestSerialize(t *testing.T) {

	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	err = testTxout.Serialize(file)
	if err != nil {
		t.Error(err)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		t.Error(err)
	}

	txOutRead := &TxOut{}

	txOutRead.value = 0
	txOutRead.scriptPubKey = script.NewEmptyScript()

	err = txOutRead.Unserialize(file)
	if err != nil {
		t.Error(err)
	}

	if txOutRead.value != testTxout.value {
		t.Error("The value should be equal", txOutRead.value, " : ", testTxout.value)
	}

	if !bytes.Equal(txOutRead.GetScriptPubKey().GetData(), testTxout.GetScriptPubKey().GetData()) {
		t.Error("The two []byte data should be equal ", txOutRead.GetScriptPubKey(), " : ", testTxout.GetScriptPubKey())

	}

	if testTxout.SerializeSize() != 30 {
		t.Error("the serialSize should be 29 instead of ", testTxout.SerializeSize())
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}

}

func TestIsDust(t *testing.T) {
	if testTxout.scriptPubKey.IsUnspendable() && testTxout.GetDustThreshold(minRelayTxFee) != 0 {
		t.Error("the txout is unspendable but GetDustThreshold of which doesn't return 0")
	}

	if testTxout.GetDustThreshold(minRelayTxFee) != 5214 {
		t.Error("there must exist something wrong in calculation")
	}

	if !testTxout.IsDust(minRelayTxFee) {
		t.Error("there must exist something wrong in the function IsDust")
	}

}

func TestCheckValue(t *testing.T) {

	testTxout.value = amount.Amount(util.MaxMoney) + 1
	if testTxout.CheckValue() == nil {
		t.Error("the value of textTxout is in the wrong range but not detected")
	}

	if testTxout.CheckValue().Error() != errcode.New(errcode.TxErrRejectInvalid).Error() {
		t.Error("the return of error is not well defined")
	}

	testTxout.value = amount.Amount(-1)

	if testTxout.CheckValue() == nil {
		t.Error("the value of textTxout is in the wrong range but not detected")
	}

	if testTxout.CheckValue().Error() != errcode.New(errcode.TxErrRejectInvalid).Error() {
		t.Error("the return of error is not well defined")
	}

}

func TestCheckStandard(t *testing.T) {

	testTxout.scriptPubKey.ParsedOpCodes = make([]opcodes.ParsedOpCode, 5)

	poc := opcodes.NewParsedOpCode(opcodes.OP_RETURN, 1, []byte{0x6a})
	testTxout.scriptPubKey.ParsedOpCodes[0] = *poc

	var p int
	p, _ = testTxout.CheckStandard()

	if p != 5 {
		t.Error("should return 5 but not")
	}

	poc = opcodes.NewParsedOpCode(opcodes.OP_DUP, 1, []byte{0x76})
	testTxout.scriptPubKey.ParsedOpCodes[0] = *poc
	poc = opcodes.NewParsedOpCode(opcodes.OP_HASH160, 1, []byte{0xa9})
	testTxout.scriptPubKey.ParsedOpCodes[1] = *poc
	poc = opcodes.NewParsedOpCode(opcodes.OP_PUBKEYHASH, 20, []byte{0xfd})
	testTxout.scriptPubKey.ParsedOpCodes[2] = *poc
	poc = opcodes.NewParsedOpCode(opcodes.OP_EQUALVERIFY, 1, []byte{0x88})
	testTxout.scriptPubKey.ParsedOpCodes[3] = *poc
	poc = opcodes.NewParsedOpCode(opcodes.OP_CHECKSIG, 1, []byte{0xac})
	testTxout.scriptPubKey.ParsedOpCodes[4] = *poc

	p, _ = testTxout.CheckStandard()

	if p != 2 {
		t.Error("should return 2 but not")
	}

}

func TestIsCommitment(t *testing.T) {

	p := testTxout.scriptPubKey.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4})
	if p {
		t.Error("not qualified committing data but not detected")
	}

	p = testTxout.scriptPubKey.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})

	if p {
		t.Error("header of script not OP_RETURN but not detected")
	}

	myscript = []byte{0x6a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	script1 = script.NewScriptRaw(myscript)
	var t1 = NewTxOut(9, script1)
	testTxout = t1

	p = testTxout.scriptPubKey.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})

	if !p {
		t.Error("func IsCommitment never returns true")
	}
	b := make([]byte, 10001)
	p = testTxout.scriptPubKey.IsCommitment(b)

	if p {
		t.Error("data too much to commit but misjudged")
	}

}

func TestIsSpendable(t *testing.T) {
	p := testTxout.IsSpendable()
	if !p {
		t.Error("the txout is spendable but misjudged")
	}

	myscript = []byte{0x6a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	script1 = script.NewScriptRaw(myscript)
	var t1 = NewTxOut(9, script1)
	testTxout = t1

	p = testTxout.IsSpendable()

	if p {
		t.Error("the txout has 'returned' but still judged spendable")
	}

}

func TestIsEqual(t *testing.T) {

	myscript = []byte{0x14, 0x69, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	script1 = script.NewScriptRaw(myscript)
	var t0 = NewTxOut(9, script1)
	var testTxout1 = t0

	if !testTxout.IsEqual(testTxout1) {
		t.Error("totally the same but misjudged")
	}

	testTxout1.value = 8

	if testTxout.IsEqual(testTxout1) {
		t.Error("value mismatched but misjudged")
	}

	myscript = []byte{0x7a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	script1 = script.NewScriptRaw(myscript)
	var t1 = NewTxOut(9, script1)
	testTxout1 = t1

	if testTxout.IsEqual(testTxout1) {
		t.Error("data mismatched but misjudged")
	}

}
