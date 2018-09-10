package lscript

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"io/ioutil"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var opMap map[string]byte

func init() {
	opMap = createName2OpCodeMap()
	crypto.InitSecp256()
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

type scriptErrChecker struct {
	// compare string in errtable with expected scriptErrorString
	errtable []string
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
		err = fmt.Errorf("scripterr.go the first const should be declared ending with \" ScriptErr = ScriptErrorBase + iota\"")
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
		if perr, ok = err.(errcode.ProjectError); !ok {
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
		actualErrUpper = strings.TrimPrefix(actualErrUpper, "Sig")
		actualErrUpper = strings.ToUpper(actualErrUpper)
	}
	scriptErrorStringUpper := strings.TrimPrefix(scriptErrorString, "INVALID")
	scriptErrorStringUpper = strings.TrimPrefix(scriptErrorStringUpper, "SIG")
	scriptErrorStringUpper = strings.Replace(scriptErrorStringUpper, "_", "", -1)
	if actualErrUpper != scriptErrorStringUpper {
		err = fmt.Errorf("expect: %v err: %v", scriptErrorString, actualErr)
		return err
	}
	return nil
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
	preScriptSig.PushMultData(scriptNumBytes)

	pretx := tx.NewTx(0, 1)
	pretx.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(util.Hash{}, 0xffffffff),
		preScriptSig, script.SequenceFinal))
	pretx.AddTxOut(txout.NewTxOut(amount.Amount(nValue), scriptPubKey))

	trax := tx.NewTx(0, 1)
	trax.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(pretx.GetHash(), 0), scriptSig, script.SequenceFinal))
	trax.AddTxOut(txout.NewTxOut(amount.Amount(nValue), script.NewScriptRaw([]byte{})))

	err = VerifyScript(trax, scriptSig, scriptPubKey, 0, amount.Amount(nValue), flags, NewScriptRealChecker())

	if err = sec.check(err, scriptErrorString); err != nil {
		t.Error(err)
		return err
	}
	return nil
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

func signMultisig(scriptPubKey *script.Script, keys []crypto.PrivateKey, transaction *tx.Tx) *script.Script {
	hash, _ := tx.SignatureHash(transaction, scriptPubKey,
		uint32(txscript.SigHashAll), 0, amount.Amount(0), 0)
	result := script.NewEmptyScript()
	result.PushOpCode(opcodes.OP_0)
	for _, key := range keys {
		vchsig, _ := key.Sign(hash.GetCloneBytes())
		result.PushSingleData(bytes.Join([][]byte{vchsig.Serialize(),
			{byte(txscript.SigHashAll)}}, []byte{}))
	}
	return result
}

func NewPrivateKey() crypto.PrivateKey {
	var keyBytes []byte
	for i := 0; i < 32; i++ {
		keyBytes = append(keyBytes, byte(rand.Uint32()%256))
	}
	return *crypto.PrivateKeyFromBytes(keyBytes)
}

func TestScriptCHECKMULTISIG12(t *testing.T) {
	var flag uint32 = script.ScriptVerifyP2SH | script.ScriptVerifyStrictEnc
	key1 := NewPrivateKey()
	key2 := NewPrivateKey()
	key3 := NewPrivateKey()
	scriptPubKey12 := script.NewEmptyScript()
	scriptPubKey12.PushOpCode(opcodes.OP_1)
	scriptPubKey12.PushSingleData(key1.PubKey().ToBytes())
	scriptPubKey12.PushSingleData(key2.PubKey().ToBytes())
	scriptPubKey12.PushOpCode(opcodes.OP_2)
	scriptPubKey12.PushOpCode(opcodes.OP_CHECKMULTISIG)

	var txFrom12, txTo12 tx.Tx
	txFrom12.AddTxOut(txout.NewTxOut(0, scriptPubKey12))
	txTo12.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(txFrom12.GetHash(), 0),
		script.NewEmptyScript(), script.SequenceFinal))
	goodsig1 := signMultisig(scriptPubKey12, []crypto.PrivateKey{key1}, &txTo12)
	if err := VerifyScript(&txTo12, goodsig1, scriptPubKey12, 0, 0, flag, NewScriptRealChecker()); err != nil {
		t.Errorf("checkMultiSig fail, sk = key1, pk = key12")
	}

	txTo12.AddTxOut(txout.NewTxOut(0, script.NewEmptyScript()))
	if err := VerifyScript(&txTo12, goodsig1, scriptPubKey12, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key1, pk = key12, bug sig damaged")
	}

	goodsig2 := signMultisig(scriptPubKey12, []crypto.PrivateKey{key2}, &txTo12)
	if err := VerifyScript(&txTo12, goodsig2, scriptPubKey12, 0, 0, flag, NewScriptRealChecker()); err != nil {
		t.Errorf("checkMultiSig fail, sk = key2, pk = key12")
	}

	badsig1 := signMultisig(scriptPubKey12, []crypto.PrivateKey{key3}, &txTo12)
	if err := VerifyScript(&txTo12, badsig1, scriptPubKey12, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key3, pk = key12")
	}
}

