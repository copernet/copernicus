package txout

import (
	"github.com/stretchr/testify/assert"

	"github.com/copernet/copernicus/model/script"
	"testing"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"os"
)

var myscript = []byte{0x14, 0x69, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
	0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31} //21 bytes

var (
	script1       = script.NewScriptRaw(myscript)
	testTxout     = NewTxOut(9, script1)
	minRelayTxFee = util.NewFeeRate(9766)
)

func TestNewTxOut(t *testing.T) {
	assert.Equal(t, amount.Amount(9), testTxout.value)
	assert.Equal(t, myscript, testTxout.GetScriptPubKey().GetData())
}

func TestSerialize(t *testing.T) {
	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	assert.Nil(t, err)

	err = testTxout.Serialize(file)
	assert.Nil(t, err)

	_, err = file.Seek(0, 0)
	assert.Nil(t, err)

	txOutRead := &TxOut{}

	txOutRead.value = 0
	txOutRead.scriptPubKey = script.NewEmptyScript()

	err = txOutRead.Unserialize(file)
	assert.Nil(t, err)

	assert.Equal(t, testTxout.value, txOutRead.value)

	assert.Equal(t, testTxout.GetScriptPubKey().GetData(), txOutRead.GetScriptPubKey().GetData())

	assert.Equal(t, uint32(30), testTxout.SerializeSize())

	err = os.Remove("tmp.txt")
	assert.Nil(t, err)
}

func TestTxOut_UnspendableOpReturnScript(t *testing.T) {
	opReturnScript := script.NewScriptRaw([]byte{opcodes.OP_RETURN, 0x01, 0x01})
	unspendableTxout := NewTxOut(9, opReturnScript)

	assert.True(t, unspendableTxout.scriptPubKey.IsUnspendable())
}

func TestTxOut_ScriptSizeTooLarge(t *testing.T) {
	bytes := make([]byte, script.MaxScriptSize+1)
	opReturnScript := script.NewScriptRaw(bytes)
	unspendableTxout := NewTxOut(9, opReturnScript)

	assert.True(t, unspendableTxout.scriptPubKey.IsUnspendable())
}

func TestIsDust(t *testing.T) {
	expected := 3 * minRelayTxFee.GetFee(int(testTxout.SerializeSize()+32+4+1+107+4))

	assert.Equal(t, expected, testTxout.GetDustThreshold(minRelayTxFee))

	assert.True(t, testTxout.IsDust(minRelayTxFee))
}

func TestCheckValue(t *testing.T) {
	testTxout.value = amount.Amount(util.MaxMoney) + 1
	expectedError := errcode.NewError(errcode.RejectInvalid, "bad-txns-vout-toolarge")
	assert.Error(t, expectedError, testTxout.CheckValue().Error())

	testTxout.value = amount.Amount(-1)
	expectedError = errcode.NewError(errcode.RejectInvalid, "bad-txns-vout-negative")
	assert.Error(t, expectedError, testTxout.CheckValue().Error())
}

func TestCheckStandard(t *testing.T) {
	conf.Cfg = conf.InitConfig([]string{})
	testTxout.scriptPubKey.ParsedOpCodes = make([]opcodes.ParsedOpCode, 5)

	poc := opcodes.NewParsedOpCode(opcodes.OP_RETURN, 1, []byte{0x6a})
	testTxout.scriptPubKey.ParsedOpCodes[0] = *poc

	var p int
	p, _ = testTxout.CheckStandard()

	assert.Equal(t, 5, p)

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
	assert.Equal(t, 2, p)
}

func TestIsCommitment(t *testing.T) {
	p := testTxout.scriptPubKey.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4})
	assert.False(t, p, "not qualified committing data")

	p = testTxout.scriptPubKey.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})
	assert.False(t, p, "header of script not OP_RETURN")

	myscript = []byte{0x6a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	script1 = script.NewScriptRaw(myscript)
	var t1 = NewTxOut(9, script1)
	testTxout = t1

	p = testTxout.scriptPubKey.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})
	assert.True(t, p)

	b := make([]byte, 10001)
	p = testTxout.scriptPubKey.IsCommitment(b)
	assert.False(t, p, "data too much to commit but misjudged")
}

func TestIsSpendable(t *testing.T) {
	//p := testTxout.IsSpendable()
	//if !p {
	//	t.Error("the txout is spendable but misjudged")
	//}
	//
	//myscript = []byte{0x6a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
	//	0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	//script1 = script.NewScriptRaw(myscript)
	//var t1 = NewTxOut(9, script1)
	//testTxout = t1
	//
	//p = testTxout.IsSpendable()
	//
	//if p {
	//	t.Error("the txout has 'returned' but still judged spendable")
	//}

}

func TestIsEqual(t *testing.T) {
	//myscript = []byte{0x14, 0x69, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
	//	0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	//script1 = script.NewScriptRaw(myscript)
	//var t0 = NewTxOut(9, script1)
	//var testTxout1 = t0
	//
	//if !testTxout.IsEqual(testTxout1) {
	//	t.Error("totally the same but misjudged")
	//}
	//
	//testTxout1.value = 8
	//
	//if testTxout.IsEqual(testTxout1) {
	//	t.Error("value mismatched but misjudged")
	//}
	//
	//myscript = []byte{0x7a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
	//	0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	//script1 = script.NewScriptRaw(myscript)
	//var t1 = NewTxOut(9, script1)
	//testTxout1 = t1
	//
	//if testTxout.IsEqual(testTxout1) {
	//	t.Error("data mismatched but misjudged")
	//}

}
