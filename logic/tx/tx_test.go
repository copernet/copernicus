package tx

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

func TestScriptJsonTests(t *testing.T) {
	data, err := ioutil.ReadFile("script_tests.json")
	if err != nil {
		t.Error(err)
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
				return
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

func parseScriptFrom(s string, opMap map[string]byte) ([]byte, error) {
	var res []byte
	words := strings.Split(s, " ")

	for _, w := range words {
		if w == "" {
			continue
		}
		//fmt.Printf("w=%v\n", w)

		if opcode, ok := opMap[w]; ok {
			res = append(res, opcode)
		} else if isAllDigitalNumber(w) {
			i64, _ := strconv.ParseInt(w, 10, 32)
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(i64))
			res = append(res, buf...)
		} else if strings.HasPrefix(w, "0x") || strings.HasPrefix(w, "0X") {
			bs, err := hex.DecodeString(w[2:])
			if err != nil {
				return nil, err
			}
			res = append(res, bs...)
		} else if len(w) > 2 && w[0] == '\'' && w[len(w)-1] == '\'' {
			res = append(res, []byte(w[1:len(w)-1])...)
		} else {
			return nil, errors.New("parse script error")
		}
	}

	return res, nil
}

// static std::map<std::string, uint32_t> mapFlagNames = {
//     {"NONE", SCRIPT_VERIFY_NONE},
//     {"P2SH", SCRIPT_VERIFY_P2SH},
//     {"STRICTENC", SCRIPT_VERIFY_STRICTENC},
//     {"DERSIG", SCRIPT_VERIFY_DERSIG},
//     {"LOW_S", SCRIPT_VERIFY_LOW_S},
//     {"SIGPUSHONLY", SCRIPT_VERIFY_SIGPUSHONLY},
//     {"MINIMALDATA", SCRIPT_VERIFY_MINIMALDATA},
//     {"NULLDUMMY", SCRIPT_VERIFY_NULLDUMMY},
//     {"DISCOURAGE_UPGRADABLE_NOPS", SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS},
//     {"CLEANSTACK", SCRIPT_VERIFY_CLEANSTACK},
//     {"MINIMALIF", SCRIPT_VERIFY_MINIMALIF},
//     {"NULLFAIL", SCRIPT_VERIFY_NULLFAIL},
//     {"CHECKLOCKTIMEVERIFY", SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY},
//     {"CHECKSEQUENCEVERIFY", SCRIPT_VERIFY_CHECKSEQUENCEVERIFY},
//     {"COMPRESSED_PUBKEYTYPE", SCRIPT_VERIFY_COMPRESSED_PUBKEYTYPE},
//     {"SIGHASH_FORKID", SCRIPT_ENABLE_SIGHASH_FORKID},
//     {"REPLAY_PROTECTION", SCRIPT_ENABLE_REPLAY_PROTECTION},
//     {"MONOLITH_OPCODES", SCRIPT_ENABLE_MONOLITH_OPCODES},
// };

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
		if flag, ok := scriptFlagMap[w]; ok {
			res |= flag
		} else {
			return 0, fmt.Errorf("not found scirpt flag for name %s", w)
		}
	}
	return res, nil
}

func doScriptJSONTest(t *testing.T, itest []interface{}) error {
	if len(itest) == 0 {
		err := errors.New("empty itest[]")
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
		err := errors.New("can not convert a test to a string slice")
		t.Error(err)
		return err
	}
	fmt.Printf("%#v\n", itest)
	opMap := createName2OpCodeMap()
	scriptSigString, scriptPubKeyString, scriptFlagString, scriptErrorString := test[0], test[1], test[2], test[3]
	fmt.Printf("sig(%s), pubkey(%s), flag(%s), err(%s)\n",
		scriptSigString, scriptPubKeyString, scriptFlagString, scriptErrorString)

	scriptSigBytes, err := parseScriptFrom(scriptSigString, opMap)
	if err != nil {
		t.Error(err)
		return err
	}

	scriptPubKeyBytes, err := parseScriptFrom(scriptPubKeyString, opMap)
	if err != nil {
		t.Error(err)
		return err
	}
	fmt.Printf("sig:%v pub:%v\n", scriptSigBytes, scriptPubKeyBytes)

	scriptSig := script.NewScriptRaw(scriptSigBytes)
	if scriptSig == nil {
		err = errors.New("NewScriptRaw error")
		t.Error(err)
		return err
	}
	scriptPubKey := script.NewScriptRaw(scriptPubKeyBytes)
	if scriptPubKey == nil {
		err = errors.New("NewScriptRaw error")
		t.Error(err)
		return err
	}

	flags, err := parseScriptFlag(scriptFlagString)
	if err != nil {
		t.Error(err)
		return err
	}
	trax := tx.NewTx(0, 1)
	trax.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(util.Hash{}, 0xffffffff), scriptSig, script.SequenceFinal))
	trax.AddTxOut(txout.NewTxOut(amount.Amount(nValue), script.NewScriptRaw([]byte{})))

	err = verifyScript(trax, scriptSig, scriptPubKey, 0, amount.Amount(nValue), flags)

	if !(scriptErrorString == "OK" && err == nil || scriptErrorString != "OK" && err != nil) {
		err = fmt.Errorf("expect error: scriptErrorString(%s) err(%v)", scriptErrorString, err)
		t.Error(err)
		return err
	}
	return nil
}
