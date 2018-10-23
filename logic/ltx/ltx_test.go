package ltx_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/logic/lmerkleroot"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/service"
	"github.com/copernet/copernicus/service/mining"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/btcsuite/btcutil"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/logic/lscript"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
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

func testVecF64ToUint32(f float64) uint32 {
	return uint32(int32(f))
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
			return 0, fmt.Errorf("not found script flag for name %s", w)
		}
	}
	return res, nil
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

func isAllDigitalNumber(n string) bool {
	for _, c := range n {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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

		shreg, err := util.GetHashBytesFromStr(test[4].(string))
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

			err := lscript.VerifyScript(newTx, txin.GetScriptSig(), pkscript, k, amount.Amount(prevOut.inputVal),
				flags, lscript.NewScriptRealChecker())
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
			err := lscript.VerifyScript(newTx, txin.GetScriptSig(), pkscript, k, amount.Amount(prevOut.inputVal),
				flags, lscript.NewScriptRealChecker())
			if err != nil {
				continue testloop
			}
		}
		t.Errorf("test (%d:%v) succeeded when should fail",
			i, test)
	}
}

func NewPrivateKey() crypto.PrivateKey {
	var keyBytes []byte
	for i := 0; i < 32; i++ {
		keyBytes = append(keyBytes, byte(rand.Uint32()%256))
	}
	return *crypto.PrivateKeyFromBytes(keyBytes)
}

// The struct Var contains some variable which testing using.
// keyMap is used to save the relation publicKeyHash and privateKey, k is publicKeyHash, v is privateKey.
type Var struct {
	priKeys       []crypto.PrivateKey
	pubKeys       []crypto.PublicKey
	prevHolder    tx.Tx
	spender       tx.Tx
	keyMap        map[string]*crypto.PrivateKey
	redeemScripts map[string]string
}

// Initial the test variable
func initVar() *Var {
	var v Var
	v.keyMap = make(map[string]*crypto.PrivateKey)
	v.redeemScripts = make(map[string]string)

	for i := 0; i < 3; i++ {
		privateKey := NewPrivateKey()
		v.priKeys = append(v.priKeys, privateKey)

		pubKey := *privateKey.PubKey()
		v.pubKeys = append(v.pubKeys, pubKey)

		pubKeyHash := string(util.Hash160(pubKey.ToBytes()))
		v.keyMap[pubKeyHash] = &privateKey
	}

	return &v
}

func checkError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

