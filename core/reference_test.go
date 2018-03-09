package core

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

	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
)

var scriptErrorDesc = map[string]crypto.ScriptError{
	"OK":                                    crypto.SCRIPT_ERR_OK,
	"UNKNOWN_ERROR":                         crypto.SCRIPT_ERR_UNKNOWN_ERROR,
	"OP_RETURN":                             crypto.SCRIPT_ERR_OP_RETURN,
	"SCRIPT_SIZE":                           crypto.SCRIPT_ERR_SCRIPT_SIZE,
	"PUSH_SIZE":                             crypto.SCRIPT_ERR_PUSH_SIZE,
	"OP_COUNT":                              crypto.SCRIPT_ERR_OP_COUNT,
	"STACK_SIZE":                            crypto.SCRIPT_ERR_STACK_SIZE,
	"SIG_COUNT":                             crypto.SCRIPT_ERR_SIG_COUNT,
	"PUBKEY_COUNT":                          crypto.SCRIPT_ERR_PUBKEY_COUNT,
	"VERIFY":                                crypto.SCRIPT_ERR_VERIFY,
	"EQUALVERIFY":                           crypto.SCRIPT_ERR_EQUALVERIFY,
	"CHECKMULTISIGVERIFY":                   crypto.SCRIPT_ERR_CHECKMULTISIGVERIFY,
	"CHECKSIGVERIFY":                        crypto.SCRIPT_ERR_CHECKSIGVERIFY,
	"NUMEQUALVERIFY":                        crypto.SCRIPT_ERR_NUMEQUALVERIFY,
	"BAD_OPCODE":                            crypto.SCRIPT_ERR_BAD_OPCODE,
	"DISABLED_OPCODE":                       crypto.SCRIPT_ERR_DISABLED_OPCODE,
	"INVALID_STACK_OPERATION":               crypto.SCRIPT_ERR_INVALID_STACK_OPERATION,
	"INVALID_ALTSTACK_OPERATION":            crypto.SCRIPT_ERR_INVALID_ALTSTACK_OPERATION,
	"UNBALANCED_CONDITIONAL":                crypto.SCRIPT_ERR_UNBALANCED_CONDITIONAL,
	"NEGATIVE_LOCKTIME":                     crypto.SCRIPT_ERR_NEGATIVE_LOCKTIME,
	"UNSATISFIED_LOCKTIME":                  crypto.SCRIPT_ERR_UNSATISFIED_LOCKTIME,
	"SIG_HASHTYPE":                          crypto.SCRIPT_ERR_SIG_HASHTYPE,
	"SIG_DER":                               crypto.SCRIPT_ERR_SIG_DER,
	"MINIMALDATA":                           crypto.SCRIPT_ERR_MINIMALDATA,
	"SIG_PUSHONLY":                          crypto.SCRIPT_ERR_SIG_PUSHONLY,
	"SIG_HIGH_S":                            crypto.SCRIPT_ERR_SIG_HIGH_S,
	"SIG_NULLDUMMY":                         crypto.SCRIPT_ERR_SIG_NULLDUMMY,
	"PUBKEYTYPE":                            crypto.SCRIPT_ERR_PUBKEYTYPE,
	"CLEANSTACK":                            crypto.SCRIPT_ERR_CLEANSTACK,
	"MINIMALIF":                             crypto.SCRIPT_ERR_MINIMALIF,
	"NULLFAIL":                              crypto.SCRIPT_ERR_SIG_NULLFAIL,
	"DISCOURAGE_UPGRADABLE_NOPS":            crypto.SCRIPT_ERR_DISCOURAGE_UPGRADABLE_NOPS,
	"DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM": crypto.SCRIPT_ERR_DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM,
	"WITNESS_PROGRAM_WRONG_LENGTH":          crypto.SCRIPT_ERR_WITNESS_PROGRAM_WRONG_LENGTH,
	"WITNESS_PROGRAM_WITNESS_EMPTY":         crypto.SCRIPT_ERR_WITNESS_PROGRAM_WITNESS_EMPTY,
	"WITNESS_PROGRAM_MISMATCH":              crypto.SCRIPT_ERR_WITNESS_PROGRAM_MISMATCH,
	"WITNESS_MALLEATED":                     crypto.SCRIPT_ERR_WITNESS_MALLEATED,
	"WITNESS_MALLEATED_P2SH":                crypto.SCRIPT_ERR_WITNESS_MALLEATED_P2SH,
	"WITNESS_UNEXPECTED":                    crypto.SCRIPT_ERR_WITNESS_UNEXPECTED,
	"WITNESS_PUBKEYTYPE":                    crypto.SCRIPT_ERR_WITNESS_PUBKEYTYPE,
}

