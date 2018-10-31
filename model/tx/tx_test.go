package tx

import (
	"bytes"
	"encoding/hex"
	"github.com/btcsuite/btcutil"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/util/amount"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
)

var tests = []struct {
	GetHash string
	txRaw   string
	tx      *Tx
}{
	{
		GetHash: "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87",
		txRaw:   "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff08044c86041b020602ffffffff0100f2052a010000004341041b0e8c2567c12536aa13357b79a073dc4444acb83c4ec7a0e2f99dd7457516c5817242da796924ca4e99947d087fedf9ce467cb9f7c6287078f801df276fdf84ac00000000",
		tx: &Tx{
			version: 1,
			ins: []*txin.TxIn{
				txin.NewTxIn(outpoint.NewOutPoint(util.Hash{}, 0xffffffff),
					script.NewScriptRaw([]byte{0x04, 0x4c, 0x86, 0x04, 0x1b, 0x02, 0x06, 0x02}),
					0xffffffff),
			},
			outs: []*txout.TxOut{
				txout.NewTxOut(0x12a05f200, script.NewScriptRaw([]byte{
					0x41, // OP_DATA_65
					0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
					0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
					0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
					0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
					0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
					0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
					0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
					0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
					0x84, // 65-byte signature
					0xac, // OP_CHECKSIG
				})),
			},
		},
	},
	{
		GetHash: "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
		txRaw:   "0100000001032e38e9c0a84c6046d687d10556dcacc41d275ec55fc00779ac88fdf357a187000000008c493046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748014104f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3ffffffff0200e32321000000001976a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac000fe208010000001976a914948c765a6914d43f2a7ac177da2c2f6b52de3d7c88ac00000000",
		tx: &Tx{
			version: 1,
			ins: []*txin.TxIn{
				txin.NewTxIn(outpoint.NewOutPoint(util.Hash{
					0x03, 0x2e, 0x38, 0xe9, 0xc0, 0xa8, 0x4c, 0x60,
					0x46, 0xd6, 0x87, 0xd1, 0x05, 0x56, 0xdc, 0xac,
					0xc4, 0x1d, 0x27, 0x5e, 0xc5, 0x5f, 0xc0, 0x07,
					0x79, 0xac, 0x88, 0xfd, 0xf3, 0x57, 0xa1, 0x87,
				}, 0), script.NewScriptRaw([]byte{
					0x49, // OP_DATA_73
					0x30, 0x46, 0x02, 0x21, 0x00, 0xc3, 0x52, 0xd3,
					0xdd, 0x99, 0x3a, 0x98, 0x1b, 0xeb, 0xa4, 0xa6,
					0x3a, 0xd1, 0x5c, 0x20, 0x92, 0x75, 0xca, 0x94,
					0x70, 0xab, 0xfc, 0xd5, 0x7d, 0xa9, 0x3b, 0x58,
					0xe4, 0xeb, 0x5d, 0xce, 0x82, 0x02, 0x21, 0x00,
					0x84, 0x07, 0x92, 0xbc, 0x1f, 0x45, 0x60, 0x62,
					0x81, 0x9f, 0x15, 0xd3, 0x3e, 0xe7, 0x05, 0x5c,
					0xf7, 0xb5, 0xee, 0x1a, 0xf1, 0xeb, 0xcc, 0x60,
					0x28, 0xd9, 0xcd, 0xb1, 0xc3, 0xaf, 0x77, 0x48,
					0x01, // 73-byte signature
					0x41, // OP_DATA_65
					0x04, 0xf4, 0x6d, 0xb5, 0xe9, 0xd6, 0x1a, 0x9d,
					0xc2, 0x7b, 0x8d, 0x64, 0xad, 0x23, 0xe7, 0x38,
					0x3a, 0x4e, 0x6c, 0xa1, 0x64, 0x59, 0x3c, 0x25,
					0x27, 0xc0, 0x38, 0xc0, 0x85, 0x7e, 0xb6, 0x7e,
					0xe8, 0xe8, 0x25, 0xdc, 0xa6, 0x50, 0x46, 0xb8,
					0x2c, 0x93, 0x31, 0x58, 0x6c, 0x82, 0xe0, 0xfd,
					0x1f, 0x63, 0x3f, 0x25, 0xf8, 0x7c, 0x16, 0x1b,
					0xc6, 0xf8, 0xa6, 0x30, 0x12, 0x1d, 0xf2, 0xb3,
					0xd3, // 65-byte pubkey
				}), 0xffffffff),
			},
			outs: []*txout.TxOut{
				txout.NewTxOut(0x2123e300, script.NewScriptRaw([]byte{
					0x76, // OP_DUP
					0xa9, // OP_HASH160
					0x14, // OP_DATA_20
					0xc3, 0x98, 0xef, 0xa9, 0xc3, 0x92, 0xba, 0x60,
					0x13, 0xc5, 0xe0, 0x4e, 0xe7, 0x29, 0x75, 0x5e,
					0xf7, 0xf5, 0x8b, 0x32,
					0x88, // OP_EQUALVERIFY
					0xac, // OP_CHECKSIG
				})),
				txout.NewTxOut(0x108e20f00, script.NewScriptRaw([]byte{
					0x76, // OP_DUP
					0xa9, // OP_HASH160
					0x14, // OP_DATA_20
					0x94, 0x8c, 0x76, 0x5a, 0x69, 0x14, 0xd4, 0x3f,
					0x2a, 0x7a, 0xc1, 0x77, 0xda, 0x2c, 0x2f, 0x6b,
					0x52, 0xde, 0x3d, 0x7c,
					0x88, // OP_EQUALVERIFY
					0xac, // OP_CHECKSIG
				})),
			},
		},
	},
}