func check(v *Var, lockingScript *script.Script, t *testing.T) {

	empty := script.NewEmptyScript()
	realChecker := lscript.NewScriptRealChecker()
	standardScriptVerifyFlags := uint32(script.StandardScriptVerifyFlags)
	hashType := uint32(crypto.SigHashAll | crypto.SigHashForkID)

	err := ltx.SignRawTransaction(&v.spender, v.redeemScripts, v.keyMap, hashType)
	checkError(err, t)
	scriptSig := v.spender.GetIns()[0].GetScriptSig()

	combineSig, err := ltx.CombineSignature(
		&v.spender,
		lockingScript,
		scriptSig,
		empty,
		0, 0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	// swap the position of empty and scriptSig
	combineSig, err = ltx.CombineSignature(
		&v.spender,
		lockingScript,
		empty,
		scriptSig,
		0, 0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	// Signing again will give a different, valid signature:
	err = ltx.SignRawTransaction(&v.spender, v.redeemScripts, v.keyMap, hashType)
	checkError(err, t)
	scriptSig = v.spender.GetIns()[0].GetScriptSig()
	fmt.Println(hex.EncodeToString(scriptSig.GetData()))
	combineSig, err = ltx.CombineSignature(
		&v.prevHolder,
		lockingScript,
		scriptSig,
		empty,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}
}

// Test the CombineSignature function
func TestCombineSignature(t *testing.T) {
	v := initVar()

	// Initial the coin cache
	conf.Cfg = conf.InitConfig([]string{})
	config := utxo.UtxoConfig{
		Do: &db.DBOption{
			FilePath:  conf.Cfg.DataDir + "/chainstate",
			CacheSize: (1 << 20) * 8,
		},
	}
	utxo.InitUtxoLruTip(&config)

	coinsMap := utxo.NewEmptyCoinsMap()

	// Create a p2PKHLockingScript script
	p2PKHLockingScript := script.NewEmptyScript()
	p2PKHLockingScript.PushOpCode(opcodes.OP_DUP)
	p2PKHLockingScript.PushOpCode(opcodes.OP_HASH160)
	p2PKHLockingScript.PushSingleData(btcutil.Hash160(v.pubKeys[0].ToBytes()))
	p2PKHLockingScript.PushOpCode(opcodes.OP_EQUALVERIFY)
	p2PKHLockingScript.PushOpCode(opcodes.OP_CHECKSIG)

	// Add locking script to prevHolder
	v.prevHolder.AddTxOut(txout.NewTxOut(0, p2PKHLockingScript))

	v.spender.AddTxIn(
		txin.NewTxIn(
			outpoint.NewOutPoint(v.prevHolder.GetHash(), 0),
			script.NewEmptyScript(),
			script.SequenceFinal,
		),
	)

	coinsMap.AddCoin(
		v.spender.GetIns()[0].PreviousOutPoint,
		utxo.NewFreshCoin(v.prevHolder.GetTxOut(0), 1, false),
		true,
	)
	utxo.GetUtxoCacheInstance().UpdateCoins(coinsMap, &util.Hash{})

	// Some variable used in all function
	empty := script.NewEmptyScript()
	realChecker := lscript.NewScriptRealChecker()
	standardScriptVerifyFlags := uint32(script.StandardScriptVerifyFlags)

	combineSig, err := ltx.CombineSignature(
		&v.prevHolder,
		p2PKHLockingScript,
		empty,
		empty,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, empty) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	// Single signature case:
	check(v, p2PKHLockingScript, t)

	// P2SHLockingScript, single-signature case

	// Create a P2SHLockingScript script
	pubKey := script.NewEmptyScript()
	pubKey.PushSingleData(v.pubKeys[0].ToBytes())
	pubKey.PushOpCode(opcodes.OP_CHECKSIG)

	pubKeyHash160 := util.Hash160(pubKey.GetData())
	v.redeemScripts[string(pubKeyHash160)] = string(pubKey.GetData())

	P2SHLockingScript := script.NewEmptyScript()
	P2SHLockingScript.PushOpCode(opcodes.OP_HASH160)
	P2SHLockingScript.PushSingleData(pubKeyHash160)
	P2SHLockingScript.PushOpCode(opcodes.OP_EQUAL)

	v.prevHolder.GetTxOut(0).SetScriptPubKey(P2SHLockingScript)

	coinsMap = utxo.NewEmptyCoinsMap()
	coinsMap.AddCoin(
		v.spender.GetIns()[0].PreviousOutPoint,
		utxo.NewFreshCoin(v.prevHolder.GetTxOut(0), 1, false),
		true,
	)
	utxo.GetUtxoCacheInstance().UpdateCoins(coinsMap, &util.Hash{})

	v.spender.GetIns()[0].SetScriptSig(empty)
	check(v, P2SHLockingScript, t)

	hashType := uint32(crypto.SigHashAll | crypto.SigHashForkID)
	err = ltx.SignRawTransaction(&v.spender, v.redeemScripts, v.keyMap, hashType)
	checkError(err, t)
	scriptSig := v.spender.GetIns()[0].GetScriptSig()

	// dummy scriptSigCopy with placeHolder, should always choose
	// non-placeholder:
	dummyLockingScript := script.NewEmptyScript()
	dummyLockingScript.PushOpCode(opcodes.OP_0)
	dummyLockingScript.PushSingleData(pubKey.GetData())

	combineSig, err = ltx.CombineSignature(
		&v.prevHolder,
		P2SHLockingScript,
		dummyLockingScript,
		scriptSig,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	combineSig, err = ltx.CombineSignature(
		&v.prevHolder,
		P2SHLockingScript,
		scriptSig,
		dummyLockingScript,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	// Hardest case: Multisig 2-of-3
	// the stack like this: 2 << <pubKey1> << <pubKey2> << <pubKey3> << 3 << OP_CHECKMULTISIG
	MultiLockingScript := script.NewEmptyScript()
	MultiLockingScript.PushInt64(2)

	for i := 0; i < 3; i++ {
		MultiLockingScript.PushSingleData(v.pubKeys[i].ToBytes())
	}
	MultiLockingScript.PushInt64(3)

	MultiLockingScript.PushOpCode(opcodes.OP_CHECKMULTISIG)

	// Add multi sig script to out
	v.prevHolder.GetTxOut(0).SetScriptPubKey(MultiLockingScript)

	// Add tx to coinsMap and update coins
	coinsMap = utxo.NewEmptyCoinsMap()
	coinsMap.AddCoin(
		v.spender.GetIns()[0].PreviousOutPoint,
		utxo.NewFreshCoin(v.prevHolder.GetTxOut(0), 1, false),
		true,
	)
	utxo.GetUtxoCacheInstance().UpdateCoins(coinsMap, &util.Hash{})

	v.spender.GetIns()[0].SetScriptSig(empty)

	err = ltx.SignRawTransaction(&v.spender, v.redeemScripts, v.keyMap, hashType)
	checkError(err, t)
	scriptSig = v.spender.GetIns()[0].GetScriptSig()

	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		scriptSig,
		empty,
		0, 0,
		standardScriptVerifyFlags,
		realChecker,
	)
	if err != nil {
		t.Error(err, t)
	}
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	// swap the position of empty and scriptSig
	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		empty,
		scriptSig,
		0, 0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	//check(v, MultiLockingScript, t, true)

	// A couple of partially-signed versions:
	hash, err := tx.SignatureHash(
		&v.spender, MultiLockingScript, uint32(crypto.SigHashAll), 0, 0, 0)
	checkError(err, t)
	vchSig, err := v.priKeys[0].Sign(hash.GetCloneBytes())
	checkError(err, t)
	sig1 := bytes.Join([][]byte{vchSig.Serialize(), {byte(crypto.SigHashAll)}}, []byte{})

	hash, err = tx.SignatureHash(
		&v.spender, MultiLockingScript, uint32(crypto.SigHashNone), 0, 0, 0)
	checkError(err, t)
	vchSig, err = v.priKeys[1].Sign(hash.GetCloneBytes())
	checkError(err, t)
	sig2 := bytes.Join([][]byte{vchSig.Serialize(), {byte(crypto.SigHashNone)}}, []byte{})

	hash, err = tx.SignatureHash(
		&v.spender, MultiLockingScript, uint32(crypto.SigHashSingle), 0, 0, 0)
	checkError(err, t)
	vchSig, err = v.priKeys[2].Sign(hash.GetCloneBytes())
	checkError(err, t)
	sig3 := bytes.Join([][]byte{vchSig.Serialize(), {byte(crypto.SigHashSingle)}}, []byte{})

	// By sig1, sig2, sig3 generate some different combination to check
	partial1a := script.NewEmptyScript()
	partial1a.PushOpCode(opcodes.OP_0)
	partial1a.PushSingleData(sig1)
	partial1a.PushOpCode(opcodes.OP_0)

	partial1b := script.NewEmptyScript()
	partial1b.PushOpCode(opcodes.OP_0)
	partial1b.PushOpCode(opcodes.OP_0)
	partial1b.PushSingleData(sig1)

	partial2a := script.NewEmptyScript()
	partial2a.PushOpCode(opcodes.OP_0)
	partial2a.PushSingleData(sig2)

	partial2b := script.NewEmptyScript()
	partial2b.PushSingleData(sig2)
	partial2b.PushOpCode(opcodes.OP_0)

	partial3a := script.NewEmptyScript()
	partial3a.PushSingleData(sig3)

	partial3b := script.NewEmptyScript()
	partial3b.PushOpCode(opcodes.OP_0)
	partial3b.PushOpCode(opcodes.OP_0)
	partial3b.PushSingleData(sig3)

	partial3c := script.NewEmptyScript()
	partial3c.PushOpCode(opcodes.OP_0)
	partial3c.PushSingleData(sig3)
	partial3c.PushOpCode(opcodes.OP_0)

	complete12 := script.NewEmptyScript()
	complete12.PushOpCode(opcodes.OP_0)
	complete12.PushSingleData(sig1)
	complete12.PushSingleData(sig2)

	complete13 := script.NewEmptyScript()
	complete13.PushOpCode(opcodes.OP_0)
	complete13.PushSingleData(sig1)
	complete13.PushSingleData(sig3)

	complete23 := script.NewEmptyScript()
	complete23.PushOpCode(opcodes.OP_0)
	complete23.PushSingleData(sig2)
	complete23.PushSingleData(sig3)

	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial1a,
		partial1b,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, partial1a) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial1a,
		partial2a,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)

	checkError(err, t)
	if !reflect.DeepEqual(combineSig, complete12) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial2a,
		partial1a,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, complete12) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial1b,
		partial2b,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, complete12) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial3b,
		partial1b,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, complete13) {
		t.Error("SIGNATURE NOT EXPECTED")
	}
	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial2a,
		partial3a,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, complete23) {
		t.Error("SIGNATURE NOT EXPECTED")
	}
	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial3b,
		partial2b,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, complete23) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	combineSig, err = ltx.CombineSignature(
		&v.spender,
		MultiLockingScript,
		partial3b,
		partial3a,
		0,
		0,
		standardScriptVerifyFlags,
		realChecker,
	)
	checkError(err, t)
	if !reflect.DeepEqual(combineSig, partial3c) {
		t.Error("SIGNATURE NOT EXPECTED")
	}
}

