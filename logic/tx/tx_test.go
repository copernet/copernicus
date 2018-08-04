package tx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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
	"github.com/copernet/copernicus/errcode"
)

var opMap map[string]byte

func init() {
	opMap = createName2OpCodeMap()
	crypto.InitSecp256()
}

func testVecF64ToUint32(f float64) uint32 {
	return uint32(int32(f))
}

type scriptErrChecker struct {
	// compare string in errtable with expected scriptErrorString
	errtable []string
}

func newScriptErrChecker() (*scriptErrChecker, error) {
	// Construction of errtable([]string) from errcode/scripterror.go
	var sec scriptErrChecker
	b, err := ioutil.ReadFile("../../errcode/scripterror.go")
	if err != nil {
		err = fmt.Errorf("scripterr.go not found")
		return nil, err
	}
	content := string(b)
	index := strings.Index(content, "const")
	if index == -1 {
		err = fmt.Errorf("scripterr.go does not contain \"const\"")
		return nil, err
	}
	content = content[index:]
	index = strings.Index(content, ")")
	if index == -1 {
		err = fmt.Errorf("scripterr.go const without \")\"")
		return nil, err
	}
	content = content[:index]
	contents := strings.Split(content, "\n")
	if contents[0] != "const (" {
		err = fmt.Errorf("scripterr.go \"const (\" should be the first line")
		return nil, err
	}
	if strings.HasSuffix(contents[1], " ScriptErr = ScriptErrorBase + iota") {
		contents[1] = strings.TrimSuffix(contents[1], " ScriptErr = ScriptErrorBase + iota")
	} else {
		err = fmt.Errorf("scripterr.go the first const should be declared ending with \" ScriptErr = ScriptErrorBase + iota\"");
		return nil, err
	}
	contents = contents[1:]
	sec.errtable = []string{}
	for _, line0 := range contents {
		line := strings.Replace(line0, "\n", "", -1)
		line = strings.Replace(line, "\t", "", -1)
		line = strings.Replace(line, " ", "", -1)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}
		sec.errtable = append(sec.errtable, line)
	}
	// got errtable
	return &sec, nil
}

