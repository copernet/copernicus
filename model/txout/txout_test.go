package txout

import (
	"bytes"
	"github.com/copernet/copernicus/crypto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"

	"github.com/copernet/copernicus/model/script"
	"testing"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"os"
)

type TestWriter struct {
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("test writer error")
}

type ScriptBuilder struct {
	s *script.Script
}

func NewScriptBuilder() *ScriptBuilder {
	return &ScriptBuilder{
		s: script.NewEmptyScript(),
	}
}

func (sb *ScriptBuilder) PushNumber(n int) *ScriptBuilder {
	sb.s.PushScriptNum(script.NewScriptNum(int64(n)))
	return sb
}

func (sb *ScriptBuilder) PushOPCode(n int) *ScriptBuilder {
	sb.s.PushOpCode(n)
	return sb
}

func (sb *ScriptBuilder) PushBytesWithOP(data []byte) *ScriptBuilder {
	sb.s.PushSingleData(data)
	return sb
}

func (sb *ScriptBuilder) Script() *script.Script {
	return sb.s
}

func (sb *ScriptBuilder) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	sb.s.Serialize(buf)
	return buf.Bytes()
}

var (
	key0bytes = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	key1bytes = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0}

	key2bytes = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0}

	key3bytes = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0}
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
	p, _ = testTxout.IsStandard()

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

	p, _ = testTxout.IsStandard()
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

func TestTxOut_Encode(t *testing.T) {
	var buf bytes.Buffer
	txout := NewTxOut(9, nil)
	assert.NoError(t, txout.Encode(&buf))

	var w TestWriter
	txout = NewTxOut(-1, nil)
	assert.NotNil(t, txout.Encode(&w))
}

func TestTxOut_Decode(t *testing.T) {
	var r io.Reader = strings.NewReader("ABCDEFG")
	txout := NewTxOut(9, nil)
	assert.NotNil(t, txout.Decode(r))
}

func TestTxOut_GetDustThreshold(t *testing.T) {
	script := script.NewScriptRaw([]byte{opcodes.OP_RETURN, 0x01, 0x01})
	txout := NewTxOut(9, script)

	assert.Equal(t, int64(0), txout.GetDustThreshold(&util.FeeRate{SataoshisPerK: 1}))
}

func TestTxOut_CheckValue(t *testing.T) {
	txout := NewTxOut(9, nil)
	assert.NoError(t, txout.CheckValue())
}

func TestTxOut_IsStandard_true(t *testing.T) {
	scriptarr := script.NewScriptRaw([]byte{
		opcodes.OP_DUP,
		opcodes.OP_HASH160,
		0x14,
		0x41, 0xc5, 0xda, 0x42, 0x2d,
		0x1d, 0x3e, 0x6c, 0x06, 0xaf,
		0xb1, 0x9c, 0xa6, 0x2d, 0x83,
		0xb1, 0x57, 0xfc, 0x93, 0x55,
		opcodes.OP_EQUALVERIFY,
		opcodes.OP_CHECKSIG})
	txout := NewTxOut(9, scriptarr)
	pubKeyType, isStandard := txout.IsStandard()
	assert.Equal(t, script.ScriptPubkeyHash, pubKeyType)
	assert.True(t, isStandard)
}

func TestTxOut_IsStandard_ScriptNonStandard_false(t *testing.T) {
	scriptarr2 := script.NewScriptRaw([]byte{
		opcodes.OP_DUP,
		0x00,
		opcodes.OP_HASH160,
		0x14,
		0x41, 0xc5, 0xda, 0x42, 0x2d,
		0x1d, 0x3e, 0x6c, 0x06, 0xaf,
		0xb1, 0x9c, 0xa6, 0x2d, 0x83,
		0xb1, 0x57, 0xfc, 0x93, 0x55,
		opcodes.OP_EQUALVERIFY,
		opcodes.OP_CHECKSIG})

	txout := NewTxOut(9, scriptarr2)
	pubKeyType, isStandard := txout.IsStandard()
	assert.Equal(t, script.ScriptNonStandard, pubKeyType)
	assert.False(t, isStandard)
}

func TestTxOut_IsStandard_ScriptMultiSig_false(t *testing.T) {
	crypto.InitSecp256()
	key0c := crypto.NewPrivateKeyFromBytes(key0bytes[:], true)
	pubkey0c := key0c.PubKey()
	key1c := crypto.NewPrivateKeyFromBytes(key1bytes[:], true)
	pubkey1c := key1c.PubKey()
	key2c := crypto.NewPrivateKeyFromBytes(key2bytes[:], true)
	pubkey2c := key2c.PubKey()
	key3c := crypto.NewPrivateKeyFromBytes(key3bytes[:], true)
	pubkey3c := key3c.PubKey()
	scriptarr3 := NewScriptBuilder().PushOPCode(opcodes.OP_4).PushBytesWithOP(pubkey2c.ToBytes()).
		PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey0c.ToBytes()).PushBytesWithOP(pubkey3c.ToBytes()).
		PushOPCode(opcodes.OP_4).PushOPCode(opcodes.OP_CHECKMULTISIG).Script()

	txout := NewTxOut(9, scriptarr3)
	pubKeyType, isStandard := txout.IsStandard()
	assert.Equal(t, script.ScriptMultiSig, pubKeyType)
	assert.False(t, isStandard)
}