var mapFlagNames = map[string]uint32{
	"NONE":                                  crypto.SCRIPT_VERIFY_NONE,
	"P2SH":                                  crypto.SCRIPT_VERIFY_P2SH,
	"STRICTENC":                             crypto.SCRIPT_VERIFY_STRICTENC,
	"DERSIG":                                crypto.SCRIPT_VERIFY_DERSIG,
	"LOW_S":                                 crypto.SCRIPT_VERIFY_LOW_S,
	"SIGPUSHONLY":                           crypto.SCRIPT_VERIFY_SIGPUSHONLY,
	"MINIMALDATA":                           crypto.SCRIPT_VERIFY_MINIMALDATA,
	"NULLDUMMY":                             crypto.SCRIPT_VERIFY_NULLDUMMY,
	"DISCOURAGE_UPGRADABLE_NOPS":            crypto.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS,
	"CLEANSTACK":                            crypto.SCRIPT_VERIFY_CLEANSTACK,
	"MINIMALIF":                             crypto.SCRIPT_VERIFY_MINIMALIF,
	"NULLFAIL":                              crypto.SCRIPT_VERIFY_NULLFAIL,
	"CHECKLOCKTIMEVERIFY":                   crypto.SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY,
	"CHECKSEQUENCEVERIFY":                   crypto.SCRIPT_VERIFY_CHECKSEQUENCEVERIFY,
	"DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM": crypto.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM,
	"COMPRESSED_PUBKEYTYPE":                 crypto.SCRIPT_VERIFY_COMPRESSED_PUBKEYTYPE,
	"SIGHASH_FORKID":                        crypto.SCRIPT_ENABLE_SIGHASH_FORKID,
}

func genTestName(test []interface{}) (string, error) {
	// Account for any optional leading witness data.
	var witnessOffset int
	if _, ok := test[0].([]interface{}); ok {
		witnessOffset++
	}

	// In addition to the optional leading witness data, the test must
	// consist of at least a signature script, public key script, flags,
	// and expected error.  Finally, it may optionally contain a comment.
	if len(test) < witnessOffset+4 || len(test) > witnessOffset+5 {
		return "", fmt.Errorf("invalid test length %d", len(test))
	}

	// Use the comment for the test name if one is specified, otherwise,
	// construct the name based on the signature script, public key script,
	// and flags.
	var name string
	if len(test) == witnessOffset+5 {
		name = fmt.Sprintf("test (%s)", test[witnessOffset+4])
	} else { //len(test) == 4
		name = fmt.Sprintf("test ([%s, %s, %s])", test[witnessOffset],
			test[witnessOffset+1], test[witnessOffset+2])
	}

	return name, nil
}

// parse hex string into a []byte.
func parseHex(tok string) ([]byte, error) {
	if !strings.HasPrefix(tok, "0x") {
		return nil, errors.New("not a hex number")
	}
	return hex.DecodeString(tok[2:])
}

// shortFormOps holds a map of opcode names to values for use in short form
// parsing.  It is declared here so it only needs to be created once.
var shortFormOps map[string]byte

