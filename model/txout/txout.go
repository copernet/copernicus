package txout

import (
	"io"

	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

type TxOut struct {
	value        amount.Amount
	scriptPubKey *script.Script
}

func (txOut *TxOut) SerializeSize() uint32 {
	return txOut.EncodeSize()
}

func (txOut *TxOut) Serialize(writer io.Writer) error {
	return txOut.Encode(writer)
}

func (txOut *TxOut) Unserialize(reader io.Reader) error {
	return txOut.Decode(reader)
}

func (txOut *TxOut) EncodeSize() uint32 {
	return 8 + txOut.scriptPubKey.EncodeSize()
}

func (txOut *TxOut) Encode(writer io.Writer) error {
	err := util.BinarySerializer.PutUint64(writer, binary.LittleEndian, uint64(txOut.value))
	if err != nil {
		return err
	}
	if txOut.scriptPubKey == nil {
		return util.WriteVarInt(writer, 0)
	}
	return txOut.scriptPubKey.Encode(writer)
}

func (txOut *TxOut) Decode(reader io.Reader) error {
	err := util.ReadElements(reader, &txOut.value)
	if err != nil {
		return err
	}
	bytes, err := script.ReadScript(reader, script.MaxMessagePayload, "tx output script")
	txOut.scriptPubKey = script.NewScriptRaw(bytes)
	return err
}

func (txOut *TxOut) IsDust(minRelayTxFee *util.FeeRate) bool {
	return txOut.value < amount.Amount(txOut.GetDustThreshold(minRelayTxFee))
}

func (txOut *TxOut) GetDustThreshold(minRelayTxFee *util.FeeRate) int64 {
	// "Dust" is defined in terms of CTransaction::minRelayTxFee, which has
	// units satoshis-per-kilobyte. If you'd pay more than 1/3 in fees to
	// spend something, then we consider it dust. A typical spendable
	// non-segwit txout is 34 bytes big, and will need a CTxIn of at least
	// 148 bytes to spend: so dust is a spendable txout less than
	// 546*minRelayTxFee/1000 (in satoshis). A typical spendable segwit
	// txout is 31 bytes big, and will need a CTxIn of at least 67 bytes to
	// spend: so dust is a spendable txout less than 294*minRelayTxFee/1000
	// (in satoshis).
	if txOut.scriptPubKey.IsUnspendable() {
		return 0
	}
	size := txOut.SerializeSize()
	size += 32 + 4 + 1 + 107 + 4 //=148
	return 3 * minRelayTxFee.GetFee(int(size))
}

func (txOut *TxOut) CheckValue() error { //3
	if !amount.MoneyRange(txOut.value) {
		log.Warn("bad txout value :%d", txOut.value)
		return errcode.NewError(errcode.RejectInvalid, "bad-txns-vout-out-range")
	}

	return nil
}

func (txOut *TxOut) CheckStandard() (pubKeyType int, err error) { //4
	//var pubKeys [][]byte
	pubKeyType, pubKeys, err := txOut.scriptPubKey.CheckScriptPubKeyStandard()
	if err != nil {
		return pubKeyType, errcode.New(errcode.TxErrRejectNonstandard)
	}
	if pubKeyType == script.ScriptMultiSig {
		opM := pubKeys[0][0]
		opN := pubKeys[len(pubKeys)-1][0]
		if opN < 1 || opN > 3 || opM < 1 || opM > opN {
			return pubKeyType, errcode.New(errcode.TxErrRejectNonstandard)
		}
	} else if pubKeyType == script.ScriptNullData {
		if !conf.Cfg.Script.AcceptDataCarrier || uint(txOut.scriptPubKey.Size()) > conf.Cfg.Script.MaxDatacarrierBytes {
			log.Debug("ScriptErrNullData")
			return pubKeyType, errcode.New(errcode.TxErrRejectNonstandard)
		}
	}

	return
}

func (txOut *TxOut) GetPubKeyType() (pubKeyType int, err error) {
	pubKeyType, _, err = txOut.scriptPubKey.CheckScriptPubKeyStandard()
	return
}

func (txOut *TxOut) GetValue() amount.Amount {
	return txOut.value
}
func (txOut *TxOut) SetValue(v amount.Amount) {
	txOut.value = v
}
func (txOut *TxOut) GetScriptPubKey() *script.Script {
	return txOut.scriptPubKey
}
func (txOut *TxOut) SetScriptPubKey(s *script.Script) {
	txOut.scriptPubKey = s
}

func (txOut *TxOut) IsCommitment(data []byte) bool {
	return txOut.scriptPubKey.IsCommitment(data)
}

// IsSpendable returns whether the TxOut can be spent or not,
// but doesn't care whether it has already been spent or not

func (txOut *TxOut) IsSpendable() bool {
	if txOut == nil || txOut.scriptPubKey == nil {
		return false
	}
	return txOut.scriptPubKey.IsSpendable()
}

func (txOut *TxOut) SetNull() {
	txOut.value = -1
	txOut.scriptPubKey = nil
}

func (txOut *TxOut) IsNull() bool {
	return txOut.value == -1 //&& txOut.scriptPubKey == nil
}
func (txOut *TxOut) String() string {
	return fmt.Sprintf("Value :%d Script:%s", txOut.value, hex.EncodeToString(txOut.scriptPubKey.GetData()))
}

func (txOut *TxOut) IsEqual(out *TxOut) bool {
	if txOut.value != out.value {
		return false
	}

	return txOut.scriptPubKey.IsEqual(out.scriptPubKey)
}

func NewTxOut(value amount.Amount, scriptPubKey *script.Script) *TxOut {
	txOut := TxOut{
		value:        value,
		scriptPubKey: nil,
	}
	if scriptPubKey != nil {
		txOut.scriptPubKey = script.NewScriptRaw(scriptPubKey.GetData())
	}

	return &txOut
}