func TestTxDeSerializeAndSerialize(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	for i, e := range tests {
		buf.Reset()
		b, err := hex.DecodeString(e.txRaw)
		if err != nil {
			t.Errorf("decode txRaw hex string :%v\n", err)
		}
		if _, err := buf.Write(b); err != nil {
			t.Errorf("write to buf :%v\n", err)
		}

		tx1 := Tx{}
		err = tx1.Unserialize(buf)
		if err != nil {
			t.Errorf("Unserialize error :%v\n", err)
		}

		buf.Reset()
		if err := e.tx.Serialize(buf); err != nil {
			t.Errorf("failed Serialize tx %d tx: %v\n", i, err)
		}
		h := util.DoubleSha256Hash(buf.Bytes())
		if e.GetHash != h.String() {
			t.Errorf("failed compute GetHash for tx %d, expect=(%s), but got %s\n", i, e.GetHash, h)
		}
	}
}

func TestTxSerializeAndTxUnserialize(t *testing.T) {
	rawTxString := "0100000002f6b52b137227b1992a6289e2c1b265953b559d782faba905c209ddf1c7a48fb8" +
		"000000006b48304502200a0c787cb9c132584e640b7686a8f6a78d9c4a41201a0c7a139d5383970b39c50" +
		"22100d6fdc88b87328cdd772ed4dd9f15fea84c85968fe84308bb4a207ba03889cd680121020b76df009c" +
		"b91ce792ae00461e15a9340652c30d1b816129fc61246b3441a9e2ffffffff9285678bbc575493ea20372" +
		"3507fa22e37d775e766eccadde8894fc561602f2c000000006a47304402201ba951afbdeda2cb70483aac" +
		"b144b1ddd9db6fdbe4b6ccaec27c005b4bc4048f0220549ac7d19ddb6c37852bfd23ca2ed4aef429f2432" +
		"e27b309bed1e9217ce68d03012102ee67fdeb2f4484a2342db30f851808942016ff8df57f7e7798a14710" +
		"fa761590ffffffff02b58d1300000000001976a914d8a6c89e4207a50d7f57c6a02fc09f38113ccb6b88a" +
		"c6c400a000000000017a914abaad350c1c83d2f33cc35e8c1c886cd1287bda98700000000"

	originBytes, err := hex.DecodeString(rawTxString)
	assert.NoError(t, err)

	tx := Tx{}
	err = tx.Unserialize(bytes.NewReader(originBytes))
	assert.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	err = tx.Serialize(buf)
	assert.NoError(t, err)

	assert.Equal(t, originBytes, buf.Bytes())
}

func Test_should_able_to_decode_bch_testnet_tx(t *testing.T) {
	//testnet tx hash: 661d4380878bf997c751252b4cee5784c732646c116eeccc8057d0e565122c69
	txstr := "0200000001bebf7bab9021fd3422231e13b39744ee788584bc42b63d15d163d26860d2ea5b010000006b4830450221009439545e50e255cc03d9685c9182415fa1c31bdae1c74fb41cf55ef371e9c5ac022070367d1685deec304bc92863d291788f4355ab3e1ab755afb1189cee4403a891412102413ce16bc4975dcc6945febd4b4786e0d0006c1ac4ff49cdbf71cc2cb5734b70feffffff030000000000000000456a4362636e73000001f4676f626162793200626368746573743a71716b35396a736870386c7934793667346c35686e6565766871663334347a647271756879306e657178008d340f00000000001976a9142d42ca1709fe4a9348afe979e72cb8131ad44d1888ac22020000000000001976a9142d42ca1709fe4a9348afe979e72cb8131ad44d1888acb83d1300"

	txn := NewEmptyTx()
	rawTx, _ := hex.DecodeString(txstr)
	assert.NoError(t, txn.Decode(bytes.NewReader(rawTx)))
	assert.NoError(t, txn.CheckRegularTransaction())
}

func Test_should_able_to_decode_bch_mainnet_tx(t *testing.T) {
	txn := mainNetTx(t)
	assert.Equal(t, int32(1), txn.GetVersion())
	assert.Equal(t, 1, len(txn.GetIns()))
	assert.Equal(t, 2, len(txn.GetOuts()))
	assert.Equal(t, uint32(0), txn.GetLockTime())

	assert.NoError(t, txn.CheckRegularTransaction())
}

func Test_should_able_to_check_duplicate_txins(t *testing.T) {
	txn := mainNetTx(t)
	txn.ins = append(txn.ins, txn.ins[0])

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-inputs-duplicate", t)
}