func assertError(err error, code errcode.RejectCode, reason string, t *testing.T) {
	c, r, isReject := errcode.IsRejectCode(err)
	assert.True(t, isReject)
	assert.Equal(t, code, c)
	assert.Equal(t, reason, r)
}

func mainNetTx(version int32) *tx.Tx {
	// Random real transaction
	// (e2769b09e784f32f62ef849763d4f45b98e07ba658647343b915ff832b110436)

	txin := txin.NewTxIn(outpoint.NewOutPoint(util.Hash{
		0x6b, 0xff, 0x7f, 0xcd, 0x4f, 0x85, 0x65, 0xef,
		0x40, 0x6d, 0xd5, 0xd6, 0x3d, 0x4f, 0xf9, 0x4f,
		0x31, 0x8f, 0xe8, 0x20, 0x27, 0xfd, 0x4d, 0xc4,
		0x51, 0xb0, 0x44, 0x74, 0x01, 0x9f, 0x74, 0xb4,
	}, 0),
		script.NewScriptRaw([]byte{
			0x49, //pushdata opcode 73bytes
			0x30, //signature header
			0x46, //sig length
			0x02, //integer
			0x21, //R length 33bytes
			0x00,
			0xda, 0x0d, 0xc6, 0xae, 0xce, 0xfe, 0x1e, 0x06, 0xef, 0xdf, 0x05, 0x77,
			0x37, 0x57, 0xde, 0xb1, 0x68, 0x82, 0x09, 0x30, 0xe3, 0xb0, 0xd0, 0x3f,
			0x46, 0xf5, 0xfc, 0xf1, 0x50, 0xbf, 0x99, 0x0c,
			0x02, //integer
			0x21, //S Length 33bytes
			0x00, 0xd2,
			0x5b, 0x5c, 0x87, 0x04, 0x00, 0x76, 0xe4, 0xf2, 0x53, 0xf8, 0x26, 0x2e,
			0x76, 0x3e, 0x2d, 0xd5, 0x1e, 0x7f, 0xf0, 0xbe, 0x15, 0x77, 0x27, 0xc4,
			0xbc, 0x42, 0x80, 0x7f, 0x17, 0xbd, 0x39,
			0x01, //sighash code
			0x41, //pushdata opcode 65
			0x04, //prefix, uncompressed public keys are 64bytes ples a prefix of 04
			0xe6, 0xc2,
			0x6e, 0xf6, 0x7d, 0xc6, 0x10, 0xd2, 0xcd, 0x19, 0x24, 0x84, 0x78, 0x9a,
			0x6c, 0xf9, 0xae, 0xa9, 0x93, 0x0b, 0x94, 0x4b, 0x7e, 0x2d, 0xb5, 0x34,
			0x2b, 0x9d, 0x9e, 0x5b, 0x9f, 0xf7, 0x9a, 0xff, 0x9a, 0x2e, 0xe1, 0x97,
			0x8d, 0xd7, 0xfd, 0x01, 0xdf, 0xc5, 0x22, 0xee, 0x02, 0x28, 0x3d, 0x3b,
			0x06, 0xa9, 0xd0, 0x3a, 0xcf, 0x80, 0x96, 0x96, 0x8d, 0x7d, 0xbb, 0x0f,
			0x91, 0x78}),
		0xffffffff)

	return newTestTx(txin, 0, version)
}

