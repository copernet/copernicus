package lscript

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	. "github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	bcrypto "github.com/detailyang/go-bcrypto"
)

// ScripBuilder exposes friendly API to build the script code
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

type TestBuilder struct {
	script       *script.Script
	redeemscript *script.Script
	comment      string
	amount       uint64
	flag         uint32

	scriptError errcode.ScriptErr

	havePush bool
	push     []byte

	creditTx *tx.Tx
	spendTx  *tx.Tx
}

func NewCreditingTransaction(s *script.Script, value uint64) *tx.Tx {
	tx := tx.NewTx(1, 0)
	input := txin.NewTxIn(
		outpoint.NewDefaultOutPoint(),
		NewScriptBuilder().PushNumber(0).PushNumber(0).Script(),
		0xffffffff,
	)
	output := txout.NewTxOut(amount.Amount(value), s)
	tx.AddTxIn(input)
	tx.AddTxOut(output)

	return tx
}

func NewSpendingTransaction(s *script.Script, creditTx *tx.Tx) *tx.Tx {
	tx := tx.NewTx(1, 0)
	input := txin.NewTxIn(
		outpoint.NewOutPoint(creditTx.GetHash(), 0),
		s,
		0xffffffff,
	)
	output := txout.NewTxOut(amount.Amount(creditTx.GetTxOut(0).GetValue()), script.NewEmptyScript())
	tx.AddTxIn(input)
	tx.AddTxOut(output)

	return tx
}

func NewTestBuilder(s *script.Script, comment string, flag uint32, P2SH bool, amount uint64) *TestBuilder {
	var redeemscript *script.Script
	scriptPubkey := s

	if P2SH {
		redeemscript = scriptPubkey
		scriptPubkey = NewScriptBuilder().PushOPCode(OP_HASH160).
			PushBytesWithOP(util.Hash160(redeemscript.Bytes())).
			PushOPCode(OP_EQUAL).Script()
	}

	// OP_HASH160 [20-byte-hash-value] OP_EQUAL
	creditTx := NewCreditingTransaction(scriptPubkey, amount)
	spendTx := NewSpendingTransaction(script.NewEmptyScript(), creditTx)

	return &TestBuilder{
		spendTx:      spendTx,
		creditTx:     creditTx,
		amount:       amount,
		flag:         flag,
		script:       s,
		comment:      comment,
		redeemscript: redeemscript,
	}
}

func clone(s []byte) []byte {
	h := make([]byte, len(s))
	copy(h, s)
	return h
}

// 30 + 02 + len(r) + r + 02 + len(s) + s
func negateSigantureS(sig []byte) []byte {
	r := clone(sig[4 : 4+sig[3]])
	s := clone(sig[6+sig[3] : 6+sig[3]+sig[5+sig[3]]])

	for len(s) < 33 {
		s = append([]byte{0x00}, s...)
	}

	order := []byte{
		0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE, 0xBA, 0xAE, 0xDC, 0xE6, 0xAF,
		0x48, 0xA0, 0x3B, 0xBF, 0xD2, 0x5E, 0x8C, 0xD0, 0x36, 0x41, 0x41}

	carry := 0
	for p := 32; p >= 1; p-- {
		n := int(order[p]) - int(s[p]) - carry
		s[p] = byte(int(n+256) & 0xFF)
		if n < 0 {
			carry = 1
		} else {
			carry = 0
		}
	}

	if len(s) > 1 && s[0] == 0 && s[1] < 0x80 {
		s = s[1:]
	}

	newsig := make([]byte, 0, len(sig))
	newsig = append(newsig, 0x30)
	newsig = append(newsig, byte(4+len(r)+len(s)))
	newsig = append(newsig, 0x02)
	newsig = append(newsig, byte(len(r)))
	newsig = append(newsig, r...)
	newsig = append(newsig, 0x02)
	newsig = append(newsig, byte(len(s)))
	newsig = append(newsig, s...)

	return newsig
}

func (tb *TestBuilder) Num(n int) *TestBuilder {
	tb.DoPush()
	scriptSig := tb.spendTx.GetTxIn(0).GetScriptSig()
	scriptSig.PushInt64(int64(n))
	tb.spendTx.GetTxIn(0).SetScriptSig(scriptSig)
	return tb
}

func (tb *TestBuilder) PushPubkey(pubkey *crypto.PublicKey) *TestBuilder {
	return tb.DoPushBytes(pubkey.ToBytes())
}

