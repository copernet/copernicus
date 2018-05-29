package txout

import (
	"encoding/hex"
	"fmt"
	"io"

	"encoding/binary"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/util/amount"
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
		return util.BinarySerializer.PutUint64(writer, binary.LittleEndian, 0)
	} else {
		return txOut.scriptPubKey.Encode(writer)
	}
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
	size += 32 + 4 + 1 + 107 + 4
	return 3 * minRelayTxFee.GetFee(int(size))
}

func (txOut *TxOut) CheckValue() error {
	if txOut.value < 0 {
		//state.Dos(100, false, RejectInvalid, "bad-txns-vout-negative", false, "")
		return errcode.New(errcode.TxErrRejectInvalid)
	}
	if txOut.value > amount.Amount(util.MaxMoney) {
		//state.Dos(100, false, RejectInvalid, "bad-txns-vout-toolarge", false, "")
		return errcode.New(errcode.TxErrRejectInvalid)
	}

	return nil
}

func (txOut *TxOut) CheckStandard() (pubKeyType int, err error) {
	pubKeyType, _, err = txOut.scriptPubKey.CheckScriptPubKeyStandard()
	if err != nil {
		return
	}
	if pubKeyType == script.ScriptNullData {
		if !conf.Cfg.Script.AcceptDataCarrier || uint(txOut.scriptPubKey.Size()) > conf.Cfg.Script.MaxDatacarrierBytes {
			return pubKeyType, errcode.New(errcode.ScriptErrNullData)
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

/*
  The TxOut can be spent or not according script, but don't care it is already been spent or not
*/
func (txOut *TxOut) IsSpendable() bool {
	return txOut.scriptPubKey.IsSpendable()
}

/*
  The TxOut already spent
*/
func (txOut *TxOut) IsSpent() bool {
	return true
}

func (txOut *TxOut) SetNull() {
	txOut.value = -1
	txOut.scriptPubKey = nil
}

func (txOut *TxOut) IsNull() bool {
	return txOut.value == -1
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
		scriptPubKey: scriptPubKey,
	}
	return &txOut
}
