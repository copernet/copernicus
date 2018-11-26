package script

import (
	"bytes"
	"encoding/hex"
	"github.com/copernet/copernicus/errcode"
	"github.com/stretchr/testify/assert"
	"testing"

	. "github.com/copernet/copernicus/model/opcodes"
)

var p2SHScript = [23]byte{
	OP_HASH160,
	0x14, //length
	0x89, 0xAB, 0xCD, 0xEF, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
	0xAB, 0xBA, 0xAB, 0xBA, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA, //script GetHash
	OP_EQUAL,
}

var p2PKHScript = [...]byte{
	OP_DUP,
	OP_HASH160,
	0x14,
	0x41, 0xc5, 0xda, 0x42, 0x2d,
	0x1d, 0x3e, 0x6c, 0x06, 0xaf,
	0xb1, 0x9c, 0xa6, 0x2d, 0x83,
	0xb1, 0x57, 0xfc, 0x93, 0x55,
	OP_EQUALVERIFY,
	OP_CHECKSIG,
}

func TestScriptEncodeOPN(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		// for i in $(seq 0 16); do echo "{$i, opcodes.OP_$i},"; done
		{0, OP_0},
		{1, OP_1},
		{2, OP_2},
		{3, OP_3},
		{4, OP_4},
		{5, OP_5},
		{6, OP_6},
		{7, OP_7},
		{8, OP_8},
		{9, OP_9},
		{10, OP_10},
		{11, OP_11},
		{12, OP_12},
		{13, OP_13},
		{14, OP_14},
		{15, OP_15},
		{16, OP_16},
	}

	for _, test := range tests {
		rv, err := EncodeOPN(test.input)
		if err != nil {
			t.Error(err)
		}
		if rv != test.expected {
			t.Errorf("EncodeOPN: expect %d got %d", test.expected, rv)
		}
	}

	_, err := EncodeOPN(OP_16 + 1)
	if err == nil {
		t.Error("EncodeOPN(OP_16+1) expect error")
	}
}

func TestScriptDecodeOPN(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		// for i in $(seq 0 16); do echo "{opcodes.OP_$i, $i},"; done
		{OP_0, 0},
		{OP_1, 1},
		{OP_2, 2},
		{OP_3, 3},
		{OP_4, 4},
		{OP_5, 5},
		{OP_6, 6},
		{OP_7, 7},
		{OP_8, 8},
		{OP_9, 9},
		{OP_10, 10},
		{OP_11, 11},
		{OP_12, 12},
		{OP_13, 13},
		{OP_14, 14},
		{OP_15, 15},
		{OP_16, 16},
	}

	for _, test := range tests {
		rv := DecodeOPN(byte(test.input))
		if rv != test.expected {
			t.Errorf("EncodeOPN: expect %d got %d", test.expected, rv)
		}
	}
}

func TestScript_SerializeUnSerialize(t *testing.T) {
	var buf bytes.Buffer

	testScript := NewEmptyScript()
	testScript.Serialize(&buf)

	serializeSize := testScript.SerializeSize()
	encodeSize := testScript.EncodeSize()

	resultScript := NewEmptyScript()
	resultScript.Unserialize(&buf, false)

	flag := testScript.IsEqual(resultScript)

	assert.Equal(t, testScript, resultScript)
	assert.Equal(t, serializeSize, encodeSize)
	assert.Equal(t, true, flag)
}

func TestScript_IsSpendable(t *testing.T) {
	tests := []struct {
		in   []byte
		want bool
	}{
		{[]byte{OP_RETURN, 0x00}, false},
		{[]byte{}, true},
	}

	for _, v := range tests {
		value := v

		scripts := NewScriptRaw(value.in)
		isSpendAble := scripts.IsSpendable()
		assert.Equal(t, value.want, isSpendAble)
	}
}

func TestScript_RemoveOpcode(t *testing.T) {
	tests := []struct {
		name   string
		before []byte
		remove byte
		after  []byte
	}{
		{
			// Nothing to remove.
			name:   "nothing to remove",
			before: []byte{OP_NOP},
			remove: OP_CODESEPARATOR,
			after:  []byte{OP_NOP},
		},
		{
			// Test basic opcode removal.
			name:   "codeseparator 1",
			before: []byte{OP_NOP, OP_CODESEPARATOR, OP_TRUE},
			remove: OP_CODESEPARATOR,
			after:  []byte{OP_NOP, OP_TRUE},
		},
	}

	for _, v := range tests {
		value := v

		testScript := NewScriptRaw(value.before)
		afterRemove := testScript.RemoveOpcode(value.remove)
		wantScript := NewScriptRaw(value.after)

		assert.Equal(t, wantScript, afterRemove, value.name)
	}
}