func Test_should_able_to_reject_empty_vin_txn(t *testing.T) {
	txn := NewTx(0, 1)

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-vin-empty", t)
}

func Test_should_able_to_reject_empty_out_txn(t *testing.T) {
	txn := mainNetTx(t)
	txn.outs = []*txout.TxOut{}

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-vout-empty", t)
}

func Test_should_able_to_reject_too_large_txn(t *testing.T) {
	txn := mainNetTx(t)
	txn.outs[0].SetScriptPubKey(script.NewScriptRaw(make([]byte, consensus.MaxTxSize)))

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-oversize", t)
}

func Test_should_able_to_reject_txn_with__too_large_output_value(t *testing.T) {
	txn := mainNetTx(t)
	assert.Equal(t, 2, len(txn.outs))
	txn.GetTxOut(0).SetValue(amount.Amount(util.MaxMoney + 1))

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-vout-toolarge", t)
}

func Test_should_able_to_reject_txn_with__txout_total_too_large_output_value(t *testing.T) {
	txn := mainNetTx(t)
	assert.Equal(t, 2, len(txn.outs))
	txn.outs[0].SetValue(amount.Amount(util.MaxMoney / 2))
	txn.outs[1].SetValue(amount.Amount(util.MaxMoney/2 + 1))

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-txouttotal-toolarge", t)
}

func Test_should_able_to_reject_txn_with__negative_output_value(t *testing.T) {
	txn := mainNetTx(t)
	assert.Equal(t, 2, len(txn.outs))
	txn.outs[0].SetValue(amount.Amount(-1))

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-vout-negative", t)
}

func Test_should_able_to_reject_txn_with__total_negative_output_value(t *testing.T) {
	txn := mainNetTx(t)
	assert.Equal(t, 2, len(txn.outs))
	txn.outs[0].SetValue(amount.Amount(-2))
	txn.outs[1].SetValue(amount.Amount(1))

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txns-vout-negative", t)
}

func Test_should_able_to_reject_txn_with__too_many_sigops(t *testing.T) {
	txn := mainNetTx(t)
	txn.outs[0].SetScriptPubKey(makeDummyScript(MaxTxSigOpsCounts))
	txn.outs[1].SetScriptPubKey(makeDummyScript(1))

	err := txn.CheckRegularTransaction()

	assertError(err, errcode.RejectInvalid, "bad-txn-sigops", t)
}

func makeDummyScript(size int) *script.Script {
	op := opcodes.NewParsedOpCode(opcodes.OP_CHECKSIG, 1, nil)
	ops := make([]opcodes.ParsedOpCode, size)
	for i := 0; i < size; i++ {
		ops[i] = *op
	}
	return script.NewScriptOps(ops)
}

func Test_should_able_to_reject_coinbase_tx__during_regular_tx_check(t *testing.T) {
	o := outpoint.NewOutPoint(util.HashZero, 0xffffffff)
	coinbaseTx := newCoinbaseTx(o)
	err := coinbaseTx.CheckRegularTransaction()
	assertError(err, errcode.RejectInvalid, "bad-tx-coinbase", t)

	coinbaseTx = newCoinbaseTx(nil)
	err = coinbaseTx.CheckRegularTransaction()
	assertError(err, errcode.RejectInvalid, "bad-tx-coinbase", t)
}

func Test_should_able_to_reject_tx_with_null_prevout__during_regular_tx_check(t *testing.T) {
	txn := mainNetTx(t)
	assert.Equal(t, 1, len(txn.ins))

	outpoint := outpoint.NewOutPoint(util.HashZero, 0xffffffff)
	txin1 := txin.NewTxIn(outpoint, script.NewEmptyScript(), 0)
	txn.ins = append(txn.ins, txin1)

	err := txn.CheckRegularTransaction()
	assertError(err, errcode.RejectInvalid, "bad-txns-prevout-null", t)
}

func Test_genesis_coinbase_tx_should_be_valid_coinbase_tx(t *testing.T) {
	err := NewGenesisCoinbaseTx().CheckCoinbaseTransaction()
	assert.NoError(t, err)
}

func Test_should_able_to_check_coinbase_txn(t *testing.T) {
	o := outpoint.NewOutPoint(util.HashZero, 0xffffffff)
	coinbaseTx := newCoinbaseTx(o)

	err := coinbaseTx.CheckCoinbaseTransaction()

	assert.NoError(t, err)
}

func Test_should_able_to_reject_invalid_coinbase_txn___with_non_null_pre_hash(t *testing.T) {
	coinbaseTx := newCoinbaseTx(outpoint.NewOutPoint(util.HashOne, 0xffffffff))

	err := coinbaseTx.CheckCoinbaseTransaction()
	assertError(err, errcode.RejectInvalid, "bad-cb-missing", t)
}

func Test_should_able_to_reject_invalid_coinbase_txn___with_empty_ins(t *testing.T) {
	coinbaseTx := newCoinbaseTx(outpoint.NewOutPoint(util.HashZero, 0xffffffff))
	coinbaseTx.outs = []*txout.TxOut{}

	err := coinbaseTx.CheckCoinbaseTransaction()
	assertError(err, errcode.RejectInvalid, "bad-txns-vout-empty", t)
}