func newTestTx(txin *txin.TxIn, locktime uint32, version int32) *tx.Tx {
	// Random real transaction
	// (e2769b09e784f32f62ef849763d4f45b98e07ba658647343b915ff832b110436)
	txn := tx.NewTx(locktime, version)
	txn.AddTxIn(txin)

	txOuts := []*txout.TxOut{
		txout.NewTxOut(0x0e94a78b, script.NewScriptRaw([]byte{
			0x76, // OP_DUP
			0xa9, // OP_HASH160
			0x14, // length
			0xba, 0xde, 0xec, 0xfd, 0xef, 0x05, 0x07, 0x24, 0x7f, 0xc8,
			0xf7, 0x42, 0x41, 0xd7, 0x3b, 0xc0, 0x39, 0x97, 0x2d, 0x7b,
			0x88, // OP_EQUALVERIFY
			0xac, // OP_CHECKSIG

		})),
		txout.NewTxOut(0x02a89440, script.NewScriptRaw([]byte{
			0x76, // OP_DUP
			0xa9, // OP_HASH160
			0x14, // length
			0xc1, 0x09, 0x32, 0x48, 0x3f, 0xec, 0x93, 0xed, 0x51, 0xf5,
			0xfe, 0x95, 0xe7, 0x25, 0x59, 0xf2, 0xcc, 0x70, 0x43, 0xf9,
			0x88, // OP_EQUALVERIFY
			0xac, // OP_CHECKSIG
		})),
	}

	txn.AddTxOut(txOuts[0])
	txn.AddTxOut(txOuts[1])
	return txn
}