func TestTxOut_IsStandard_ScriptNullData_false(t *testing.T) {
	scriptarr4 := script.NewScriptRaw([]byte{opcodes.OP_RETURN, 1, 1})
	conf.Cfg.Script.AcceptDataCarrier = false
	txout := NewTxOut(9, scriptarr4)
	pubKeyType, isStandard := txout.IsStandard()

	assert.Equal(t, script.ScriptNullData, pubKeyType)
	assert.False(t, isStandard)
}

func TestTxOut_GetPubKeyType(t *testing.T) {
	scriptarr := script.NewScriptRaw([]byte{
		opcodes.OP_DUP,
		opcodes.OP_HASH160,
		0x14,
		0x41, 0xc5, 0xda, 0x42, 0x2d,
		0x1d, 0x3e, 0x6c, 0x06, 0xaf,
		0xb1, 0x9c, 0xa6, 0x2d, 0x83,
		0xb1, 0x57, 0xfc, 0x93, 0x55,
		opcodes.OP_EQUALVERIFY,
		opcodes.OP_CHECKSIG})
	txout := NewTxOut(9, scriptarr)

	pubKeyType, isStandard := txout.GetPubKeyType()

	assert.Equal(t, script.ScriptPubkeyHash, pubKeyType)
	assert.True(t, isStandard)
}

func TestTxOut_GetValue(t *testing.T) {
	txout := NewTxOut(9, nil)
	assert.Equal(t, amount.Amount(9), txout.GetValue())
}

func TestTxOut_SetValue(t *testing.T) {
	txout := NewTxOut(9, nil)
	txout.SetValue(8)
	assert.Equal(t, amount.Amount(8), txout.value)
}

func TestTxOut_SetScriptPubKey(t *testing.T) {
	txout := NewTxOut(9, nil)
	script := script.NewScriptRaw([]byte{
		opcodes.OP_DUP,
		opcodes.OP_HASH160,
		0x14,
		0x41, 0xc5, 0xda, 0x42, 0x2d,
		0x1d, 0x3e, 0x6c, 0x06, 0xaf,
		0xb1, 0x9c, 0xa6, 0x2d, 0x83,
		0xb1, 0x57, 0xfc, 0x93, 0x55,
		opcodes.OP_EQUALVERIFY,
		opcodes.OP_CHECKSIG})

	txout.SetScriptPubKey(script)
	assert.Equal(t, script, txout.scriptPubKey)
}

func TestTxOut_IsSpendable(t *testing.T) {
	script := script.NewScriptRaw([]byte{
		opcodes.OP_DUP,
		opcodes.OP_HASH160,
		0x14,
		0x41, 0xc5, 0xda, 0x42, 0x2d,
		0x1d, 0x3e, 0x6c, 0x06, 0xaf,
		0xb1, 0x9c, 0xa6, 0x2d, 0x83,
		0xb1, 0x57, 0xfc, 0x93, 0x55,
		opcodes.OP_EQUALVERIFY,
		opcodes.OP_CHECKSIG})
	txout := NewTxOut(9, script)

	assert.True(t, txout.IsSpendable())

	txout.SetNull()

	assert.False(t, txout.IsSpendable())
}

func TestTxOut_SetNull(t *testing.T) {
	script := script.NewScriptRaw([]byte{opcodes.OP_RETURN, 0x01, 0x01})
	txout := NewTxOut(9, script)

	txout.SetNull()
	assert.Equal(t, amount.Amount(-1), txout.value)
	assert.Nil(t, txout.scriptPubKey)
}

func TestTxOut_IsNull(t *testing.T) {
	script := script.NewScriptRaw([]byte{opcodes.OP_RETURN, 0x01, 0x01})
	txout := NewTxOut(9, script)

	txout.SetNull()

	assert.True(t, txout.IsNull())
}

func TestTxOut_String(t *testing.T) {
	script := script.NewScriptRaw([]byte{opcodes.OP_RETURN, 0x01, 0x01})
	txout := NewTxOut(9, script)

	s := txout.String()
	assert.Equal(t, "Value :9 Script:6a0101", s)
}

func TestTxOut_IsEqual(t *testing.T) {
	script := script.NewScriptRaw([]byte{opcodes.OP_RETURN, 0x01, 0x01})
	txout := NewTxOut(9, script)

	txout2 := NewTxOut(9, script)

	assert.True(t, txout.IsEqual(txout2))

	txout2.value = 8
	assert.False(t, txout.IsEqual(txout2))
}