func Test_should_able_to_reject_invalid_coinbase_txn___with_too_short_scriptsig(t *testing.T) {
	coinbaseTx := newCoinbaseTx(outpoint.NewOutPoint(util.HashZero, 0xffffffff))
	coinbaseTx.ins[0].SetScriptSig(makeDummyScript(1))

	err := coinbaseTx.CheckCoinbaseTransaction()
	assertError(err, errcode.RejectInvalid, "bad-cb-length", t)
}

func Test_should_able_to_reject_invalid_coinbase_txn___with_too_long_scriptsig(t *testing.T) {
	coinbaseTx := newCoinbaseTx(outpoint.NewOutPoint(util.HashZero, 0xffffffff))
	coinbaseTx.ins[0].SetScriptSig(makeDummyScript(101))

	err := coinbaseTx.CheckCoinbaseTransaction()
	assertError(err, errcode.RejectInvalid, "bad-cb-length", t)
}

func Test_assure_the_mainnet_tx_example__is_standard(t *testing.T) {
	givenNoDustRelayFeeLimits()

	tx := mainNetTx(t)
	is, _ := tx.IsStandard()
	assert.True(t, is)
}

func Test_tx_with_wrong_version___should_be_non_standard(t *testing.T) {
	tx := mainNetTx(t)
	tx.version = 0

	isStandard, reason := tx.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "version", reason)
}

func Test_tx_with_unknown_new_version___should_be_non_standard(t *testing.T) {
	tx := mainNetTx(t)
	tx.version = MaxStandardVersion + 1

	isStandard, reason := tx.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "version", reason)
}

func Test_tx_of_too_large_size__should_be_non_standard(t *testing.T) {
	tx := mainNetTx(t)
	tx.ins[0].SetScriptSig(makeDummyScript(int(MaxStandardTxSize)))

	isStandard, reason := tx.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "tx-size", reason)
}

func Test_tx_with__non_pushonly_scriptsig___should_be_non_standard(t *testing.T) {
	tx := mainNetTx(t)
	tx.ins[0].SetScriptSig(makeDummyScript(80))

	isStandard, reason := tx.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "scriptsig-not-pushonly", reason)
}

func Test_tx_with_too_large_scriptsig___should_be_non_standard(t *testing.T) {
	tx := mainNetTx(t)
	tx.ins[0].SetScriptSig(makeDummyScript(script.MaxStandardScriptSigSize + 1))

	isStandard, reason := tx.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "scriptsig-size", reason)
}

func Test_tx_with_too_large_scriptsig_in_any_ins___should_be_non_standard(t *testing.T) {
	txn := mainNetTx(t)
	txn.ins = append(txn.ins, txn.ins[0])
	txn.ins[1].SetScriptSig(makeDummyScript(script.MaxStandardScriptSigSize + 1))

	isStandard, reason := txn.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "scriptsig-size", reason)
}

func Test_tx_with_non_standard_scriptpubkey___should_be_non_standard(t *testing.T) {
	givenNoDustRelayFeeLimits()
	txn := mainNetTx(t)
	txn.outs[1].SetScriptPubKey(makeDummyScript(100))

	isStandard, reason := txn.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "scriptpubkey", reason)
}

func NewPrivateKey() crypto.PrivateKey {
	var keyBytes []byte
	for i := 0; i < 32; i++ {
		keyBytes = append(keyBytes, byte(rand.Uint32()%256))
	}
	return *crypto.PrivateKeyFromBytes(keyBytes)
}

func Test_when_configured_not_BareMultiSigStd___tx_with_multisig_pubkey_should_be_non_standard(t *testing.T) {
	givenNoDustRelayFeeLimits()
	txn := mainNetTx(t)
	txn.outs[0].SetScriptPubKey(createMultiSigScript())
	txn.outs[1].SetScriptPubKey(createMultiSigScript())

	isStandard, reason := txn.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "bare-multisig", reason)
}

func Test_when_configured_BareMultiSigStd___tx_with_multisig_pubkey_should_be_standard(t *testing.T) {
	givenNoDustRelayFeeLimits()
	givenBareMultiSigAsStandardScriptPubKey()

	txn := mainNetTx(t)
	txn.outs[0].SetScriptPubKey(createMultiSigScript())
	txn.outs[1].SetScriptPubKey(createMultiSigScript())

	isStandard, _ := txn.IsStandard()

	assert.True(t, isStandard)
}

func createMultiSigScript() *script.Script {
	crypto.InitSecp256()
	key1 := NewPrivateKey()
	key2 := NewPrivateKey()
	multisig := script.NewEmptyScript()
	multisig.PushOpCode(opcodes.OP_1)
	multisig.PushSingleData(key1.PubKey().ToBytes())
	multisig.PushSingleData(key2.PubKey().ToBytes())
	multisig.PushOpCode(opcodes.OP_2)
	multisig.PushOpCode(opcodes.OP_CHECKMULTISIG)
	return multisig
}

