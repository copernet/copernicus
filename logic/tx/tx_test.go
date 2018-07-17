package tx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

var opMap map[string]byte

func init() {
	opMap = createName2OpCodeMap()
}

func TestScriptJsonTests(t *testing.T) {
	data, err := ioutil.ReadFile("test_data/script_tests.json")
	if err != nil {
		t.Error(err)
		return
	}
	var tests []interface{}
	err = json.Unmarshal(data, &tests)
	if err != nil {
		t.Error(err)
	}

	for _, itest := range tests {
		test, ok := itest.([]interface{})
		if ok {
			if err := doScriptJSONTest(t, test); err != nil {
				t.Error(err)
				break
			}
		} else {
			t.Errorf("test is not []interface{}")
		}
	}
}

func interface2string(sli []interface{}) []string {
	var res []string
	for _, i := range sli {
		if s, ok := i.(string); ok {
			res = append(res, s)
		} else {
			return nil
		}
	}
	return res
}

func createName2OpCodeMap() map[string]byte {
	n2o := make(map[string]byte)
	for opc := 0; opc <= opcodes.OP_INVALIDOPCODE; opc++ {
		if name := opcodes.GetOpName(opc); name != "OP_UNKNOWN" {
			if strings.HasPrefix(name, "OP_") {
				name = name[3:]
			}
			n2o[name] = byte(opc)
		}
	}
	return n2o
}

func isAllDigitalNumber(n string) bool {
	for _, c := range n {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

type byteSlice []byte

func (b byteSlice) Less(i, j int) bool {
	return b[i] < b[j]
}

func (b byteSlice) Len() int {
	return len(b)
}

func (b byteSlice) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func reverseBytes(bs []byte) []byte {
	for i := 0; i < len(bs)/2; i++ {
		bs[i], bs[len(bs)-i] = bs[len(bs)-i], bs[i]
	}
	return bs
}

func ScriptNumSerialize(n int64) []byte {
	if n == 0 {
		return []byte{}
	}
	var res []byte
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}

	for n > 0 {
		res = append(res, byte(n&0xff))
		n >>= 8
	}

	if res[len(res)-1]&0x80 != 0 {
		if neg {
			res = append(res, 0x80)
		} else {
			res = append(res, 0)
		}
	} else if neg {
		res[len(res)-1] |= 0x80
	}

	return res
}

func appendData(res, w []byte) []byte {
	if len(w) < opcodes.OP_PUSHDATA1 {
		res = append(res, byte(len(w)))
	} else if len(w) <= 0xff {
		res = append(res, []byte{opcodes.OP_PUSHDATA1, byte(len(w))}...)
	} else if len(w) <= 0xffff {
		res = append(res, opcodes.OP_PUSHDATA2)
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(len(w)))
		res = append(res, buf...)
	} else {
		res = append(res, opcodes.OP_PUSHDATA4)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(len(w)))
		res = append(res, buf...)
	}

	res = append(res, w...)
	return res
}

func parseScriptFrom(s string, opMap map[string]byte) ([]byte, error) {
	var res []byte
	words := strings.Split(s, " ")

	for _, w := range words {
		if w == "" {
			continue
		}

		if opcode, ok := opMap[w]; ok {
			res = append(res, opcode)
		} else if isAllDigitalNumber(w) || strings.HasPrefix(w, "-") && isAllDigitalNumber(w[1:]) {
			n, _ := strconv.ParseInt(w, 10, 64)
			if n == -1 || (n >= 1 && n <= 16) {
				res = append(res, byte(n+(opcodes.OP_1-1)))
			} else if n == 0 {
				res = append(res, opcodes.OP_0)
			} else {
				res = appendData(res, ScriptNumSerialize(n))
			}
		} else if strings.HasPrefix(w, "0x") || strings.HasPrefix(w, "0X") {
			bs, err := hex.DecodeString(w[2:])
			if err != nil {
				return nil, err
			}

			res = append(res, bs...)
		} else if len(w) >= 2 && w[0] == '\'' && w[len(w)-1] == '\'' {
			w = w[1 : len(w)-1]
			res = appendData(res, []byte(w))
		} else {
			return nil, errors.New("parse script error")
		}
	}

	return res, nil
}