func TestScriptCHECKMULTISIG23(t *testing.T) {
	var flag uint32 = script.ScriptVerifyP2SH | script.ScriptVerifyStrictEnc
	key1 := NewPrivateKey()
	key2 := NewPrivateKey()
	key3 := NewPrivateKey()
	key4 := NewPrivateKey()
	scriptPubKey23 := script.NewEmptyScript()
	scriptPubKey23.PushOpCode(opcodes.OP_2)
	scriptPubKey23.PushSingleData(key1.PubKey().ToBytes())
	scriptPubKey23.PushSingleData(key2.PubKey().ToBytes())
	scriptPubKey23.PushSingleData(key3.PubKey().ToBytes())
	scriptPubKey23.PushOpCode(opcodes.OP_3)
	scriptPubKey23.PushOpCode(opcodes.OP_CHECKMULTISIG)
	var txFrom23, txTo23 tx.Tx
	txFrom23.AddTxOut(txout.NewTxOut(0, scriptPubKey23))
	txTo23.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(txFrom23.GetHash(), 0),
		script.NewEmptyScript(), script.SequenceFinal))
	goodsig1 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key1, key2}, &txTo23)
	if err := VerifyScript(&txTo23, goodsig1, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err != nil {
		t.Errorf("checkMultiSig fail, sk = key12, pk = key123")
	}
	goodsig2 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key1, key3}, &txTo23)
	if err := VerifyScript(&txTo23, goodsig2, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err != nil {
		t.Errorf("checkMultiSig fail, sk = key13, pk = key123")
	}
	goodsig3 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key2, key3}, &txTo23)
	if err := VerifyScript(&txTo23, goodsig3, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err != nil {
		t.Errorf("checkMultiSig fail, sk = key23, pk = key123")
	}
	badsig1 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key2, key2}, &txTo23)
	if err := VerifyScript(&txTo23, badsig1, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key22, pk = key123")
	}
	badsig2 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key2, key1}, &txTo23)
	if err := VerifyScript(&txTo23, badsig2, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key21, pk = key123")
	}
	badsig3 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key3, key2}, &txTo23)
	if err := VerifyScript(&txTo23, badsig3, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key32, pk = key123")
	}
	badsig4 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key4, key2}, &txTo23)
	if err := VerifyScript(&txTo23, badsig4, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key42, pk = key123")
	}
	badsig5 := signMultisig(scriptPubKey23, []crypto.PrivateKey{key1, key4}, &txTo23)
	if err := VerifyScript(&txTo23, badsig5, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key14, pk = key123")
	}
	badsig6 := signMultisig(scriptPubKey23, []crypto.PrivateKey{}, &txTo23)
	if err := VerifyScript(&txTo23, badsig6, scriptPubKey23, 0, 0, flag, NewScriptRealChecker()); err == nil {
		t.Errorf("checkMultiSig should fail, sk = key{empty}, pk = key123")
	}
}

func TestScriptPushData(t *testing.T) {
	direct := []byte{1, 0x5a}
	pushdata1 := []byte{opcodes.OP_PUSHDATA1, 1, 0x5a}
	pushdata2 := []byte{opcodes.OP_PUSHDATA2, 1, 0, 0x5a}
	pushdata4 := []byte{opcodes.OP_PUSHDATA4, 1, 0, 0, 0, 0x5a}
	pushdatascript := [][]byte{pushdata1, pushdata2, pushdata4}
	directStack := util.NewStack()
	if err := EvalScript(directStack, script.NewScriptRaw(direct),
		nil, 0, 0, script.ScriptVerifyP2SH, NewScriptRealChecker()); err != nil {
		t.Error(err)
	}
	for i := 0; i < 3; i++ {
		pushdataStack := util.NewStack()
		if err := EvalScript(pushdataStack, script.NewScriptRaw(pushdatascript[i]),
			nil, 0, 0, script.ScriptVerifyP2SH, NewScriptRealChecker()); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(directStack, pushdataStack) {
			t.Errorf("ResultStack should be the same")
		}
	}
}

func TestScriptStandardPush(t *testing.T) {
	for i := 0; i < 67000; i++ {
		s := script.NewEmptyScript()
		s.PushInt64(int64(i))
		if !s.IsPushOnly() {
			t.Errorf("Number %d is not pure push.", i)
		}
		if VerifyScript(nil, s, script.NewScriptRaw([]byte{opcodes.OP_1}),
			0, 0, script.ScriptVerifyMinmalData, NewScriptRealChecker()) != nil {
			t.Errorf("Number %d push is not minimal data.", i)
		}
	}
	for i := 0; i <= script.MaxScriptElementSize; i++ {
		s := script.NewEmptyScript()
		s.PushSingleData(bytes.Repeat([]byte{'\111'}, i))
		if !s.IsPushOnly() {
			t.Errorf("Length %d is not pure push.", i)
		}
		if VerifyScript(nil, s, script.NewScriptRaw([]byte{opcodes.OP_1}),
			0, 0, script.ScriptVerifyMinmalData, NewScriptRealChecker()) != nil {
			t.Errorf("Length %d push is not minimal data.", i)
		}
	}
}

func TestIsPushOnlyOnInvalidScripts(t *testing.T) {
	s := script.NewEmptyScript()
	s.PushOpCode(1)
	if s.IsPushOnly() {
		t.Errorf("IsPushOnly should return false on invalid scripts")
	}
}