func Test_tx_with_any_dust_outs___should_be_non_standard(t *testing.T) {
	givenDustRelayFeeLimits(100)
	givenAcceptDataCarrier()

	txn := mainNetTx(t)
	minNonDustValue := amount.Amount(txn.outs[1].GetDustThreshold(util.NewFeeRate(conf.Cfg.TxOut.DustRelayFee)))
	txn.outs[1].SetValue(minNonDustValue - 1)

	isStandard, reason := txn.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "dust", reason)
}

func Test_tx_with_multiple_null_data_opreturn_outs___should_be_non_standard(t *testing.T) {
	givenNoDustRelayFeeLimits()
	givenAcceptDataCarrier()

	txn := mainNetTx(t)
	txn.outs[0].SetScriptPubKey(nullDataOpReturnScriptPubKey())
	txn.outs[1].SetScriptPubKey(nullDataOpReturnScriptPubKey())

	isStandard, reason := txn.IsStandard()

	assert.False(t, isStandard)
	assert.Equal(t, "multi-op-return", reason)
}

func Test_tx_with_lock_time_equal_to_zero___should_be_final_tx(t *testing.T) {
	txn := &Tx{lockTime: 0}

	assert.True(t, txn.IsFinal(0 /*unused*/, 0 /*unused*/))
}

func Test_tx_with_zero_locktime__and_any_ins_sequence___is_final_tx(t *testing.T) {
	f := func(anySeq uint32, anyHeight int32, anyTime int64) bool {
		txn := newTxWith(0, anySeq)
		return txn.IsFinal(anyHeight, anyTime)
	}

	assert.NoError(t, quick.Check(f, nil))
}

func Test_tx_is_final___when_locktime_as_height_lock_less_than_tip_height(t *testing.T) {
	f := func(anySeq uint32, anyTime int64) bool {
		heightLock100 := uint32(100)

		txn := newTxWith(heightLock100, anySeq)
		return txn.IsFinal(101, anyTime)
	}
	assert.NoError(t, quick.Check(f, nil))
}

func Test_check_final_tx___when_locktime_as_time_lock__less_than_target_time(t *testing.T) {
	f := func(anySeq uint32, anyHeight int32) bool {
		timeLock := uint32(script.LockTimeThreshold + 1)

		txn := newTxWith(timeLock, anySeq)
		return txn.IsFinal(anyHeight, int64(script.LockTimeThreshold+2))
	}
	assert.NoError(t, quick.Check(f, nil))
}

func Test_tx_is_final___when_script_sequence_is_final__even_height_lock_is_still_locked(t *testing.T) {
	f := func(anyTime int64) bool {
		lockHeight100 := uint32(100)

		txn := newTxWith(lockHeight100, script.SequenceFinal)
		return txn.IsFinal(99, anyTime)
	}
	assert.NoError(t, quick.Check(f, nil))

	f2 := func(anyTime int64) bool {
		heightLock100 := uint32(100)
		nonFinalSequence := uint32(0)

		txn := newTxWith(heightLock100, nonFinalSequence)
		return !txn.IsFinal(99, anyTime)
	}
	assert.NoError(t, quick.Check(f2, nil))
}

func newTxWith(lockTime uint32, sequence uint32) *Tx {
	txn := &Tx{
		lockTime: lockTime,
		ins: []*txin.TxIn{
			{
				Sequence: sequence,
			},
		},
	}
	return txn
}

func nullDataOpReturnScriptPubKey() *script.Script {
	nullDataOpReturn := opcodes.NewParsedOpCode(opcodes.OP_RETURN, 1, nil)
	nullDataOpReturnScript := script.NewScriptOps([]opcodes.ParsedOpCode{*nullDataOpReturn})
	return nullDataOpReturnScript
}

func givenNoDustRelayFeeLimits() {
	if conf.Cfg == nil {
		conf.Cfg = &conf.Configuration{}
	}
	conf.Cfg.TxOut.DustRelayFee = 0
}

func givenDustRelayFeeLimits(minRelayFee int64) {
	if conf.Cfg == nil {
		conf.Cfg = &conf.Configuration{}
	}
	conf.Cfg.TxOut.DustRelayFee = minRelayFee
}

func givenBareMultiSigAsStandardScriptPubKey() {
	if conf.Cfg == nil {
		conf.Cfg = &conf.Configuration{}
	}
	conf.Cfg.Script.IsBareMultiSigStd = true
}

func givenAcceptDataCarrier() {
	if conf.Cfg == nil {
		conf.Cfg = &conf.Configuration{}
	}

	conf.Cfg.Script.AcceptDataCarrier = true
	conf.Cfg.Script.MaxDatacarrierBytes = 223
}

func newCoinbaseTx(outpoint *outpoint.OutPoint) *Tx {
	coinbaseTx := &Tx{
		version: 1,
		ins: []*txin.TxIn{
			txin.NewTxIn(outpoint,
				script.NewScriptRaw([]byte{0x04, 0x4c, 0x86, 0x04, 0x1b, 0x02, 0x06, 0x02}),
				0xffffffff),
		},
		outs: []*txout.TxOut{
			txout.NewTxOut(0x12a05f200, script.NewScriptRaw([]byte{
				0x41, // OP_DATA_65
				0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
				0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
				0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
				0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
				0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
				0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
				0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
				0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
				0x84, // 65-byte signature
				0xac, // OP_CHECKSIG
			})),
		},
	}
	return coinbaseTx
}