// parseShortForm parses a string as as used in the Bitcoin Core reference tests
// into the script it came from.
//
// The format used for these tests is pretty simple if ad-hoc:
//   - Opcodes other than the push opcodes and unknown are present as
//     either OP_NAME or just NAME
//   - Plain numbers are made into push operations
//   - Numbers beginning with 0x are inserted into the []byte as-is (so
//     0x14 is OP_DATA_20)
//   - Single quoted strings are pushed as data
//   - Anything else is an error
func parseShortForm(script string) ([]byte, error) {
	// Only create the short form opcode map once.
	//fmt.Printf("got script=%q\n", script)
	if shortFormOps == nil {
		shortFormOps = make(map[string]byte)
		for i := 0; i <= OP_NOP10; i++ {
			if i < OP_NOP && i != OP_RESERVED {
				continue
			}
			name := GetOpName(i)
			if name == "OP_UNKNOWN" {
				continue
			}
			shortFormOps[name] = byte(i)
			shortFormOps[strings.TrimPrefix(name, "OP_")] = byte(i)
		}
	}
	/*
		for k, v := range shortFormOps {
			fmt.Printf("k=%v,  v=%v\n", k, v)
		}
	*/
	// Split only does one separator so convert all \n and tab into  space.
	script = strings.Replace(script, "\n", " ", -1)
	script = strings.Replace(script, "\t", " ", -1)
	tokens := strings.Split(script, " ")
	scr := NewScriptRaw(nil)

	for _, tok := range tokens {
		if len(tok) == 0 {
			continue
		}
		// if parses as a plain number
		if num, err := strconv.ParseInt(tok, 10, 64); err == nil {
			//builder.AddInt64(num)
			scr.PushInt64(num)
			continue
		} else if bts, err := parseHex(tok); err == nil {
			// Concatenate the bytes manually since the test code
			// intentionally creates scripts that are too large and
			// would cause the builder to error otherwise.
			scr.bytes = append(scr.bytes, bts...)
		} else if len(tok) >= 2 &&
			tok[0] == '\'' && tok[len(tok)-1] == '\'' {
			scr.PushData([]byte(tok[1 : len(tok)-1]))
		} else if opcode, ok := shortFormOps[tok]; ok {
			scr.PushOpCode(int(opcode))
		} else {
			return nil, fmt.Errorf("bad token %q", tok)
		}

	}

	return scr.bytes, nil
}

// parseScriptFlags parses the provided flags string from the format used in the
// reference tests into ScriptFlags suitable for use in the script engine.
func parseScriptFlags(flagStr string) (uint32, error) {
	var flags uint32

	sFlags := strings.Split(flagStr, ",")
	for _, sFlag := range sFlags {
		flag, ok := mapFlagNames[sFlag]
		if !ok {
			return 0, fmt.Errorf("unknown verification flag: %s", sFlag)
		}
		flags |= flag
	}
	return flags, nil
}

// parseExpectedResult parses the provided expected result string into allowed
// script error codes.  An error is returned if the expected result string is
// not supported.
func parseExpectedResult(expected string) crypto.ScriptError {
	return scriptErrorDesc[expected]
}

// createSpendTx generates a basic spending transaction given the passed
// signature and public key scripts.
func createSpendingTx(sigScript, pkScript []byte) *Tx {
	coinbaseTx := NewTx()

	outPoint := NewOutPoint(utils.Hash{}, ^uint32(0))
	txIn := NewTxIn(outPoint, []byte{OP_0, OP_0})
	txOut := NewTxOut(0, pkScript)
	coinbaseTx.AddTxIn(txIn)
	coinbaseTx.AddTxOut(txOut)

	spendingTx := NewTx()
	coinbaseTxHash := coinbaseTx.TxHash()
	outPoint = NewOutPoint(coinbaseTxHash, 0)
	txIn = NewTxIn(outPoint, sigScript)
	txOut = NewTxOut(0, nil)
	spendingTx.AddTxIn(txIn)
	spendingTx.AddTxOut(txOut)

	return spendingTx
}