func givenDustRelayFeeLimits(minRelayFee int64) {
	if conf.Cfg == nil {
		conf.Cfg = &conf.Configuration{}
	}
	conf.Cfg.TxOut.DustRelayFee = minRelayFee
}

func Test_coinbase_tx_should_not_be_accepted_into_mempool(t *testing.T) {
	txn := tx.NewGenesisCoinbaseTx()

	_, err := ltx.CheckTxBeforeAcceptToMemPool(txn)

	assertError(err, errcode.RejectInvalid, "bad-tx-coinbase", t)
}

func Test_non_standard_tx_should_not_be_accepted_into_mempool(t *testing.T) {
	model.ActiveNetParams.RequireStandard = true
	txnWithInvalidVersion := mainNetTx(0)

	_, err := ltx.CheckTxBeforeAcceptToMemPool(txnWithInvalidVersion)
	assertError(err, errcode.RejectNonstandard, "version", t)
}

func Test_dust_tx_should_NOT_be_accepted_into_mempool(t *testing.T) {
	txn := mainNetTx(1)

	givenDustRelayFeeLimits(int64(txn.GetValueOut() - 1))

	_, err := ltx.CheckTxBeforeAcceptToMemPool(txn)
	assertError(err, errcode.RejectNonstandard, "dust", t)
}

func TestMain(m *testing.M) {
	conf.Cfg = conf.InitConfig([]string{})
	ltx.ScriptVerifyInit()
	os.Exit(m.Run())
}

func initTestEnv() (func(), error) {
	conf.Cfg = conf.InitConfig([]string{})

	unitTestDataDirPath, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		return nil, err
	}

	model.SetRegTestParams()

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	chain.InitGlobalChain()

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	persist.InitPersistGlobal()

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()

	lchain.InitGenesisChain()

	mempool.InitMempool()
	crypto.InitSecp256()

	cleanup := func() {
		os.RemoveAll(unitTestDataDirPath)
		log.Debug("cleanup test dir: %s", unitTestDataDirPath)
		gChain := chain.GetInstance()
		*gChain = *chain.NewChain()
	}

	return cleanup, nil
}

const nInnerLoopCount = 0x100000

func generateBlocks(t *testing.T, scriptPubKey *script.Script, generate int, maxTries uint64) ([]*block.Block, error) {
	heightStart := chain.GetInstance().Height()
	heightEnd := heightStart + int32(generate)
	height := heightStart
	params := model.ActiveNetParams

	ret := make([]*block.Block, 0)
	var extraNonce uint
	for height < heightEnd {
		ba := mining.NewBlockAssembler(params)
		bt := ba.CreateNewBlock(scriptPubKey, mining.CoinbaseScriptSig(extraNonce))
		if bt == nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCInternalError,
				Message: "Could not create new block",
			}
		}

		bt.Block.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(bt.Block.Txs, nil)

		powCheck := pow.Pow{}
		bits := bt.Block.Header.Bits
		for maxTries > 0 && bt.Block.Header.Nonce < nInnerLoopCount {
			maxTries--
			bt.Block.Header.Nonce++
			hash := bt.Block.GetHash()
			if powCheck.CheckProofOfWork(&hash, bits, params) {
				break
			}
		}

		if maxTries == 0 {
			break
		}

		if bt.Block.Header.Nonce == nInnerLoopCount {
			extraNonce++
			continue
		}

		fNewBlock := false
		if service.ProcessNewBlock(bt.Block, true, &fNewBlock) != nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCInternalError,
				Message: "ProcessNewBlock, block not accepted",
			}
		}

		height++
		extraNonce = 0

		ret = append(ret, bt.Block)
	}

	return ret, nil
}