func (sec *scriptErrChecker) check(err error, scriptErrorString string) error {
	var (
		actualErr      string
		actualErrUpper string
		perr           errcode.ProjectError
	)
	if err == nil {
		actualErr = "OK"
		actualErrUpper = "OK"
	} else {
		var ok bool
		if perr, ok = err.(errcode.ProjectError);
			!ok {
			err = fmt.Errorf("Error in converting err to ProjectErr")
			return err
		}
		actualErr = sec.errtable[perr.Code-errcode.ScriptErrorBase]

		if strings.HasPrefix(actualErr, "ScriptErr") {
			actualErrUpper = strings.TrimPrefix(actualErr, "ScriptErr")
		} else {
			err = fmt.Errorf("ScriptErr should begin with \"ScriptErr\"")
			return err
		}
		actualErrUpper = strings.TrimPrefix(actualErrUpper, "Invalid")
		actualErrUpper = strings.ToUpper(actualErrUpper)
	}
	scriptErrorStringUpper := strings.TrimPrefix(scriptErrorString, "INVALID")
	scriptErrorStringUpper = strings.Replace(scriptErrorStringUpper, "_", "", -1)
	if actualErrUpper != scriptErrorStringUpper {
		err = fmt.Errorf("expect: %v err: %v", scriptErrorString, actualErr)
		return err
	}
	return nil
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

	var sec *scriptErrChecker
	if sec, err = newScriptErrChecker(); err != nil {
		t.Errorf("error new scriptErrChecker")
		return
	}
	for i, itest := range tests {
		test, ok := itest.([]interface{})
		if ok {
			if err := doScriptJSONTest(t, test, *sec); err != nil {
				t.Errorf("%dth test error: itest:%#v", i, itest)
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
			name = strings.TrimPrefix(name, "OP_")
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

	for i, w := range words {
		if w == "" {
			continue
		}

		w = strings.TrimPrefix(w, "OP_")
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
			return nil, fmt.Errorf("parse script error when parse %dth with word(%s)", i, w)
		}
	}

	return res, nil
}

var scriptFlagMap = map[string]uint32{
	"NONE":                                  script.ScriptVerifyNone,
	"P2SH":                                  script.ScriptVerifyP2SH,
	"STRICTENC":                             script.ScriptVerifyStrictEnc,
	"DERSIG":                                script.ScriptVerifyDersig,
	"LOW_S":                                 script.ScriptVerifyLowS,
	"SIGPUSHONLY":                           script.ScriptVerifySigPushOnly,
	"MINIMALDATA":                           script.ScriptVerifyMinmalData,
	"NULLDUMMY":                             script.ScriptVerifyNullDummy,
	"DISCOURAGE_UPGRADABLE_NOPS":            script.ScriptVerifyDiscourageUpgradableNops,
	"CLEANSTACK":                            script.ScriptVerifyCleanStack,
	"MINIMALIF":                             script.ScriptVerifyMinimalIf,
	"NULLFAIL":                              script.ScriptVerifyNullFail,
	"CHECKLOCKTIMEVERIFY":                   script.ScriptVerifyCheckLockTimeVerify,
	"CHECKSEQUENCEVERIFY":                   script.ScriptVerifyCheckSequenceVerify,
	"DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM": script.ScriptVerifyDiscourageUpgradableWitnessProgram,
	"COMPRESSED_PUBKEYTYPE":                 script.ScriptVerifyCompressedPubkeyType,
	"SIGHASH_FORKID":                        script.ScriptEnableSigHashForkID,
	"REPLAY_PROTECTION":                     script.ScriptEnableReplayProtection,
	"MONOLITH_OPCODES":                      script.ScriptEnableMonolithOpcodes,
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

func doScriptJSONTest(t *testing.T, itest []interface{}, sec scriptErrChecker) (err error) {
	if len(itest) == 0 {
		err = fmt.Errorf("empty itest[] %#v", itest)
		t.Error(err)
		return err
	}

	var nValue int64
	if farr, ok := itest[0].([]interface{}); ok {
		nValue = int64(farr[0].(float64)) * 1e8
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

	err = verifyScript(trax, scriptSig, scriptPubKey, 0, amount.Amount(nValue), flags)

	if err = sec.check(err, scriptErrorString); err != nil {
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

		nIn := int(testVecF64ToUint32(test[2].(float64)))
		hashType := testVecF64ToUint32(test[3].(float64))

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
			amount.Amount(0), script.ScriptEnableSigHashForkID)
		if err != nil {
			t.Errorf("verify error for test %d", i)
			continue
		}
		if !bytes.Equal([]byte(hash[:]), shreg) {
			t.Fatalf("TestCalcSignatureHash failed test #%d: "+
				"Signature hash mismatch. %v,  hash: %x", i, test, hash)
		}
	}
}

type scriptWithInputVal struct {
	inputVal int64
	pkScript []byte
}

func TestTxValidTests(t *testing.T) {
	file, err := ioutil.ReadFile("test_data/tx_valid.json")
	if err != nil {
		t.Fatalf("TestTxValidTests: %v\n", err)
	}
	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestTxValidTests unmarshal err:%v\n", err)
	}

testloop:
	for i, test := range tests {
		inputs, ok := test[0].([]interface{})
		if !ok {
			continue
		}

		if len(test) != 3 {
			t.Errorf("bad test (bad length) %d: %v", i, test)
			continue
		}
		serializedhex, ok := test[1].(string)
		if !ok {
			t.Errorf("bad test (arg 2 not string) %d: %v", i, test)
			continue
		}
		serializedTx, err := hex.DecodeString(serializedhex)
		if err != nil {
			t.Errorf("bad test (arg 2 not hex %v) %d: %v", err, i,
				test)
			continue
		}
		newTx := tx.NewEmptyTx()
		err = newTx.Decode(bytes.NewReader(serializedTx))
		if err != nil {
			t.Errorf("bad test (arg 2 not msgtx %v) %d: %v", err,
				i, test)
			continue
		}

		verifyFlags, ok := test[2].(string)
		if !ok {
			t.Errorf("bad test (arg 3 not string) %d: %v", i, test)
			continue
		}

		flags, err := parseScriptFlag(verifyFlags)
		if err != nil {
			t.Errorf("bad test %d: %v", i, err)
			continue
		}

		prevOuts := make(map[outpoint.OutPoint]scriptWithInputVal)
		for j, iinput := range inputs {
			input, ok := iinput.([]interface{})
			if !ok {
				t.Errorf("bad test (%dth input not array)"+
					"%d: %v", j, i, test)
				continue
			}

			if len(input) < 3 || len(input) > 4 {
				t.Errorf("bad test (%dth input wrong length)"+
					"%d: %v", j, i, test)
				continue
			}

			previoustx, ok := input[0].(string)
			if !ok {
				t.Errorf("bad test (%dth input hash not string)"+
					"%d: %v", j, i, test)
				continue
			}

			prevhash := util.HashFromString(previoustx)
			idxf, ok := input[1].(float64)
			if !ok {
				t.Errorf("bad test (%dth input idx not number)"+
					"%d: %v", j, i, test)
				continue
			}
			idx := testVecF64ToUint32(idxf)

			oscript, ok := input[2].(string)
			if !ok {
				t.Errorf("bad test (%dth input script not "+
					"string) %d: %v", j, i, test)
				continue
			}

			script, err := parseScriptFrom(oscript, opMap)
			if err != nil {
				t.Errorf("bad test (%dth input script doesn't "+
					"parse %v) %d: %v, oscript is:%v", j, err, i, test, oscript)
				continue
			}

			var inputValue float64
			if len(input) == 4 {
				inputValue, ok = input[3].(float64)
				if !ok {
					t.Errorf("bad test (%dth input value not int) "+
						"%d: %v", j, i, test)
					continue
				}
			}

			v := scriptWithInputVal{
				inputVal: int64(inputValue),
				pkScript: script,
			}
			prevOuts[*outpoint.NewOutPoint(*prevhash, idx)] = v
		}

		for k, txin := range newTx.GetIns() {
			prevOut, ok := prevOuts[*txin.PreviousOutPoint]
			if !ok {
				t.Errorf("bad test (missing %dth input) %d:%v",
					k, i, test)
				continue testloop
			}

			pkscript := script.NewScriptRaw(prevOut.pkScript)

			err := verifyScript(newTx, txin.GetScriptSig(), pkscript, k, amount.Amount(prevOut.inputVal), flags)
			if err != nil {
				t.Errorf("verifyScript error: %v, %dth test, test=%v", err, i, test)
			}
		}
	}
}

func TestTxInvalidTests(t *testing.T) {
	file, err := ioutil.ReadFile("test_data/tx_invalid.json")
	if err != nil {
		t.Fatalf("TestTxInvalidTests: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestTxInvalidTests couldn't Unmarshal: %v\n", err)
	}

	// form is either:
	//   ["this is a comment "]
	// or:
	//   [[[previous hash, previous index, previous scriptPubKey]...,]
	//	serializedTransaction, verifyFlags]
testloop:
	for i, test := range tests {
		inputs, ok := test[0].([]interface{})
		if !ok {
			continue
		}

		if len(test) != 3 {
			t.Errorf("bad test (bad length) %d: %v", i, test)
			continue

		}
		serializedhex, ok := test[1].(string)
		if !ok {
			t.Errorf("bad test (arg 2 not string) %d: %v", i, test)
			continue
		}
		serializedTx, err := hex.DecodeString(serializedhex)
		if err != nil {
			t.Errorf("bad test (arg 2 not hex %v) %d: %v", err, i,
				test)
			continue
		}
		newTx := tx.NewEmptyTx()
		err = newTx.Decode(bytes.NewReader(serializedTx))
		if err != nil {
			t.Errorf("bad test (arg 2 not msgtx %v) %d: %v", err,
				i, test)
			continue
		}

		verifyFlags, ok := test[2].(string)
		if !ok {
			t.Errorf("bad test (arg 3 not string) %d: %v", i, test)
			continue
		}

		flags, err := parseScriptFlag(verifyFlags)
		if err != nil {
			t.Errorf("bad test %d: %v", i, err)
			continue
		}

		prevOuts := make(map[outpoint.OutPoint]scriptWithInputVal)
		for j, iinput := range inputs {
			input, ok := iinput.([]interface{})
			if !ok {
				t.Errorf("bad test (%dth input not array)"+
					"%d: %v", j, i, test)
				continue testloop
			}

			if len(input) < 3 || len(input) > 4 {
				t.Errorf("bad test (%dth input wrong length)"+
					"%d: %v", j, i, test)
				continue testloop
			}

			previoustx, ok := input[0].(string)
			if !ok {
				t.Errorf("bad test (%dth input hash not string)"+
					"%d: %v", j, i, test)
				continue testloop
			}

			prevhash := util.HashFromString(previoustx)
			if err != nil {
				t.Errorf("bad test (%dth input hash not hash %v)"+
					"%d: %v", j, err, i, test)
				continue testloop
			}

			idxf, ok := input[1].(float64)
			if !ok {
				t.Errorf("bad test (%dth input idx not number)"+
					"%d: %v", j, i, test)
				continue testloop
			}
			idx := testVecF64ToUint32(idxf)

			oscript, ok := input[2].(string)
			if !ok {
				t.Errorf("bad test (%dth input script not "+
					"string) %d: %v", j, i, test)
				continue testloop
			}

			//script, err := parseShortForm(oscript)
			script, err := parseScriptFrom(oscript, opMap)
			if err != nil {
				t.Errorf("bad test (%dth input script doesn't "+
					"parse %v) %d: %v", j, err, i, test)
				continue testloop
			}

			var inputValue float64
			if len(input) == 4 {
				inputValue, ok = input[3].(float64)
				if !ok {
					t.Errorf("bad test (%dth input value not int) "+
						"%d: %v", j, i, test)
					continue
				}
			}

			v := scriptWithInputVal{
				inputVal: int64(inputValue),
				pkScript: script,
			}
			prevOuts[*outpoint.NewOutPoint(*prevhash, idx)] = v
		}
		err = newTx.CheckRegularTransaction()
		if err != nil {
			continue
		}

		for k, txin := range newTx.GetIns() {
			prevOut, ok := prevOuts[*txin.PreviousOutPoint]
			if !ok {
				t.Errorf("bad test (missing %dth input) %d:%v",
					k, i, test)
				continue testloop
			}
			pkscript := script.NewScriptRaw(prevOut.pkScript)
			err := verifyScript(newTx, txin.GetScriptSig(), pkscript, k, amount.Amount(prevOut.inputVal), flags)
			if err != nil {
				continue testloop
			}
		}
		t.Errorf("test (%d:%v) succeeded when should fail",
			i, test)
	}
}