// testScripts ensures all of the passed script tests execute with the expected
// results with or without using a signature cache, as specified by the
// parameter.
func testScripts(t *testing.T, tests [][]interface{}, useSigCache bool) {
	for i, test := range tests {
		// "Format is: [[wit..., amount]?, scriptSig, scriptPubKey,
		//    flags, expected_scripterror, ... comments]"
		if i != 8 {
			continue
		}

		// Skip single line comments.
		if len(test) == 1 {
			continue
		}

		// Construct a name for the test based on the comment and test data.
		name, err := genTestName(test)
		if err != nil {
			t.Errorf("TestScripts: invalid test #%d: %v", i, err)
			continue
		}

		// When the first field of the test data is a slice it contains
		// witness data and everything else is offset by 1 as a result.
		witnessOffset := 0
		witnessData, ok := test[0].([]interface{})
		if ok {
			witnessOffset++
		}
		_ = witnessData // Unused for now until segwit code lands

		// Extract and parse the signature script from the test fields.
		scriptSigStr, ok := test[witnessOffset].(string)
		if !ok {
			t.Errorf("%s: signature script is not a string", name)
			continue
		}
		scriptSig, err := parseShortForm(scriptSigStr)
		if err != nil {
			t.Errorf("%s: can't parse signature script: %v", name,
				err)
			continue
		}
		t.Logf("scriptSig = %v, scriptSigStr : %s \n", scriptSig, scriptSigStr)

		// Extract and parse the public key script from the test fields.
		scriptPubKeyStr, ok := test[witnessOffset+1].(string)
		if !ok {
			t.Errorf("%s: public key script is not a string", name)
			continue
		}
		scriptPubKey, err := parseShortForm(scriptPubKeyStr)
		if err != nil {
			t.Errorf("%s: can't parse public key script: %v", name,
				err)
			continue
		}
		t.Logf("scriptPubKey = % 02x \n", scriptPubKey)

		flagsStr, ok := test[witnessOffset+2].(string)
		if !ok {
			t.Errorf("%s: flags field is not a string", name)
			continue
		}
		flags, err := parseScriptFlags(flagsStr)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		t.Logf("flags = %d\n", flags)

		resultStr, ok := test[witnessOffset+3].(string)
		if !ok {
			t.Errorf("%s: result field is not a string", name)
			continue
		}
		//code, ok := parseExpectedResult(resultStr)
		code, ok := scriptErrorDesc[resultStr]
		if !ok {
			t.Errorf("%s: %v", name, "not found")
			continue
		}

		tx := createSpendingTx(scriptSig, scriptPubKey)
		interpreter := Interpreter{
			stack: container.NewStack(),
		}

		result, err := interpreter.Verify(tx, 0, NewScriptRaw(scriptSig), NewScriptRaw(scriptPubKey), flags)

		if result && code != crypto.SCRIPT_ERR_OK {
			t.Errorf("%s failed to verify: %v", name, err)
			continue
		}
		if err != nil {
			errDesc, _ := err.(*crypto.ErrDesc)
			if errDesc.Code != code {
				t.Errorf("%s failed to verify, expect %v, but got %v", name, code, errDesc.Code)
				continue
			}
		}

		/*
			vm, err := NewEngine(scriptPubKey, tx, 0, flags, sigCache)
			if err == nil {
				err = vm.Execute()
			}

			// Ensure there were no errors when the expected result is OK.
			if resultStr == "OK" {
				if err != nil {
					t.Errorf("%s failed to execute: %v", name, err)
				}
				continue
			}

			// At this point an error was expected so ensure the result of
			// the execution matches it.
			success := false
			for _, code := range allowedErrorCodes {
				if IsErrorCode(err, code) {
					success = true
					break
				}
			}
			if !success {
				if serr, ok := err.(Error); ok {
					t.Errorf("%s: want error codes %v, got %v", name,
					allowedErrorCodes, serr.ErrorCode)
					continue
				}
				t.Errorf("%s: want error codes %v, got err: %v (%T)",
				name, allowedErrorCodes, err, err)
				continue
			}

		*/
	}
}