func generateTestBlocks(t *testing.T) []*block.Block {
	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_TRUE)
	blocks, _ := generateBlocks(t, pubKey, 101, 1000000)
	assert.Equal(t, 101, len(blocks))
	return blocks
}

func makeNormalTx(prevout util.Hash) *tx.Tx {
	outpoint := outpoint.NewOutPoint(prevout, 0)
	txin := txin.NewTxIn(outpoint, script.NewScriptRaw([]byte{}), 0xffffffff)
	txn := newTestTx(txin, 0, 0)
	return txn
}

func makeUniqueNormalTx(prevout util.Hash, variant uint32) *tx.Tx {
	outpoint := outpoint.NewOutPoint(prevout, 0)
	txin := txin.NewTxIn(outpoint, script.NewScriptRaw([]byte{}), variant)
	txn := newTestTx(txin, 0, 0)
	return txn
}

func makeNotFinalTx(prevout util.Hash) *tx.Tx {
	outpoint := outpoint.NewOutPoint(prevout, 0)
	txin := txin.NewTxIn(outpoint, script.NewScriptRaw([]byte{}), 0)
	txn := newTestTx(txin, 5000000, 1)
	//transaction with still locked height 5000000, and not equal 0xffffffff sequence, is not final tx
	return txn
}

func Test_not_final_tx_should_NOT_be_accepted_into_mempool(t *testing.T) {
	cleanup, err := initTestEnv()
	assert.NoError(t, err)
	defer cleanup()
	givenDustRelayFeeLimits(0)

	blocks := generateTestBlocks(t)
	txn := makeNotFinalTx(blocks[0].Txs[0].GetHash())

	_, err = ltx.CheckTxBeforeAcceptToMemPool(txn)
	assertError(err, errcode.RejectNonstandard, "bad-txns-nonfinal", t)
}

func Test_normal_tx_should_be_accepted_into_mempool(t *testing.T) {
	cleanup, err := initTestEnv()
	assert.NoError(t, err)
	defer cleanup()
	givenDustRelayFeeLimits(0)

	blocks := generateTestBlocks(t)
	txn := makeNormalTx(blocks[0].Txs[0].GetHash())

	_, err = ltx.CheckTxBeforeAcceptToMemPool(txn)
	assert.NoError(t, err)
}

func Test_already_exists_tx_should_NOT_be_accepted_into_mempool(t *testing.T) {
	cleanup, err := initTestEnv()
	assert.NoError(t, err)
	defer cleanup()
	givenDustRelayFeeLimits(0)

	blocks := generateTestBlocks(t)
	txn := makeNormalTx(blocks[0].Txs[0].GetHash())
	err = lmempool.AcceptTxToMemPool(txn)
	assert.NoError(t, err)

	_, err = ltx.CheckTxBeforeAcceptToMemPool(txn)
	assert.Equal(t, errcode.NewError(errcode.RejectAlreadyKnown, "txn-already-in-mempool"), err)
}

func Test_tx_with_already_spent_prev_outpoint_should_NOT_be_accepted_into_mempool(t *testing.T) {
	cleanup, err := initTestEnv()
	assert.NoError(t, err)
	defer cleanup()
	givenDustRelayFeeLimits(0)

	blocks := generateTestBlocks(t)
	txn := makeNormalTx(blocks[0].Txs[0].GetHash())
	fmt.Println(txn.GetHash())
	err = lmempool.AcceptTxToMemPool(txn)
	assert.NoError(t, err)

	newTx := makeUniqueNormalTx(blocks[0].Txs[0].GetHash(), 1)
	fmt.Println(newTx.GetHash())
	_, err = ltx.CheckTxBeforeAcceptToMemPool(newTx)
	assert.Equal(t, errcode.NewError(errcode.RejectConflict, "txn-mempool-conflict"), err)
}