func TestScript_RemoveOpcodeByData(t *testing.T) {
	tests := []struct {
		name   string
		before []byte
		remove []byte
		err    error
		after  []byte
	}{
		{
			name:   "nothing to do",
			before: []byte{OP_NOP},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_NOP},
		},
		{
			name: "simple case (pushdata1 miss)",
			before: append(append([]byte{OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 5},
			after: append(append([]byte{OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
		},
		{
			name:   "simple case (pushdata1 miss noncanonical)",
			before: []byte{OP_PUSHDATA1, 4, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_PUSHDATA1, 4, 1, 2, 3, 4},
		},
		{
			name: "simple case (pushdata2 miss)",
			before: append(append([]byte{OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4, 5},
			after: append(append([]byte{OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
		},
		{
			name:   "simple case (pushdata2 miss noncanonical)",
			before: []byte{OP_PUSHDATA2, 4, 0, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_PUSHDATA2, 4, 0, 1, 2, 3, 4},
		},
		{
			name:   "simple case (pushdata4 miss noncanonical)",
			before: []byte{OP_PUSHDATA4, 4, 0, 0, 0, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_PUSHDATA4, 4, 0, 0, 0, 1, 2, 3, 4},
		},
		{
			// This is padded to make the push canonical.
			name: "simple case (pushdata4 miss)",
			before: append(append([]byte{OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...), []byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4, 5},
			after: append(append([]byte{OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...), []byte{1, 2, 3, 4}...),
		},
	}

	for _, v := range tests {
		value := v

		testScript := NewScriptRaw(value.before)
		afterRemove := testScript.RemoveOpcodeByData(value.remove)

		wantScript := NewScriptRaw(value.after)

		assert.Equal(t, wantScript, afterRemove, value.name)
	}
}

func TestScript_RemoveOpCodeByIndex(t *testing.T) {
	tests := []struct {
		name   string
		before []byte
		remove int
		after  []byte
	}{
		{
			// Nothing to remove.
			name:   "nothing to remove",
			before: []byte{OP_NOP},
			remove: 0,
			after:  []byte{},
		},
		{
			// Test basic opcode removal.
			name:   "codeseparator 1",
			before: []byte{OP_NOP, OP_CODESEPARATOR, OP_TRUE},
			remove: 1,
			after:  []byte{OP_NOP, OP_TRUE},
		},
	}

	for _, v := range tests {
		value := v

		testScript := NewScriptRaw(value.before)
		afterRemove := testScript.RemoveOpCodeByIndex(value.remove)
		wantScript := NewScriptRaw(value.after)

		assert.Equal(t, wantScript, afterRemove, value.name)
	}
}

func TestScript_IsCommitment(t *testing.T) {
	scriptRaw := []byte{0x6a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	testScript := NewScriptRaw(scriptRaw)
	result := testScript.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})
	assert.Equal(t, true, result)

	scriptRaw1 := []byte{0x6b, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	testScript1 := NewScriptRaw(scriptRaw1)
	result1 := testScript1.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})
	assert.Equal(t, false, result1)

	scriptRaw2 := []byte{0x6a, 0x13, 0xe2, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	testScript2 := NewScriptRaw(scriptRaw2)
	result2 := testScript2.IsCommitment([]byte{0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})
	assert.Equal(t, false, result2)

	scriptRaw3 := []byte{0x6a, 0x13, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	testScript3 := NewScriptRaw(scriptRaw3)
	result3 := testScript3.IsCommitment([]byte{0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31})
	assert.Equal(t, false, result3)
}

func TestBytesToBool(t *testing.T) {
	tests := []struct {
		in   []byte
		want bool
	}{
		{[]byte{}, false},
		{[]byte{0x80}, false},
		{[]byte{0x00}, false},
		{[]byte{0x01}, true},
	}

	for _, v := range tests {
		value := v

		result := BytesToBool(value.in)

		assert.Equal(t, value.want, result)
	}
}

func TestScript_IsStandardScriptPubKey_ScriptHash(t *testing.T) {
	wantPubKeyType := ScriptHash
	wantPubKeys := [][]byte{hexToBytes(
		"e2b7f7d12a70737429066a449615270b6851d164")}
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_HASH160)
	testScript.PushData(hexToBytes("14"))
	testScript.PushData(hexToBytes("e2b7f7d12a70737429066a449615270b6851d164"))
	testScript.PushOpCode(OP_EQUAL)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)

	var addresses []*Address
	addr, err := AddressFromString("3NMo2DkQJsMhMzpVFawT3nTL1w3UXXumxu")
	if err != nil {
		t.Error(err)
	}

	sType, address, sigCountRequire, err := testScript.ExtractDestinations()
	assert.Equal(t, wantPubKeyType, sType)
	assert.Equal(t, append(addresses, addr), address)
	assert.Equal(t, 1, sigCountRequire)
	assert.NoError(t, err)
}

func TestScript_IsStandardScriptPubKey_NotStandard(t *testing.T) {
	var wantPubKeys [][]byte
	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_OPRETURNNonData(t *testing.T) {
	var wantPubKeys [][]byte
	wantPubKeyType := ScriptNullData
	wantPubKeys = nil
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_RETURN)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_OPRETURNError(t *testing.T) {
	var wantPubKeys [][]byte
	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_EQUAL)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_OPRETURNWithData(t *testing.T) {
	var wantPubKeys [][]byte
	wantPubKeyType := ScriptNullData
	wantPubKeys = nil
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_RETURN)
	testScript.PushData(hexToBytes("0001020304"))

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)

	wantPubKeyType = ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard = false

	testScript1 := NewEmptyScript()
	testScript1.PushOpCode(OP_RETURN)
	testScript1.PushData(hexToBytes("00ffffff"))

	pubKeyType, pubKeys, isStandard = testScript1.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_OPCHECKSIG(t *testing.T) {
	wantPubKeyType := ScriptPubkey
	wantPubKeys := [][]byte{hexToBytes(
		"0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8")}
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushData(hexToBytes("41"))
	testScript.PushData(hexToBytes("0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"))
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)

	addr, err := AddressFromString("1EHNa6Q4Jz2uvNExL497mE43ikXhwF6kZm")
	if err != nil {
		t.Error(err)
	}

	sType, address, sigCountRequire, err := testScript.ExtractDestinations()
	assert.Equal(t, wantPubKeyType, sType)
	assert.EqualValues(t, addr.String(), address[0].String())
	assert.Equal(t, 1, sigCountRequire)
	assert.NoError(t, err)
}

func TestScript_IsStandardScriptPubKey_OPCHECKSIGNotStandard(t *testing.T) {
	var wantPubKeys [][]byte
	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushData(hexToBytes("14"))
	testScript.PushData(hexToBytes("23b0ad3477f2178bc0b3eed26e4e6316f4e83aa1"))
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_P2PKH(t *testing.T) {
	wantPubKeyType := ScriptPubkeyHash
	wantPubKeys := [][]byte{hexToBytes(
		"e2b7f7d12a70737429066a449615270b6851d164")}
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_DUP)
	testScript.PushOpCode(OP_HASH160)
	testScript.PushSingleData(hexToBytes("e2b7f7d12a70737429066a449615270b6851d164"))
	testScript.PushOpCode(OP_EQUALVERIFY)
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)

	addr, err := AddressFromString("1Mfn6gFxky3KGq848VGrdA6PsQkkzEt7xo")
	if err != nil {
		t.Error(err)
	}

	sType, address, sigCountRequire, err := testScript.ExtractDestinations()
	assert.Equal(t, wantPubKeyType, sType)
	assert.EqualValues(t, addr.String(), address[0].String())
	assert.Equal(t, 1, sigCountRequire)
	assert.NoError(t, err)
}

func TestScript_IsStandardScriptPubKey_P2PKHNotStandard_WithoutPubKeyHash(t *testing.T) {
	var wantPubKeys [][]byte
	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_DUP)
	testScript.PushOpCode(OP_HASH160)
	testScript.PushOpCode(OP_EQUALVERIFY)
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_P2PKHNotStandard_PubKeyLengthError(t *testing.T) {
	var wantPubKeys [][]byte
	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_DUP)
	testScript.PushOpCode(OP_HASH160)
	testScript.PushSingleData(hexToBytes("0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"))
	testScript.PushOpCode(OP_EQUALVERIFY)
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_MultiSig(t *testing.T) {
	pubKeysData := []string{
		"0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		"038282263212c609d9ea2a6e3e172de238d8c39cabd5ac1ca10646e23fd5f51508",
		"03363d90d447b00c9c99ceac05b6262ee053441c7e55552ffe526bad8f83ff4640",
	}

	wantPubKeyType := ScriptMultiSig
	wantPubKeys := [][]byte{
		hexToBytes("02"),
		hexToBytes(pubKeysData[0]),
		hexToBytes(pubKeysData[1]),
		hexToBytes(pubKeysData[2]),
		hexToBytes("03"),
	}
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushInt64(2)
	for _, v := range pubKeysData {
		testScript.PushSingleData(hexToBytes(v))
	}
	testScript.PushInt64(3)
	testScript.PushOpCode(OP_CHECKMULTISIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)

	addresses := make([]*Address, 0, 3)
	for _, v := range pubKeysData {

		addr, err := AddressFromPublicKey(hexToBytes(v))
		if err != nil {
			continue
		}
		addresses = append(addresses, addr)
	}

	sType, address, sigCountRequire, err := testScript.ExtractDestinations()
	assert.Equal(t, wantPubKeyType, sType)
	assert.EqualValues(t, addresses, address)
	assert.Equal(t, 2, sigCountRequire)
	assert.NoError(t, err)

	notAccurateSigCount := testScript.GetSigOpCount(0, false)
	assert.EqualValues(t, 20, notAccurateSigCount)

	accurateSigCount := testScript.GetSigOpCount(0, true)
	assert.EqualValues(t, 3, accurateSigCount)
}

func TestScript_IsStandardScriptPubKey_MultiSig_PubKeyLengthError(t *testing.T) {
	pubKeysData := []string{
		"23b0ad3477f2178bc0b3eed26e4e6316f4e83aa1",
		"7f67f0521934a57d3039f77f9f32cf313f3ac74b",
		"2df519943d5acc0ef5222091f9dfe3543f489a82",
	}

	var wantPubKeys [][]byte

	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushInt64(2)
	for _, v := range pubKeysData {
		testScript.PushSingleData(hexToBytes(v))
	}
	testScript.PushInt64(3)
	testScript.PushOpCode(OP_CHECKMULTISIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_MultiSig_PubKeyNumError(t *testing.T) {
	pubKeysData := []string{
		"0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		"038282263212c609d9ea2a6e3e172de238d8c39cabd5ac1ca10646e23fd5f51508",
		"03363d90d447b00c9c99ceac05b6262ee053441c7e55552ffe526bad8f83ff4640",
	}

	var wantPubKeys [][]byte

	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushInt64(2)
	for _, v := range pubKeysData {
		testScript.PushSingleData(hexToBytes(v))
	}
	testScript.PushInt64(17)
	testScript.PushOpCode(OP_CHECKMULTISIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_MultiSig_OPCODEError(t *testing.T) {
	pubKeysData := []string{
		"0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		"038282263212c609d9ea2a6e3e172de238d8c39cabd5ac1ca10646e23fd5f51508",
		"03363d90d447b00c9c99ceac05b6262ee053441c7e55552ffe526bad8f83ff4640",
	}

	var wantPubKeys [][]byte

	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushInt64(2)
	for _, v := range pubKeysData {
		testScript.PushSingleData(hexToBytes(v))
	}
	testScript.PushInt64(3)
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_MultiSig_SigNumError(t *testing.T) {
	pubKeysData := []string{
		"0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		"038282263212c609d9ea2a6e3e172de238d8c39cabd5ac1ca10646e23fd5f51508",
		"03363d90d447b00c9c99ceac05b6262ee053441c7e55552ffe526bad8f83ff4640",
	}

	var wantPubKeys [][]byte

	wantPubKeyType := ScriptMultiSig
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushInt64(4)
	for _, v := range pubKeysData {
		testScript.PushSingleData(hexToBytes(v))
	}
	testScript.PushInt64(3)
	testScript.PushOpCode(OP_CHECKMULTISIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_MultiSig_OpCodesLenError(t *testing.T) {
	var wantPubKeys [][]byte

	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	testScript.PushInt64(2)
	testScript.PushInt64(3)
	testScript.PushOpCode(OP_CHECKMULTISIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_Error(t *testing.T) {
	pubKeysData := []string{
		"0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		"038282263212c609d9ea2a6e3e172de238d8c39cabd5ac1ca10646e23fd5f51508",
		"03363d90d447b00c9c99ceac05b6262ee053441c7e55552ffe526bad8f83ff4640",
	}

	var wantPubKeys [][]byte

	wantPubKeyType := ScriptNonStandard
	wantPubKeys = nil
	wantIsStandard := false

	testScript := NewEmptyScript()
	for _, v := range pubKeysData {
		testScript.PushSingleData(hexToBytes(v))
	}
	testScript.PushInt64(3)
	testScript.PushOpCode(OP_CHECKMULTISIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubKeyType, pubKeyType)
	assert.Equal(t, wantPubKeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_CheckScriptSigStandard_ScriptPubKeyHash(t *testing.T) {
	wantPubkeyType := ScriptPubkeyHash
	wantPubkeys := [][]byte{hexToBytes(
		"23b0ad3477f2178bc0b3eed26e4e6316f4e83aa1")}
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushOpCode(OP_DUP)
	testScript.PushOpCode(OP_HASH160)
	testScript.PushData(hexToBytes("14"))
	testScript.PushData(hexToBytes("23b0ad3477f2178bc0b3eed26e4e6316f4e83aa1"))
	testScript.PushOpCode(OP_EQUALVERIFY)
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubkeyType, pubKeyType)
	assert.Equal(t, wantPubkeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)
}

func TestScript_IsStandardScriptPubKey_ScriptPubkey(t *testing.T) {
	wantPubkeyType := ScriptPubkey
	wantPubkeys := [][]byte{hexToBytes(
		"0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8")}
	wantIsStandard := true

	testScript := NewEmptyScript()
	testScript.PushData(hexToBytes("41"))
	testScript.PushData(hexToBytes("0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"))
	testScript.PushOpCode(OP_CHECKSIG)

	pubKeyType, pubKeys, isStandard := testScript.IsStandardScriptPubKey()

	assert.Equal(t, wantPubkeyType, pubKeyType)
	assert.Equal(t, wantPubkeys, pubKeys)
	assert.Equal(t, wantIsStandard, isStandard)

}

//func ()  {
//
//}

func TestScript_PushInt64(t *testing.T) {
	tests := []struct {
		in          int64
		want        error
		wantData    []byte
		description string
	}{
		{-2, nil, []byte{0x01, 0x82}, "Input data is -2"},
		{-1, nil, []byte{0x4f}, "Input data is -1"},
		{0, nil, []byte{0x00}, "Input data is 0"},
		{1, nil, []byte{0x51}, "Input data is 1"},
		{16, nil, []byte{0x60}, "Input data is 16"},
		{17, nil, []byte{0x01, 0x11}, "Input data is 17"},
	}

	for _, v := range tests {
		value := v

		bScript := NewEmptyScript()
		result := bScript.PushInt64(value.in)
		assert.Equal(t, value.want, result, value.description)

		data := bScript.GetData()
		assert.Equal(t, value.wantData, data, value.description)
	}
}

func TestScript_PushSingleData(t *testing.T) {
	tests := []struct {
		in             []byte
		wantScriptData []byte
		wantError      error
		description    string
	}{
		{nil, []byte{0x00}, nil, "Nil data to test."},
		{
			bytes.Repeat([]byte{0x00}, OP_PUSHDATA1),
			combineBytes([]byte{0x4c, 0x4c}, bytes.Repeat([]byte{0x00}, 0x4c)),
			nil,
			"Push 0X4c data to test.",
		},
		{
			bytes.Repeat([]byte{0x00}, 0xff),
			combineBytes([]byte{0x4c, 0xff}, bytes.Repeat([]byte{0x00}, 0xff)),
			nil,
			"Push 0xff data to test.",
		},
		{
			bytes.Repeat([]byte{0x00}, 0xffff),
			combineBytes([]byte{0x4d, 0xff, 0xff}, bytes.Repeat([]byte{0x00}, 0xffff)),
			nil,
			"Push 0xffff data to test.",
		},
		{
			bytes.Repeat([]byte{0x00}, 0x010000),
			combineBytes([]byte{0x4e, 0x00, 0x00, 0x01, 0x00}, bytes.Repeat([]byte{0x00}, 0x010000)),
			nil,
			"Push 0x010000 data to test.",
		},
	}

	for _, v := range tests {
		value := v

		nullScript := NewEmptyScript()
		err := nullScript.PushSingleData(value.in)
		assert.Equal(t, value.wantError, err, value.description)
		data := nullScript.Bytes()
		assert.Equal(t, hex.EncodeToString(value.wantScriptData), hex.EncodeToString(data), value.description)
	}
}

func combineBytes(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte{})
}

func TestScript_PushMultData(t *testing.T) {
	tests := []struct {
		in             [][]byte
		wantScriptData []byte
		wantError      error
		description    string
	}{
		// Single data
		{nil, []byte{}, nil, "Nil data to test."},
		{[][]byte{{0x00}, {0x00}, {0x00}}, []byte{0x01, 0x00, 0x01, 0x00, 0x01, 0x00}, nil, "Less 0x4c data to test."},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, OP_PUSHDATA1)},
			combineBytes([]byte{0x4c, 0x4c}, bytes.Repeat([]byte{0x00}, 0x4c)),
			nil,
			"Push 0X4c data to test.",
		},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, 0xff)},
			combineBytes([]byte{0x4c, 0xff}, bytes.Repeat([]byte{0x00}, 0xff)),
			nil,
			"Push 0xff data to test.",
		},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, 0xffff)},
			combineBytes([]byte{0x4d, 0xff, 0xff}, bytes.Repeat([]byte{0x00}, 0xffff)),
			nil,
			"Push 0xffff data to test.",
		},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, 0x010000)},
			combineBytes([]byte{0x4e, 0x00, 0x00, 0x01, 0x00}, bytes.Repeat([]byte{0x00}, 0x010000)),
			nil,
			"Push 0x010000 data to test.",
		},

		// Multi data
		{[][]byte{nil, nil, nil}, []byte{0x00, 0x00, 0x00}, nil, "Multi Nil data to test."},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, OP_PUSHDATA1), bytes.Repeat([]byte{0x00}, OP_PUSHDATA1)},
			combineBytes([]byte{0x4c, 0x4c}, bytes.Repeat([]byte{0x00}, 0x4c),
				[]byte{0x4c, 0x4c}, bytes.Repeat([]byte{0x00}, 0x4c)),
			nil,
			"Multi Push 0X4c data to test.",
		},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, 0xff), bytes.Repeat([]byte{0x00}, 0xff)},
			combineBytes([]byte{0x4c, 0xff}, bytes.Repeat([]byte{0x00}, 0xff),
				[]byte{0x4c, 0xff}, bytes.Repeat([]byte{0x00}, 0xff)),
			nil,
			"Multi Push 0xff data to test.",
		},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, 0x1fff), bytes.Repeat([]byte{0x00}, 0x1fff)},
			combineBytes([]byte{0x4d, 0xff, 0x1f}, bytes.Repeat([]byte{0x00}, 0x1fff),
				[]byte{0x4d, 0xff, 0x1f}, bytes.Repeat([]byte{0x00}, 0x1fff)),
			nil,
			"Multi Push 0xffff data to test.",
		},
		{
			[][]byte{bytes.Repeat([]byte{0x00}, 0x010000), bytes.Repeat([]byte{0x00}, 0x010000)},
			combineBytes([]byte{0x4e, 0x00, 0x00, 0x01, 0x00}, bytes.Repeat([]byte{0x00}, 0x010000),
				[]byte{0x4e, 0x00, 0x00, 0x01, 0x00}, bytes.Repeat([]byte{0x00}, 0x010000)),
			nil,
			"Push 0x010000 data to test.",
		},
	}

	for _, v := range tests {
		value := v

		nullScript := NewEmptyScript()
		err := nullScript.PushMultData(value.in)
		assert.Equal(t, value.wantError, err, value.description)
		data := nullScript.Bytes()
		assert.Equal(t, value.wantScriptData, data, value.description)
	}
}