// TestScripts ensures all of the tests in script_tests.json execute with the
// expected results as defined in the test data.
func TestScripts(t *testing.T) {
	file, err := ioutil.ReadFile("../test/data/script_tests.json")
	if err != nil {
		t.Fatalf("TestScripts: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestScripts couldn't Unmarshal: %v", err)
	}

	//testScripts(t, tests, true)
	testScripts(t, tests, false)
}

// testVecF64ToUint32 properly handles conversion of float64s read from the JSON
// test data to unsigned 32-bit integers.  This is necessary because some of the
// test data uses -1 as a shortcut to mean max uint32 and direct conversion of a
// negative float to an unsigned int is implementation dependent and therefore
// doesn't result in the expected value on all platforms.  This function woks
// around that limitation by converting to a 32-bit signed integer first and
// then to a 32-bit unsigned integer which results in the expected behavior on
// all platforms.
func testVecF64ToUint32(f float64) uint32 {
	return uint32(int32(f))
}

// TestTxInvalidTests ensures all of the tests in tx_invalid.json fail as
// expected.
/*
func TestTxInvalidTests(t *testing.T) {
	file, err := ioutil.ReadFile("data/tx_invalid.json")
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

		tx, err := btcutil.NewTxFromBytes(serializedTx)
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

		flags, err := parseScriptFlags(verifyFlags)
		if err != nil {
			t.Errorf("bad test %d: %v", i, err)
			continue
		}

		prevOuts := make(map[wire.OutPoint][]byte)
		for j, iinput := range inputs {
			input, ok := iinput.([]interface{})
			if !ok {
				t.Errorf("bad test (%dth input not array)"+
				"%d: %v", j, i, test)
				continue testloop
			}

			if len(input) != 3 {
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

			prevhash, err := chainhash.NewHashFromStr(previoustx)
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

			script, err := parseShortForm(oscript)
			if err != nil {
				t.Errorf("bad test (%dth input script doesn't "+
				"parse %v) %d: %v", j, err, i, test)
				continue testloop
			}

			prevOuts[*wire.NewOutPoint(prevhash, idx)] = script
		}

		for k, txin := range tx.MsgTx().TxIn {
			pkScript, ok := prevOuts[txin.PreviousOutPoint]
			if !ok {
				t.Errorf("bad test (missing %dth input) %d:%v",
				k, i, test)
				continue testloop
			}
			// These are meant to fail, so as soon as the first
			// input fails the transaction has failed. (some of the
			// test txns have good inputs, too..
			vm, err := NewEngine(pkScript, tx.MsgTx(), k, flags, nil)
			if err != nil {
				continue testloop
			}

			err = vm.Execute()
			if err != nil {
				continue testloop
			}

		}
		t.Errorf("test (%d:%v) succeeded when should fail",
		i, test)
	}
}
*/

// TestTxValidTests ensures all of the tests in tx_valid.json pass as expected.
/*
func TestTxValidTests(t *testing.T) {
	file, err := ioutil.ReadFile("data/tx_valid.json")
	if err != nil {
		t.Fatalf("TestTxValidTests: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestTxValidTests couldn't Unmarshal: %v\n", err)
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

		tx, err := btcutil.NewTxFromBytes(serializedTx)
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

		flags, err := parseScriptFlags(verifyFlags)
		if err != nil {
			t.Errorf("bad test %d: %v", i, err)
			continue
		}

		prevOuts := make(map[wire.OutPoint][]byte)
		for j, iinput := range inputs {
			input, ok := iinput.([]interface{})
			if !ok {
				t.Errorf("bad test (%dth input not array)"+
				"%d: %v", j, i, test)
				continue
			}

			if len(input) != 3 {
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

			prevhash, err := chainhash.NewHashFromStr(previoustx)
			if err != nil {
				t.Errorf("bad test (%dth input hash not hash %v)"+
				"%d: %v", j, err, i, test)
				continue
			}

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

			script, err := parseShortForm(oscript)
			if err != nil {
				t.Errorf("bad test (%dth input script doesn't "+
				"parse %v) %d: %v", j, err, i, test)
				continue
			}

			prevOuts[*wire.NewOutPoint(prevhash, idx)] = script
		}

		for k, txin := range tx.MsgTx().TxIn {
			pkScript, ok := prevOuts[txin.PreviousOutPoint]
			if !ok {
				t.Errorf("bad test (missing %dth input) %d:%v",
				k, i, test)
				continue testloop
			}
			vm, err := NewEngine(pkScript, tx.MsgTx(), k, flags, nil)
			if err != nil {
				t.Errorf("test (%d:%v:%d) failed to create "+
				"script: %v", i, test, k, err)
				continue
			}

			err = vm.Execute()
			if err != nil {
				t.Errorf("test (%d:%v:%d) failed to execute: "+
				"%v", i, test, k, err)
				continue
			}
		}
	}
}
*/

