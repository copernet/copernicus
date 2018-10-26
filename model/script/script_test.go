package script

import (
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

func TestScriptParseScript(t *testing.T) {
	p2shScript := NewScriptRaw(p2SHScript[:])
	if !p2shScript.IsPayToScriptHash() {
		t.Errorf("the script is P2SH should be true instead of false")
	}
	/*
		stk, err := p2shScript.ParseScript()
		if len(stk) != 3 || err != nil {
			t.Errorf("the P2SH script should have 3 ParsedOpCode struct instead of %d"+
				" The err : %v", len(stk), err)
		}

		for i, parseCode := range stk {
			if i == 0 {
				if stk[i].opValue != opcodes.OP_HASH160 || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0xa9 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 1 {
				if stk[i].opValue != 0x14 || len(stk[i].data) != 0x14 {
					t.Errorf("parse index %d value should be 0x14 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 2 {
				if stk[i].opValue != opcodes.OP_EQUAL || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0x87 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			}
		}

		num, err := p2shScript.GetSigOpCount()
		if err != nil || num != 0 {
			t.Errorf("Error : P2SH script have 0 OpCode instead of %d\n", num)
		}

		p2pkhScript := NewScriptRaw(p2PKHScript[:])
		if p2pkhScript.IsPayToScriptHash() {
			t.Error("script is P2PKH should be false instead of true")
		}

		stk, err = p2pkhScript.ParseScript()
		if len(stk) != 5 || err != nil {
			t.Errorf("the P2PKH script should have 5 ParsedOpCode struct instead of %d"+
				" The err : %v", len(stk), err)
		}

		for i, parseCode := range stk {
			if i == 0 {
				if stk[i].opValue != opcodes.OP_DUP || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0x76 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 1 {
				if stk[i].opValue != opcodes.OP_HASH160 || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0xa9 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 2 {
				if stk[i].opValue != 0x14 || len(stk[i].data) != 0x14 {
					t.Errorf("parse index %d value should be 0x14 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 3 {
				if stk[i].opValue != opcodes.OP_EQUALVERIFY || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0x88 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 4 {
				if stk[i].opValue != opcodes.OP_CHECKSIG || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0xac instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			}
		}

		num, err = p2pkhScript.GetSigOpCount()
		if err != nil || num != 1 {
			t.Errorf("Error : P2PKH script have 1 OpCode instead of %d\n", num)
		}
	*/

}

/*
func TestCScriptPushData(t *testing.T) {
	script := NewScriptRaw(make([]byte, 0))

	err := script.PushOpCode(opcodes.OP_HASH160)
	if err != nil {
		t.Error(err)
	}

	data := [...]byte{
		0x89, 0xAB, 0xCD, 0xEF, 0xAB,
		0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
		0xAB, 0xBA, 0xAB, 0xBA, 0xAB,
		0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
	}

	script.PushData(data[:])
	err = script.PushOpCode(opcodes.OP_EQUAL)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(script.bytes, p2SHScript[:]) {
		t.Errorf("push data and OpCode composition script %v "+
			"should be equal origin script data %v", script.bytes, p2SHScript)
	}
}

func TestScriptPushInt64(t *testing.T) {
	var script Script
	script.PushInt64(3)
	if len(script.bytes) != 1 {
		t.Error("func PushInt64() error: should have one element")
	}
	if script.bytes[0] != opcodes.OP_3 {
		t.Error("func PushInt64() error: the element should be 83 instead of : ", script.bytes[0])
	}

	script.bytes = make([]byte, 0)
	script.PushInt64(35)
	if len(script.bytes) != 1 {
		t.Error("func PushInt64() error: should have one element")
	}
	if script.bytes[0] != 35 {
		t.Error("func PushInt64() error: the element should be 35 instead of : ", script.bytes[0])
	}

	script.bytes = make([]byte, 0)
	script.PushInt64(235)
	if len(script.bytes) != 2 {
		t.Errorf("func PushInt64() error: should have two element instead of %d element", len(script.bytes))
	}
	if script.bytes[0] != 235 && script.bytes[1] != 0 {
		t.Errorf("func PushInt64() error: the element should be 235 instead of : %d", script.bytes[0])
	}
}
*/

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

		err := CheckSignatureEncoding(value.vchSig, value.flags)
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

func TestIsOpCodeDisabled(t *testing.T) {
	tests := []struct {
		in      byte
		flags   uint32
		want    bool
		message string
	}{
		{OP_INVERT, 0, true, "OP_INVERT"},
		{OP_2MUL, 0, true, "OP_2MUL"},
		{OP_2DIV, 0, true, "OP_2DIV"},
		{OP_MUL, 0, true, "OP_MUL"},
		{OP_LSHIFT, 0, true, "OP_LSHIFT"},
		{OP_RSHIFT, 0, true, "OP_RSHIFT"},

		{OP_CAT, ScriptEnableMonolithOpcodes, false, "OP_CAT"},
		{OP_SPLIT, ScriptEnableMonolithOpcodes, false, "OP_SPLIT"},
		{OP_AND, ScriptEnableMonolithOpcodes, false, "OP_AND"},
		{OP_OR, ScriptEnableMonolithOpcodes, false, "OP_OR"},
		{OP_XOR, ScriptEnableMonolithOpcodes, false, "OP_XOR"},
		{OP_NUM2BIN, ScriptEnableMonolithOpcodes, false, "OP_NUM2BIN"},
		{OP_BIN2NUM, ScriptEnableMonolithOpcodes, false, "OP_BIN2NUM"},
		{OP_DIV, ScriptEnableMonolithOpcodes, false, "OP_DIV"},
		{OP_MOD, 0, true, "OP_MOD, flag is 262144"},
		{OP_MOD, ScriptEnableMonolithOpcodes, false, "OP_MOD, flag is 0"},
		{OP_TRUE, 0, false, "OTHER MESSAGE"},
	}

	for _, v := range tests {
		value := v
		result := IsOpCodeDisabled(value.in, value.flags)
		assert.Equal(t, result, value.want, value.message)
	}
}