/**
 * IsValidSignatureEncoding  A canonical signature exists of: <30> <total len> <02> <len R> <R> <02> <len S> <S> <hashtype>
 * Where R and S are not negative (their first byte has its highest bit not set), and not
 * excessively padded (do not start with a 0 byte, unless an otherwise negative number follows,
 * in which case a single 0 byte is necessary and even required).
 *
 * See https://bitcointalk.org/index.php?topic=8392.msg127623#msg127623
 *
 * This function is consensus-critical since BIP66.
 */

func TestCheckSignatureEncoding(t *testing.T) {
	validSig := hexToBytes("3045022100d83c96e2656d8c91bf508c4dda68e13f6ea55cfd728e9f55d841d9e32d9325d302201673c42ba6b6546bda1fa0e072c5a423cb02d156406c8a5b59310aa86cab4af701")
	notValidSig := hexToBytes("3145022100d83c96e2656d8c91bf508c4dda68e13f6ea55cfd728e9f55d841d9e32d9325d302201673c42ba6b6546bda1fa0e072c5a423cb02d156406c8a5b59310aa86cab4af701")
	//errSig := hexToBytes("3045022100d83c96e2656d8c91bf508c4dda68e13f6ea55cfd728e9f55d841d9e32d9325d30221673c42ba6b6546bda1fa0e072c5a423cb02d156406c8a5b59310aa86cab4af701")
	notDefinedHashTypeSig := hexToBytes("3045022100d83c96e2656d8c91bf508c4dda68e13f6ea55cfd728e9f55d841d9e32d9325d302201673c42ba6b6546bda1fa0e072c5a423cb02d156406c8a5b59310aa86cab4af700")
	validSigWithSigHashForkID := hexToBytes("3045022100d83c96e2656d8c91bf508c4dda68e13f6ea55cfd728e9f55d841d9e32d9325d302201673c42ba6b6546bda1fa0e072c5a423cb02d156406c8a5b59310aa86cab4af741")

	tests := []struct {
		vchSig      []byte
		flags       uint32
		want        error
		description string
	}{
		{nil, 0, nil,
			"Non vchSig test, should return nil"},
		{
			notValidSig,
			ScriptVerifyDersig,
			errcode.New(errcode.ScriptErrSigDer),
			"Signature is not valid and flag is ScriptVerifyDersig, should return error.",
		},
		{
			notDefinedHashTypeSig,
			ScriptVerifyStrictEnc,
			errcode.New(errcode.ScriptErrSigHashType),
			"Signature is valid but hashtype is not defined, and flag is ScriptVerifyStrictEnc, should return error.",
		},
		{
			validSigWithSigHashForkID,
			ScriptVerifyStrictEnc,
			errcode.New(errcode.ScriptErrIllegalForkID),
			"Signature with SigHashForkID, should return illegal forkID error.",
		},
		{
			validSig,
			ScriptEnableSigHashForkID | ScriptVerifyStrictEnc,
			errcode.New(errcode.ScriptErrMustUseForkID),
			"Signature is valid, flag is ScriptEnableSigHashForkID xor ScriptVerifyStrictEnc, should return error",
		},
		{
			validSig,
			0,
			nil,
			"Without error, return nil",
		},
	}

	for _, v := range tests {
		value := v

		err := CheckTransactionSignatureEncoding(value.vchSig, value.flags)
		assert.Equal(t, err, value.want, value.description)
	}
}