// TestCalcSignatureHash runs the Bitcoin Core signature hash calculation tests
// in sighash.json.
// https://github.com/bitcoin/bitcoin/blob/master/src/test/data/sighash.json
/*
func TestCalcSignatureHash(t *testing.T) {
	file, err := ioutil.ReadFile("data/sighash.json")
	if err != nil {
		t.Fatalf("TestCalcSignatureHash: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestCalcSignatureHash couldn't Unmarshal: %v\n",
		err)
	}

	for i, test := range tests {
		if i == 0 {
			// Skip first line -- contains comments only.
			continue
		}
		if len(test) != 5 {
			t.Fatalf("TestCalcSignatureHash: Test #%d has "+
			"wrong length.", i)
		}
		var tx wire.MsgTx
		rawTx, _ := hex.DecodeString(test[0].(string))
		err := tx.Deserialize(bytes.NewReader(rawTx))
		if err != nil {
			t.Errorf("TestCalcSignatureHash failed test #%d: "+
			"Failed to parse transaction: %v", i, err)
			continue
		}

		subScript, _ := hex.DecodeString(test[1].(string))
		parsedScript, err := parseScript(subScript)
		if err != nil {
			t.Errorf("TestCalcSignatureHash failed test #%d: "+
			"Failed to parse sub-script: %v", i, err)
			continue
		}

		hashType := SigHashType(testVecF64ToUint32(test[3].(float64)))
		hash := calcSignatureHash(parsedScript, hashType, &tx,
		int(test[2].(float64)))

		expectedHash, _ := utils.HashFromString(test[4].(string))
		if !bytes.Equal(hash, expectedHash[:]) {
			t.Errorf("TestCalcSignatureHash failed test #%d: "+
			"Signature hash mismatch.", i)
		}
	}
}
*/

func TestNewScriptWithRaw(t *testing.T) {
	parseScriptTmp()
}

func parseScriptTmp() (stk []ParsedOpCode, err error) {
	stk = make([]ParsedOpCode, 0)
	scriptLen := 3
	script := NewScriptRaw([]byte{116, 0, 135})

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.bytes[i]
		parsedopCode := ParsedOpCode{opValue: opcode}

		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
			parsedopCode.data = script.bytes[i+1 : i+1+nSize]
		} else if opcode == OP_PUSHDATA1 {
			if scriptLen-i < 1 {
				err = errors.New("OP_PUSHDATA1 has no enough data")
				return
			}
			nSize = int(script.bytes[i+1])
			parsedopCode.data = script.bytes[i+2 : i+2+nSize]
			i++

		} else if opcode == OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				err = errors.New("OP_PUSHDATA2 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint16(script.bytes[i+1 : i+3]))
			parsedopCode.data = script.bytes[i+3 : i+3+nSize]
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				err = errors.New("OP_PUSHDATA4 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint32(script.bytes[i+1 : i+5]))
			parsedopCode.data = script.bytes[i+5 : i+5+nSize]
			i += 4
		}
		if scriptLen-i < 0 || (scriptLen-i) < nSize {
			err = errors.New("size is wrong")
			return
		}

		stk = append(stk, parsedopCode)
		i += nSize
	}
	return

}