func (tb *TestBuilder) PushHex(hexstring string) *TestBuilder {
	data, err := hex.DecodeString(hexstring)
	if err != nil {
		panic(err)
	}

	return tb.DoPushBytes(data)
}

func (tb *TestBuilder) Push(publickey *crypto.PublicKey) *TestBuilder {
	tb.DoPushBytes(publickey.ToBytes())
	return tb
}

func (tb *TestBuilder) DoPush() {
	if tb.havePush {
		scriptSig := tb.spendTx.GetTxIn(0).GetScriptSig()
		scriptSig.PushSingleData(tb.push)
		tb.spendTx.GetTxIn(0).SetScriptSig(scriptSig)
		tb.havePush = false
	}
}

func (tb *TestBuilder) EditPush(pos int, hexin, hexout string) *TestBuilder {
	datain, _ := hex.DecodeString(hexin)
	dataout, _ := hex.DecodeString(hexout)

	if !bytes.Equal(tb.push[pos:pos+len(datain)], datain) {
		panic(tb.comment)
	}

	left := clone(tb.push[:pos])
	right := clone(tb.push[pos+len(datain):])

	tb.push = append(left, dataout...)
	tb.push = append(tb.push, right...)

	return tb
}

func (tb *TestBuilder) Add(s *script.Script) *TestBuilder {
	tb.DoPush()
	scriptSig := tb.spendTx.GetTxIn(0).GetScriptSig()
	scriptSig.PushData(s.Bytes())
	tb.spendTx.GetTxIn(0).SetScriptSig(scriptSig)
	return tb
}

func (tb *TestBuilder) DoPushBytes(data []byte) *TestBuilder {
	tb.DoPush()
	tb.push = data
	tb.havePush = true

	return tb
}

func (tb *TestBuilder) PushRedeem() *TestBuilder {
	tb.DoPushBytes(tb.redeemscript.Bytes())
	return tb
}

func (tb *TestBuilder) PushSig(
	key *crypto.PrivateKey,
	sigHash uint32,
	nr, ns int,
	value uint64,
	flags int,
) *TestBuilder {
	txSigHash, err := tx.SignatureHash(tb.spendTx, tb.script, sigHash,
		0, amount.Amount(value), uint32(flags))
	if err != nil {
		panic(err)
	}
	sig := tb.DoSign(key, txSigHash, nr, ns)
	sig = append(sig, byte(sigHash))

	tb.DoPushBytes(sig)

	return tb
}

func (tb *TestBuilder) DamagePush(pos int) *TestBuilder {
	tb.push[pos] ^= 1
	return tb
}

func (tb *TestBuilder) ScriptError(err errcode.ScriptErr) *TestBuilder {
	tb.scriptError = err
	return tb
}

func DoTest(
	t *testing.T,
	scriptPubkey *script.Script,
	scriptSig *script.Script,
	flag uint32,
	message string,
	scriptError errcode.ScriptErr,
	value uint64,
) {
	creditTx := NewCreditingTransaction(
		scriptPubkey, value,
	)
	spentTx := NewSpendingTransaction(scriptSig, creditTx)

	// err := lscript.VerifyScript(newTx, txin.GetScriptSig(), pkscript, k, amount.Amount(prevOut.inputVal),
	// 	flags, lscript.NewScriptRealChecker())

	err := VerifyScript(spentTx, scriptSig, scriptPubkey,
		0,
		amount.Amount(value),
		flag,
		NewScriptRealChecker(),
	)

	if scriptError == 0 {
		if err != nil {
			t.Errorf("%s - got %s", message, err)
		}
	} else {
		if err != errcode.New(scriptError) {
			t.Errorf("%s - got %s", message, err)
		}
	}

}

func (tb *TestBuilder) Test(t *testing.T) {
	tb.DoPush()
	DoTest(
		t,
		tb.creditTx.GetTxOut(0).GetScriptPubKey(),
		tb.spendTx.GetTxIn(0).GetScriptSig(),
		tb.flag,
		tb.comment,
		tb.scriptError,
		tb.amount,
	)
}