func TestCheckPubKeyEncoding(t *testing.T) {
	CompressPubKey := hexToBytes("0338131e766199b56abd45b07fec07b39b7143dacb2111551ba15207c9f20dca58")
	err := CheckPubKeyEncoding(CompressPubKey, ScriptVerifyStrictEnc)
	assert.NoError(t, err,
		"Without compress's pubKey with ScriptVerifyStrictEnc check encoding error.")

	NoCompressPubkey := hexToBytes("04c4f74f58fe4b365037b79ed89d517db597cde8e375aad7cd3e173e887ac4939095b84d6d7c732921ec96df868c9d52dc645c3ab0dbe796af805f65cea8b9062d")
	err = CheckPubKeyEncoding(NoCompressPubkey, ScriptVerifyCompressedPubkeyType)
	assert.Equal(t, err, errcode.New(errcode.ScriptErrNonCompressedPubKey),
		"Without compress's pubKey with ScriptVerifyCompressedPubkeyType check encoding error.")

	err = CheckPubKeyEncoding(CompressPubKey[:len(CompressPubKey)-1], ScriptVerifyStrictEnc)
	assert.Equal(t, err, errcode.New(errcode.ScriptErrPubKeyType),
		"Without compress's pubKey with ScriptVerifyCompressedPubkeyType check encoding error.")

}