func assertError(err error, code errcode.RejectCode, reason string, t *testing.T) {
	c, r, isReject := errcode.IsRejectCode(err)
	assert.True(t, isReject)
	assert.Equal(t, code, c)
	assert.Equal(t, reason, r)
}

func Test_should_able_to_correctly_calculate_hash(t *testing.T) {
	txn := mainNetTx(t)
	txnHash := txn.GetHash()
	assert.Equal(t, "e2769b09e784f32f62ef849763d4f45b98e07ba658647343b915ff832b110436", txnHash.String())
	assert.Equal(t, "e2769b09e784f32f62ef849763d4f45b98e07ba658647343b915ff832b110436", txn.GetHash().String())
	assert.Equal(t, "e2769b09e784f32f62ef849763d4f45b98e07ba658647343b915ff832b110436", txn.calHash().String())
}

func mainNetTx(t *testing.T) *Tx {
	// Random real transaction
	// (e2769b09e784f32f62ef849763d4f45b98e07ba658647343b915ff832b110436)
	txBytes := []byte{
		0x01, 0x00, 0x00, 0x00, 0x01, 0x6b, 0xff, 0x7f, 0xcd, 0x4f, 0x85, 0x65,
		0xef, 0x40, 0x6d, 0xd5, 0xd6, 0x3d, 0x4f, 0xf9, 0x4f, 0x31, 0x8f, 0xe8,
		0x20, 0x27, 0xfd, 0x4d, 0xc4, 0x51, 0xb0, 0x44, 0x74, 0x01, 0x9f, 0x74,
		0xb4, 0x00, 0x00, 0x00, 0x00, 0x8c, 0x49, 0x30, 0x46, 0x02, 0x21, 0x00,
		0xda, 0x0d, 0xc6, 0xae, 0xce, 0xfe, 0x1e, 0x06, 0xef, 0xdf, 0x05, 0x77,
		0x37, 0x57, 0xde, 0xb1, 0x68, 0x82, 0x09, 0x30, 0xe3, 0xb0, 0xd0, 0x3f,
		0x46, 0xf5, 0xfc, 0xf1, 0x50, 0xbf, 0x99, 0x0c, 0x02, 0x21, 0x00, 0xd2,
		0x5b, 0x5c, 0x87, 0x04, 0x00, 0x76, 0xe4, 0xf2, 0x53, 0xf8, 0x26, 0x2e,
		0x76, 0x3e, 0x2d, 0xd5, 0x1e, 0x7f, 0xf0, 0xbe, 0x15, 0x77, 0x27, 0xc4,
		0xbc, 0x42, 0x80, 0x7f, 0x17, 0xbd, 0x39, 0x01, 0x41, 0x04, 0xe6, 0xc2,
		0x6e, 0xf6, 0x7d, 0xc6, 0x10, 0xd2, 0xcd, 0x19, 0x24, 0x84, 0x78, 0x9a,
		0x6c, 0xf9, 0xae, 0xa9, 0x93, 0x0b, 0x94, 0x4b, 0x7e, 0x2d, 0xb5, 0x34,
		0x2b, 0x9d, 0x9e, 0x5b, 0x9f, 0xf7, 0x9a, 0xff, 0x9a, 0x2e, 0xe1, 0x97,
		0x8d, 0xd7, 0xfd, 0x01, 0xdf, 0xc5, 0x22, 0xee, 0x02, 0x28, 0x3d, 0x3b,
		0x06, 0xa9, 0xd0, 0x3a, 0xcf, 0x80, 0x96, 0x96, 0x8d, 0x7d, 0xbb, 0x0f,
		0x91, 0x78, 0xff, 0xff, 0xff, 0xff, 0x02, 0x8b, 0xa7, 0x94, 0x0e, 0x00,
		0x00, 0x00, 0x00, 0x19, 0x76, 0xa9, 0x14, 0xba, 0xde, 0xec, 0xfd, 0xef,
		0x05, 0x07, 0x24, 0x7f, 0xc8, 0xf7, 0x42, 0x41, 0xd7, 0x3b, 0xc0, 0x39,
		0x97, 0x2d, 0x7b, 0x88, 0xac, 0x40, 0x94, 0xa8, 0x02, 0x00, 0x00, 0x00,
		0x00, 0x19, 0x76, 0xa9, 0x14, 0xc1, 0x09, 0x32, 0x48, 0x3f, 0xec, 0x93,
		0xed, 0x51, 0xf5, 0xfe, 0x95, 0xe7, 0x25, 0x59, 0xf2, 0xcc, 0x70, 0x43,
		0xf9, 0x88, 0xac, 0x00, 0x00, 0x00, 0x00}
	txn := NewEmptyTx()
	assert.NoError(t, txn.Decode(bytes.NewReader(txBytes)))

	txn2 := &Tx{
		version: 1,
		ins: []*txin.TxIn{
			txin.NewTxIn(outpoint.NewOutPoint(util.Hash{
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
				0xffffffff),
		},
		outs: []*txout.TxOut{
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
		},
		lockTime: 0,
	}

	buf := bytes.NewBuffer(nil)
	err := txn2.Serialize(buf)
	assert.NoError(t, err)

	assert.Equal(t, txBytes, buf.Bytes())

	return txn
}

func Test_basic_tx_methods(t *testing.T) {
	txn := mainNetTx(t)

	assert.Nil(t, txn.GetTxIn(-1))
	assert.Nil(t, txn.GetTxIn(txn.GetInsCount()))
	assert.Nil(t, txn.GetTxOut(-1))
	assert.Nil(t, txn.GetTxOut(txn.GetOutsCount()))

	prevouts := txn.GetAllPreviousOut()
	hashes := txn.PrevoutHashs()
	assert.Equal(t, len(txn.GetIns()), len(prevouts))
	assert.Equal(t, len(txn.GetIns()), len(hashes))

	txHashes := make(map[util.Hash]struct{})
	assert.False(t, txn.AnyInputTxIn(txHashes))

	txHashes[txn.ins[0].PreviousOutPoint.Hash] = struct{}{}
	assert.True(t, txn.AnyInputTxIn(txHashes))

	outValue := txn.GetValueOut()
	expectOutValue := amount.Amount(0)
	for _, out := range txn.GetOuts() {
		expectOutValue += out.GetValue()
	}
	assert.Equal(t, expectOutValue, outValue)

	assert.Equal(t, txn.EncodeSize(), txn.SerializeSize())
}

// The struct Var contains some variable which testing using.
// keyMap is used to save the relation publicKeyHash and privateKey, k is publicKeyHash, v is privateKey.
type Var struct {
	priKeys    []crypto.PrivateKey
	pubKeys    []crypto.PublicKey
	prevHolder Tx
	spender    Tx
	keyStore   *crypto.KeyStore
}

// Initial the test variable
func initVar() *Var {
	var v Var
	v.keyStore = crypto.NewKeyStore()

	for i := 0; i < 3; i++ {
		privateKey := NewPrivateKey()
		v.priKeys = append(v.priKeys, privateKey)

		pubKey := *privateKey.PubKey()
		v.pubKeys = append(v.pubKeys, pubKey)

		v.keyStore.AddKey(&privateKey)
	}

	return &v
}

func TestSignStepP2PKH(t *testing.T) {
	v := initVar()

	// Create a P2PKHLockingScript script
	scriptPubKey := script.NewEmptyScript()
	scriptPubKey.PushOpCode(opcodes.OP_DUP)
	scriptPubKey.PushOpCode(opcodes.OP_HASH160)
	scriptPubKey.PushSingleData(btcutil.Hash160(v.pubKeys[0].ToBytes()))
	scriptPubKey.PushOpCode(opcodes.OP_EQUALVERIFY)
	scriptPubKey.PushOpCode(opcodes.OP_CHECKSIG)

	// Add locking script to prevHolder
	v.prevHolder.AddTxOut(txout.NewTxOut(0, scriptPubKey))

	v.spender.AddTxIn(
		txin.NewTxIn(
			outpoint.NewOutPoint(v.prevHolder.GetHash(), 0),
			script.NewEmptyScript(),
			script.SequenceFinal,
		),
	)
	hashType := uint32(crypto.SigHashAll | crypto.SigHashForkID)

	// Single signature case:
	sigData, err := v.spender.SignStep(0, v.keyStore, nil, hashType, scriptPubKey, 1)
	assert.Nil(t, err)
	// <signature> <pubkey>
	assert.Equal(t, len(sigData), 2)
	assert.Equal(t, sigData[1], v.pubKeys[0].ToBytes())
}

func TestSignStepP2SH(t *testing.T) {
	v := initVar()

	// Create a P2SHLockingScript script
	pubKey := script.NewEmptyScript()
	pubKey.PushSingleData(v.pubKeys[0].ToBytes())
	pubKey.PushOpCode(opcodes.OP_CHECKSIG)

	pubKeyHash160 := util.Hash160(pubKey.GetData())

	scriptPubKey := script.NewEmptyScript()
	scriptPubKey.PushOpCode(opcodes.OP_HASH160)
	scriptPubKey.PushSingleData(pubKeyHash160)
	scriptPubKey.PushOpCode(opcodes.OP_EQUAL)

	// Add locking script to prevHolder
	v.prevHolder.AddTxOut(txout.NewTxOut(0, scriptPubKey))

	v.spender.AddTxIn(
		txin.NewTxIn(
			outpoint.NewOutPoint(v.prevHolder.GetHash(), 0),
			script.NewEmptyScript(),
			script.SequenceFinal,
		),
	)

	hashType := uint32(crypto.SigHashAll | crypto.SigHashForkID)

	// Single signature case:
	sigData, err := v.spender.SignStep(0, v.keyStore, pubKey, hashType, scriptPubKey, 1)
	assert.Nil(t, err)
	// <signature> <redeemscript>
	assert.Equal(t, len(sigData), 2)
	assert.Equal(t, sigData[1], pubKey.GetData())
}

func TestSignStepMultiSig(t *testing.T) {
	v := initVar()

	// Hardest case: Multisig 2-of-3
	// the stack like this: 2 << <pubKey1> << <pubKey2> << <pubKey3> << 3 << OP_CHECKMULTISIG
	scriptPubKey := script.NewEmptyScript()
	scriptPubKey.PushInt64(2)
	for i := 0; i < 3; i++ {
		scriptPubKey.PushSingleData(v.pubKeys[i].ToBytes())
	}
	scriptPubKey.PushInt64(3)
	scriptPubKey.PushOpCode(opcodes.OP_CHECKMULTISIG)

	// Add locking script to prevHolder
	v.prevHolder.AddTxOut(txout.NewTxOut(0, scriptPubKey))

	v.spender.AddTxIn(
		txin.NewTxIn(
			outpoint.NewOutPoint(v.prevHolder.GetHash(), 0),
			script.NewEmptyScript(),
			script.SequenceFinal,
		),
	)

	hashType := uint32(crypto.SigHashAll | crypto.SigHashForkID)

	// Multiple signature case:
	sigData, err := v.spender.SignStep(0, v.keyStore, nil, hashType, scriptPubKey, 1)
	assert.Nil(t, err)
	// <OP_0> <signature0> ... <signatureM>
	assert.Equal(t, len(sigData), 3)
	assert.Equal(t, sigData[0], []byte{})
}

func TestUpdateInScript(t *testing.T) {
	scriptSig := script.NewEmptyScript()
	scriptSig.PushSingleData([]byte{0})

	txPrev := NewTx(0, DefaultVersion)

	tx := NewTx(0, DefaultVersion)
	tx.AddTxIn(
		txin.NewTxIn(
			outpoint.NewOutPoint(txPrev.GetHash(), 0),
			script.NewEmptyScript(),
			script.SequenceFinal,
		),
	)

	// update scriptSig for an valid index
	err := tx.UpdateInScript(0, scriptSig)
	assert.Nil(t, err)
	if !reflect.DeepEqual(tx.GetIns()[0].GetScriptSig(), scriptSig) {
		t.Error("SIGNATURE NOT EXPECTED")
	}

	// update scriptSig for an invalid index
	err = tx.UpdateInScript(1, scriptSig)
	assert.NotNil(t, err)
}

func TestInsertTxOut(t *testing.T) {
	v := initVar()

	scriptPubKey0 := script.NewEmptyScript()
	scriptPubKey0.PushInt64(2)
	for i := 0; i < 3; i++ {
		scriptPubKey0.PushSingleData(v.pubKeys[i].ToBytes())
	}
	scriptPubKey0.PushInt64(3)
	scriptPubKey0.PushOpCode(opcodes.OP_CHECKMULTISIG)

	scriptPubKey1 := script.NewEmptyScript()
	scriptPubKey1.PushOpCode(opcodes.OP_DUP)
	scriptPubKey1.PushOpCode(opcodes.OP_HASH160)
	scriptPubKey1.PushSingleData(btcutil.Hash160(v.pubKeys[0].ToBytes()))
	scriptPubKey1.PushOpCode(opcodes.OP_EQUALVERIFY)
	scriptPubKey1.PushOpCode(opcodes.OP_CHECKSIG)

	pubKey := script.NewEmptyScript()
	pubKey.PushSingleData(v.pubKeys[0].ToBytes())
	pubKey.PushOpCode(opcodes.OP_CHECKSIG)
	pubKeyHash160 := util.Hash160(pubKey.GetData())
	scriptPubKey2 := script.NewEmptyScript()
	scriptPubKey2.PushOpCode(opcodes.OP_HASH160)
	scriptPubKey2.PushSingleData(pubKeyHash160)
	scriptPubKey2.PushOpCode(opcodes.OP_EQUAL)

	scriptPubKey3 := script.NewEmptyScript()
	scriptPubKey3.PushSingleData([]byte{})

	txn := NewTx(0, DefaultVersion)
	txn.AddTxOut(txout.NewTxOut(0, scriptPubKey0))
	txn.AddTxOut(txout.NewTxOut(0, scriptPubKey1))
	assert.Equal(t, 2, txn.GetOutsCount())
	assert.Equal(t, scriptPubKey0, txn.GetTxOut(0).GetScriptPubKey())
	assert.Equal(t, scriptPubKey1, txn.GetTxOut(1).GetScriptPubKey())

	txn.InsertTxOut(1, txout.NewTxOut(0, scriptPubKey2))
	assert.Equal(t, 3, txn.GetOutsCount())
	assert.Equal(t, scriptPubKey0, txn.GetTxOut(0).GetScriptPubKey())
	assert.Equal(t, scriptPubKey2, txn.GetTxOut(1).GetScriptPubKey())
	assert.Equal(t, scriptPubKey1, txn.GetTxOut(2).GetScriptPubKey())

	txn.InsertTxOut(100, txout.NewTxOut(0, scriptPubKey3))
	assert.Equal(t, 4, txn.GetOutsCount())
	assert.Equal(t, scriptPubKey3, txn.GetTxOut(3).GetScriptPubKey())
}