var scriptFlagMap = map[string]uint32{
	"NONE":        script.ScriptVerifyNone,
	"P2SH":        script.ScriptVerifyP2SH,
	"STRICTENC":   script.ScriptVerifyStrictEnc,
	"DERSIG":      script.ScriptVerifyDersig,
	"LOW_S":       script.ScriptVerifyLowS,
	"SIGPUSHONLY": script.ScriptVerifySigPushOnly,

	"MINIMALDATA": script.ScriptVerifyMinmalData,
	"NULLDUMMY":   script.ScriptVerifyNullDummy,

	"DISCOURAGE_UPGRADABLE_NOPS": script.ScriptVerifyDiscourageUpgradableNops,
	"CLEANSTACK":                 script.ScriptVerifyCleanStack,
	"MINIMALIF":                  script.ScriptVerifyMinimalIf,
	"NULLFAIL":                   script.ScriptVerifyNullFail,
	"CHECKLOCKTIMEVERIFY":        script.ScriptVerifyCheckLockTimeVerify,
	"CHECKSEQUENCEVERIFY":        script.ScriptVerifyCheckSequenceVerify,
	"COMPRESSED_PUBKEYTYPE":      script.ScriptVerifyCompressedPubkeyType,
	"SIGHASH_FORKID":             script.ScriptEnableSigHashForkId,
	"REPLAY_PROTECTION":          script.ScriptEnableReplayProtection,
	"MONOLITH_OPCODES":           script.ScriptEnableMonolithOpcodes,
}

func parseScriptFlag(s string) (uint32, error) {
	var res uint32
	words := strings.Split(s, ",")
	for _, w := range words {
		if w == "" {
			continue
		}
		if flag, ok := scriptFlagMap[w]; ok {
			res |= flag
		} else {
			return 0, fmt.Errorf("not found scirpt flag for name %s", w)
		}
	}
	return res, nil
}