func (tb *TestBuilder) DoSign(key *crypto.PrivateKey, hash util.Hash, nr, ns int) []byte {
	var sig, r, s []byte

	bkey := bcrypto.NewKey(clone(key.GetBytes()), key.IsCompressed())

	for i := 0; ; i++ {
		// TODO: use self-hosted crypto, here just for support test_case
		sig, _ = bkey.Signature(hash[:], uint32(i))

		if (ns == 33) != (sig[5]+sig[3] == 33) {
			sig = negateSigantureS(sig)
		}

		r = clone(sig[4 : 4+sig[3]])
		s = clone(sig[6+sig[3] : 6+sig[3]+sig[5+sig[3]]])

		if nr == len(r) && ns == len(s) {
			break
		}
	}

	return sig
}

var (
	key0bytes = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	key1bytes = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0}

	key2bytes = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0}
)

func TestScriptuint32(t *testing.T) {
	key0 := crypto.NewPrivateKeyFromBytes(key0bytes[:], false)
	key0c := crypto.NewPrivateKeyFromBytes(key0bytes[:], true)
	pubkey0 := key0.PubKey()
	pubkey0h := key0.PubKey().ToBytes()
	pubkey0h[0] = 0x06 | (pubkey0h[64] & 1)
	pubkey0c := key0c.PubKey()

	key1 := crypto.NewPrivateKeyFromBytes(key1bytes[:], false)
	key1c := crypto.NewPrivateKeyFromBytes(key1bytes[:], true)
	pubkey1 := key1.PubKey()
	pubkey1c := key1c.PubKey()

	key2 := crypto.NewPrivateKeyFromBytes(key2bytes[:], false)
	key2c := crypto.NewPrivateKeyFromBytes(key2bytes[:], true)
	// pubkey2, _ := key1.GetPubkey()
	pubkey2c := key2c.PubKey()

	flag := script.ScriptEnableSigHashForkID

	tests := make([]*TestBuilder, 0, 512)

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK",
		0,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(NewScriptBuilder().PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(), "P2PK, bad sig", 0,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		DamagePush(10).
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_DUP).PushOPCode(OP_HASH160).
			PushBytesWithOP(pubkey1c.ToHash160()).
			PushOPCode(OP_EQUALVERIFY).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2PKH",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		Push(pubkey1c))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_DUP).PushOPCode(OP_HASH160).
			PushBytesWithOP(pubkey2c.ToHash160()).
			PushOPCode(OP_EQUALVERIFY).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2PKH, bad pubkey",
		0,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).Push(pubkey2c).
		DamagePush(5).ScriptError(errcode.ScriptErrEqualVerify))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1.ToBytes()).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2PK anyonecanpy",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll|crypto.SigHashAnyoneCanpay, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1.ToBytes()).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2PK anyonecanpy marked with normal hashtype",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll|crypto.SigHashAnyoneCanpay, 32, 32, 0, flag).
		EditPush(70, "81", "01").ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK)",
		script.ScriptVerifyP2SH,
		true,
		0).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).PushRedeem())

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK)",
		script.ScriptVerifyP2SH,
		true,
		0).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).PushRedeem().DamagePush(10).
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_DUP).PushOPCode(OP_HASH160).
			PushBytesWithOP(pubkey0.ToHash160()).PushOPCode(OP_EQUALVERIFY).PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK)",
		script.ScriptVerifyP2SH,
		true,
		0).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).Push(pubkey0).PushRedeem())

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_DUP).PushOPCode(OP_HASH160).
			PushBytesWithOP(pubkey1.ToHash160()).PushOPCode(OP_EQUALVERIFY).PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK), bad sig but no VERIFY_P2SH",
		0,
		true,
		0).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).DamagePush(10).PushRedeem())

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_DUP).PushOPCode(OP_HASH160).
			PushBytesWithOP(pubkey1.ToHash160()).PushOPCode(OP_EQUALVERIFY).PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK), bad sig",
		script.ScriptVerifyP2SH,
		true,
		0).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).DamagePush(10).PushRedeem().
		ScriptError(errcode.ScriptErrEqualVerify))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_3).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_3).PushOPCode(OP_CHECKMULTISIG).Script(),
		"3-of-3",
		0,
		false,
		0,
	).Num(0).
		PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_3).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_3).PushOPCode(OP_CHECKMULTISIG).Script(),
		"3-of-3, 2 sigs",
		0,
		false,
		0,
	).Num(0).
		PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		Num(0).
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).
			PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_3).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"P2SH(2-of-3)",
		script.ScriptVerifyP2SH,
		true,
		0,
	).Num(0).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		PushRedeem())

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).
			PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_3).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"P2SH(2-of-3), 1 sig",
		script.ScriptVerifyP2SH,
		true,
		0,
	).Num(0).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		Num(0).
		PushRedeem().
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with too much R padding but no DERSIG",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 31, 32, 0, flag).EditPush(1, "43021F", "44022000"))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with too much R padding",
		script.ScriptVerifyDersig,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 31, 32, 0, flag).EditPush(1, "43021F", "44022000").
		ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with too much s padding but no DERSIG",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		EditPush(1, "44", "45").
		EditPush(37, "20", "2100"))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with too much s padding but no DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		EditPush(1, "44", "45").
		EditPush(37, "20", "2100").ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with too little R padding but no DERSIG",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220"))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with too little R padding but no DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK NOT with bad sig with too much R padding but no DERSIG",
		0,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 31, 32, 0, flag).
		EditPush(1, "43021f", "44022000").
		DamagePush(10))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK NOT with bad sig with too much R padding",
		script.ScriptVerifyDersig,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 31, 32, 0, flag).
		EditPush(1, "43021f", "44022000").
		DamagePush(10).ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK NOT with bad sig with too much R padding but no DERSIG",
		0,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 31, 32, 0, flag).
		EditPush(1, "43021f", "44022000").
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK NOT with bad sig with too much R padding",
		script.ScriptVerifyDersig,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 31, 32, 0, flag).
		EditPush(1, "43021f", "44022000").
		ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"BIP66 example 1, without DERSIG",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).
		EditPush(1, "45022100", "440220"))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"BIP66 example 1, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).
		EditPush(1, "45022100", "440220").ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 2, without DERSIG",
		0,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).
		EditPush(1, "45022100", "440220").ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 2, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).
		EditPush(1, "45022100", "440220").ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"BIP66 example 3, without DERSIG",
		0,
		false,
		0,
	).Num(0).ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"BIP66 example 3, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 4, without DERSIG",
		0,
		false,
		0,
	).Num(0))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 4, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 4, without DERSIG, non-null DER-compliant signature",
		0,
		false,
		0,
	).PushHex("300602010102010101"))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 4, without DERSIG, non-null DER-compliant signature",
		script.ScriptVerifyDersig|script.ScriptVerifyNullFail,
		false,
		0,
	).Num(0))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 4, with DERSIG, non-null DER-compliant signature",
		script.ScriptVerifyDersig|script.ScriptVerifyNullFail,
		false,
		0,
	).PushHex("300602010102010101").ScriptError(errcode.ScriptErrSigNullFail))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"BIP66 example 5, without DERSIG",
		0,
		false,
		0,
	).Num(0).ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"BIP66 example 5, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(1).ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 6, without DERSIG",
		0,
		false,
		0,
	).Num(1))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 6, without DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(1).ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"BIP66 example 7, without DERSIG",
		0,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"BIP66 example 7, without DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 8, without DERSIG",
		0,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 8, without DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"BIP66 example 9, without DERSIG",
		0,
		false,
		0,
	).Num(0).Num(0).PushSig(key2, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"BIP66 example 9, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).Num(0).PushSig(key2, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 10, without DERSIG",
		0,
		false,
		0,
	).Num(0).Num(0).PushSig(key2, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220"))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 10, without DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).Num(0).PushSig(key2, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").
		ScriptError(errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"BIP66 example 11, without DERSIG",
		0,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").Num(0).
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).Script(),
		"BIP66 example 11, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").Num(0).
		ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 12, without DERSIG",
		0,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").Num(0))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_2).
			PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"BIP66 example 12, without DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 33, 32, 0, flag).EditPush(1, "45022100", "440220").Num(0))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with multi-byte hashtype, without DERSIG",
		0,
		false,
		0,
	).Num(0).PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).EditPush(70, "01", "0101"))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with multi-byte hashtype, with DERSIG",
		script.ScriptVerifyDersig,
		false,
		0,
	).Num(0).PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).EditPush(70, "01", "0101").ScriptError(
		errcode.ScriptErrSigDer))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with high S but no LOW_S",
		0,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 32, 33, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with high S",
		script.ScriptVerifyLowS,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 32, 33, 0, flag).ScriptError(errcode.ScriptErrSigHighs))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0h).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with hybrid pubkey but no STRICTENC",
		0,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0h).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with hybrid pubkey",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).ScriptError(errcode.ScriptErrPubKeyType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0h).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK with hybrid pubkey but no STRICTENC",
		0,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0h).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK with hybrid pubkey",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).ScriptError(errcode.ScriptErrPubKeyType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0h).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK with invalid hybrid pubkey but no STRICTENC",
		0,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).DamagePush(10))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey0h).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK with invalid hybrid pubkey",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).DamagePush(10).ScriptError(errcode.ScriptErrPubKeyType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_1).PushBytesWithOP(pubkey0h).PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_2).PushOPCode(OP_CHECKMULTISIG).Script(),
		"1-of-2 with the second 1 hybrid pubkey and no STRICTENC",
		0,
		false,
		0,
	).Num(0).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_1).PushBytesWithOP(pubkey0h).PushBytesWithOP(pubkey1c.ToBytes()).PushOPCode(OP_2).PushOPCode(OP_CHECKMULTISIG).Script(),
		"1-of-2 with the second 1 hybrid pubkey",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_1).PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey0h).PushOPCode(OP_2).PushOPCode(OP_CHECKMULTISIG).Script(),
		"1-of-2 with the second 1 hybrid pubkey",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).ScriptError(errcode.ScriptErrPubKeyType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with undefined hashtype but no STRICTENC",
		0,
		false,
		0,
	).PushSig(key1, (5), 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey1.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with undefined hashtype",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).PushSig(key1, (5), 32, 32, 0, flag).ScriptError(errcode.ScriptErrSigHashType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_DUP).PushOPCode(OP_HASH160).
			PushBytesWithOP(pubkey0.ToHash160()).PushOPCode(OP_EQUALVERIFY).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with undefined hashtype",
		0,
		false,
		0,
	).PushSig(key0, (0x21), 32, 32, 0, 0).PushPubkey(pubkey0))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_DUP).PushOPCode(OP_HASH160).
			PushBytesWithOP(pubkey0.ToHash160()).PushOPCode(OP_EQUALVERIFY).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with undefined hashtype",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).PushSig(key0, (0x21), 32, 32, 0, script.ScriptVerifyStrictEnc).
		PushPubkey(pubkey0).ScriptError(errcode.ScriptErrSigHashType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey1.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK) with invalid sighashtype",
		script.ScriptVerifyP2SH,
		true,
		0,
	).PushSig(key1, (0x21), 32, 32, 0, flag).PushRedeem())

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey1.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK) with invalid sighashtype",
		script.ScriptVerifyP2SH|script.ScriptVerifyStrictEnc,
		true,
		0,
	).PushSig(key1, (0x21), 32, 32, 0, flag).PushRedeem().
		ScriptError(errcode.ScriptErrSigHashType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey1.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK NOT with invalid sig and undefined hashtype but no STRICTENC",
		0,
		false,
		0,
	).PushSig(key1, (5), 32, 32, 0, flag).
		DamagePush(10))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey1.ToBytes()).PushOPCode(OP_CHECKSIG).PushOPCode(OP_NOT).Script(),
		"P2PK NOT with invalid sig and undefined hashtype",
		script.ScriptVerifyStrictEnc,
		false,
		0,
	).PushSig(key1, (5), 32, 32, 0, flag).
		DamagePush(10).ScriptError(errcode.ScriptErrSigHashType))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_3).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_3).PushOPCode(OP_CHECKMULTISIG).Script(),
		"3-of-3 with nonzero dummy but no NULLDUMMY",
		0,
		false,
		0,
	).Num(1).
		PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_3).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_3).PushOPCode(OP_CHECKMULTISIG).Script(),
		"3-of-3 with nonzero dummy",
		script.ScriptVerifyNullDummy,
		false,
		0,
	).Num(1).
		PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		ScriptError(errcode.ScriptErrSigNullDummy))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_3).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_3).PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"3-of-3 NOT with invalid sig and nonzero dummy but no NULLDUMMY",
		0,
		false,
		0,
	).Num(1).
		PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		DamagePush(10))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_3).PushBytesWithOP(pubkey0c.ToBytes()).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_3).PushOPCode(OP_CHECKMULTISIG).PushOPCode(OP_NOT).Script(),
		"3-of-3 NOT with invalid sig and nonzero dummy but no NULLDUMMY",
		script.ScriptVerifyNullDummy,
		false,
		0,
	).Num(1).
		PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		DamagePush(10).
		ScriptError(errcode.ScriptErrSigNullDummy))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey1c.ToBytes()).
			PushOPCode(OP_2).PushOPCode(OP_CHECKMULTISIG).Script(),
		"2-of-2 with two identical keys and sigs pushed using OP_DUP but no SIGPUSHONLY",
		0,
		false,
		0,
	).Num(0).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		Add(NewScriptBuilder().PushOPCode(OP_DUP).Script()))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey1c.ToBytes()).
			PushOPCode(OP_2).PushOPCode(OP_CHECKMULTISIG).Script(),
		"2-of-2 with two identical keys and sigs pushed using OP_DUP",
		script.ScriptVerifySigPushOnly,
		false,
		0,
	).Num(0).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		Add(NewScriptBuilder().PushOPCode(OP_DUP).Script()).ScriptError(errcode.ScriptErrSigPushOnly))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK) with non-push scriptSig but no P2SH or SIGPUSHONLY",
		0,
		true,
		0,
	).PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		Add(NewScriptBuilder().PushOPCode(OP_NOP8).Script()).PushRedeem())

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with non-push scriptSig but with P2SH validation",
		0,
		false,
		0,
	).PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		Add(NewScriptBuilder().PushOPCode(OP_NOP8).Script()))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK) with non-push scriptSig but no SIGPUSHONLY",
		script.ScriptVerifyP2SH,
		true,
		0,
	).PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		Add(NewScriptBuilder().PushOPCode(OP_NOP8).Script()).PushRedeem().ScriptError(errcode.ScriptErrSigPushOnly))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushBytesWithOP(pubkey2c.ToBytes()).
			PushOPCode(OP_CHECKSIG).Script(),
		"P2SH(P2PK) with non-push scriptSig but not P2SH",
		script.ScriptVerifySigPushOnly,
		true,
		0,
	).PushSig(key2, crypto.SigHashAll, 32, 32, 0, flag).
		Add(NewScriptBuilder().PushOPCode(OP_NOP8).Script()).PushRedeem().ScriptError(errcode.ScriptErrSigPushOnly))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().PushOPCode(OP_2).
			PushBytesWithOP(pubkey1c.ToBytes()).PushBytesWithOP(pubkey1c.ToBytes()).
			PushOPCode(OP_2).PushOPCode(OP_CHECKMULTISIG).Script(),
		"2-of-2 with two identical keys and sigs pushed",
		script.ScriptVerifySigPushOnly,
		false,
		0,
	).Num(0).PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag).
		PushSig(key1, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with unnecessary input but no CLEANSTACK",
		script.ScriptVerifyP2SH,
		false,
		0,
	).Num(11).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with unnecessary input but no CLEANSTACK",
		script.ScriptVerifyP2SH|script.ScriptVerifyCleanStack,
		false,
		0,
	).Num(11).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).
		ScriptError(errcode.ScriptErrCleanStack))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with unnecessary input but no CLEANSTACK",
		script.ScriptVerifyP2SH,
		true,
		0,
	).Num(11).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).PushRedeem())

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with unnecessary input",
		script.ScriptVerifyP2SH|script.ScriptVerifyCleanStack,
		true,
		0,
	).Num(11).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).PushRedeem().ScriptError(
		errcode.ScriptErrCleanStack,
	))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK with CLEANSTACK",
		script.ScriptVerifyP2SH|script.ScriptVerifyCleanStack,
		true,
		0,
	).PushSig(key0, crypto.SigHashAll, 32, 32, 0, flag).PushRedeem())

	value := uint64(12345000000000)

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK FORKID",
		script.ScriptEnableSigHashForkID,
		false,
		value,
	).PushSig(key0, crypto.SigHashAll|crypto.SigHashForkID, 32, 32, value, flag))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK INVALID AMOUNT",
		script.ScriptEnableSigHashForkID,
		false,
		value,
	).PushSig(key0, crypto.SigHashAll|crypto.SigHashForkID, 32, 32, value+1, flag).ScriptError(errcode.ScriptErrEvalFalse))

	tests = append(tests, NewTestBuilder(
		NewScriptBuilder().
			PushBytesWithOP(pubkey0.ToBytes()).PushOPCode(OP_CHECKSIG).Script(),
		"P2PK INVALID FORKID",
		script.ScriptVerifyStrictEnc,
		false,
		value,
	).PushSig(key0, crypto.SigHashAll|crypto.SigHashForkID, 32, 32, value, flag).
		ScriptError(errcode.ScriptErrIllegalForkID))

	for _, test := range tests {
		test.Test(t)
	}

}
