package txout

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/errcode"
)

type TxOut struct {
	value            int64
	scriptPubKey     *script.Script
	//SigOpCount      int64
}

func (txOut *TxOut) SerializeSize() int {
	if txOut.scriptPubKey == nil {
		return 8
	}
	return 8 + util.VarIntSerializeSize(uint64(txOut.scriptPubKey.Size())) + txOut.scriptPubKey.Size()
}

func (txOut *TxOut) IsDust(minRelayTxFee util.FeeRate) bool {
	return txOut.value < txOut.GetDustThreshold(minRelayTxFee)
}

func (txOut *TxOut) GetDustThreshold(minRelayTxFee util.FeeRate) int64 {
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
	return 3 * minRelayTxFee.GetFee(size)
}

func (txOut *TxOut) Encode(writer io.Writer) error {
	err := util.BinarySerializer.PutUint64(writer, binary.LittleEndian, uint64(txOut.value))
	if err != nil {
		return err
	}
	if txOut.scriptPubKey == nil {
		return util.BinarySerializer.PutUint64(writer, binary.LittleEndian, 0  )
	} else {
	    return util.WriteVarBytes(writer, txOut.scriptPubKey.GetData())
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
func (txOut *TxOut) Serialize(writer io.Writer) error {
	return nil
}

func (txOut *TxOut) Unserialize(reader io.Reader) error {
	return nil
}

func (txOut *TxOut) CheckValue() error {
	if txOut.value < 0 {
		//state.Dos(100, false, RejectInvalid, "bad-txns-vout-negative", false, "")
		return errcode.New(errcode.TxOutErrNegativeValue)
	}
	if txOut.value > util.MaxMoney {
		//state.Dos(100, false, RejectInvalid, "bad-txns-vout-toolarge", false, "")
		return errcode.New(errcode.TxOutErrTooLargeValue)
	}

	return nil
}

func (txOut *TxOut) CheckScript(allowLargeOpReturn bool) (succeed bool, pubKeyType int)  {
	succeed, pubKeyType = txOut.scriptPubKey.CheckScriptPubKey()

	if pubKeyType == script.ScriptNullData {
		/*
		if !AcceptDataCarrier {
			return false, pubKeyType
		}

		maxScriptSize uint32 = 0

		if allowLargeOpReturn {
			maxScriptSize = MaxOpReturnRelayLarge
		} else {
			maxScriptSize = MaxOpReturnRelay
		}
		if txOut.scriptPubKey.Size() > maxScriptSize {
			state.Dos(100, false, RejectInvalid, "scriptpubkey too large", false, "")
			return false, pubKeyType
		}*/
	}

	return
}

func (txOut *TxOut) GetValue() int64 {
	return txOut.value
}
func (txOut *TxOut) SetValue(v int64) {
	txOut.value=v
}
func (txOut *TxOut) GetScriptPubKey() *script.Script {
	return txOut.scriptPubKey
}
func (txOut *TxOut) SetScriptPubKey(s *script.Script)  {
	 txOut.scriptPubKey = s
}
/*
func (txOut *TxOut) Check() bool {
	return true
}
*/
/*
  The TxOut can be spent or not according script, but don't care it is already been spent or not
 */
func (txOut *TxOut) IsSpendable() bool {
	return true
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
	return fmt.Sprintf("Value :%d Script:%s", txOut.value, hex.EncodeToString(txOut.scriptPubKey.GetByteCodes()))
}

func (txOut *TxOut) IsEqual(out *TxOut) bool {
	if txOut.value != out.value {
		return false
	}

	return txOut.scriptPubKey.IsEqual(out.scriptPubKey)
}

func NewTxOut(value int64, scriptPubKey *script.Script) *TxOut {
	txOut := TxOut{
		value: value,
		scriptPubKey: scriptPubKey,
	}
	return &txOut
}