func doScriptJSONTest(t *testing.T, itest []interface{}) (err error) {
	crypto.InitSecp256()

	if len(itest) == 0 {
		err = fmt.Errorf("empty itest[] %#v", itest)
		t.Error(err)
		return err
	}

	var nValue int64
	if farr, ok := itest[0].([]int64); ok {
		nValue = farr[0]
		itest = itest[1:]
	}
	if len(itest) < 4 {
		return nil
	}
	test := interface2string(itest)
	if test == nil {
		err = fmt.Errorf("can not convert a test to a string slice, itest[] %#v", itest)
		t.Error(err)
		return err
	}
	// fmt.Printf("%#v\n", itest)

	scriptSigString, scriptPubKeyString, scriptFlagString, scriptErrorString := test[0], test[1], test[2], test[3]
	// fmt.Printf("sig(%s), pubkey(%s), flag(%s), err(%s)\n",
	// 	scriptSigString, scriptPubKeyString, scriptFlagString, scriptErrorString)

	scriptSigBytes, err := parseScriptFrom(scriptSigString, opMap)
	if err != nil {
		t.Errorf("%v itest[] %v", err, itest)
		return err
	}

	scriptPubKeyBytes, err := parseScriptFrom(scriptPubKeyString, opMap)
	if err != nil {
		t.Errorf("%v itest[] %v", err, itest)
		return err
	}
	// fmt.Printf("sig:%v pub:%v\n", scriptSigBytes, scriptPubKeyBytes)

	scriptSig := script.NewScriptRaw(scriptSigBytes)
	if scriptSig == nil {
		t.Errorf("parse sig script err itest[] %#v", itest)
		return err
	}
	scriptPubKey := script.NewScriptRaw(scriptPubKeyBytes)
	if scriptPubKey == nil {
		t.Errorf("new script for pubkey err, itest[] %#v", itest)
		return err
	}

	flags, err := parseScriptFlag(scriptFlagString)
	if err != nil {
		t.Errorf("parse script flag err, itest[] %#v", itest)
		return err
	}
	scriptNumBytes := make([][]byte, 0)
	scriptNum := script.NewScriptNum(0)
	scriptNumBytes = append(scriptNumBytes, scriptNum.Serialize(), scriptNum.Serialize())
	preScriptSig := script.NewEmptyScript()
	preScriptSig.PushData(scriptNumBytes)

	pretx := tx.NewTx(0, 1)
	pretx.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(util.Hash{}, 0xffffffff),
		preScriptSig, script.SequenceFinal))
	pretx.AddTxOut(txout.NewTxOut(amount.Amount(nValue), scriptPubKey))

	trax := tx.NewTx(0, 1)
	trax.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(pretx.GetHash(), 0), scriptSig, script.SequenceFinal))
	trax.AddTxOut(txout.NewTxOut(amount.Amount(nValue), script.NewScriptRaw([]byte{})))

	//t.Errorf("Script BadOpcode flag: %v", scriptSig.GetBadOpCode())
	err = verifyScript(trax, scriptSig, scriptPubKey, 0, amount.Amount(nValue), flags)
	//t.Error(err)

	if !((scriptErrorString == "OK" && err == nil) || (scriptErrorString != "OK" && err != nil)) {
		err = fmt.Errorf("expect error: scriptErrorString(%s) err(%v), sig(%s), pubkey(%s), flag(%s), err(%s) itest[] %v",
			scriptErrorString, err, scriptSigString,
			scriptPubKeyString, scriptFlagString, scriptErrorString, itest)

		t.Error(err)
		return err
	}
	return nil
}

func TestSigHash(t *testing.T) {
	data, err := ioutil.ReadFile("test_data/sighash.json")
	if err != nil {
		t.Error(err)
		return
	}

	var tests [][]interface{}
	err = json.Unmarshal(data, &tests)
	if err != nil {
		t.Fatalf("TestCalcSignatureHash couldn't Unmarshal: %v\n",
			err)
	}

	for i, test := range tests[1:] {
		i++
		if len(test) < 1 {
			t.Fatalf("TestCalcSignatureHash: Test #%d has "+
				"wrong length.", i)
		}
		if len(test) == 1 {
			// comments
			continue
		}
		newTx := tx.NewEmptyTx()
		rawTx, _ := hex.DecodeString(test[0].(string))
		err := newTx.Decode(bytes.NewReader(rawTx))
		if err != nil {
			t.Errorf("failed to parse transaction for test %d", i)
			continue
		}

		subScript, _ := hex.DecodeString(test[1].(string))
		scriptPubKey := script.NewScriptRaw(subScript)

		nIn := int(test[2].(float64))
		hashType := uint32(test[3].(float64))

		shreg, err := util.DecodeHash(test[4].(string))
		if err != nil {
			t.Errorf("decode hash err for test %d: %v", i, err)
			continue
		}

		// hash := calcSignatureHash(parsedScript, hashType, &tx,
		// 	int(test[2].(float64)))

		// scriptPubKeyBytes, err := parseScriptFrom(test[1].(string), opMap)
		// if err != nil {
		// 	t.Errorf("parse script err for test %d, err:%v", i, err)
		// 	continue
		// }

		hash, err := tx.SignatureHash(newTx, scriptPubKey, hashType, nIn,
			amount.Amount(0), script.ScriptEnableSigHashForkId)
		if err != nil {
			t.Errorf("verify error for test %d", i)
			continue
		}
		if !bytes.Equal([]byte(hash[:]), shreg) {
			t.Fatalf("TestCalcSignatureHash failed test #%d: "+
				"Signature hash mismatch. %v", i, test)
		}
	}
}
