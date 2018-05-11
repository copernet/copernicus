package txout

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/util"
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

func (txOut *TxOut) GetDustThreshold(minRelayTxFee utils.FeeRate) int64 {
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

func (txOut *TxOut) Serialize(writer io.Writer) error {
	if txOut.scriptPubKey == nil {
		return nil
	}
	err := util.BinarySerializer.PutUint64(writer, binary.LittleEndian, uint64(txOut.value))
	if err != nil {
		return err
	}
	return util.WriteVarBytes(writer, txOut.scriptPubKey.GetByteCodes())
}

func (txOut *TxOut) Unserialize(reader io.Reader) error {
	err := protocol.ReadElement(reader, &txOut.value)
	if err != nil {
		return err
	}
	bytes, err := ReadScript(reader, MaxMessagePayload, "tx output script")
	txOut.scriptPubKey = NewScriptRaw(bytes)
	return err
}

func (txOut *TxOut) CheckValue(state *ValidationState) bool {
	if txOut.value < 0 {
		state.Dos(100, false, RejectInvalid, "bad-txns-vout-negative", false, "")
		return false
	}
	if txOut.value > MaxMoney {
		state.Dos(100, false, RejectInvalid, "bad-txns-vout-toolarge", false, "")
		return false
	}

	return true
}

func (txOut *TxOut) CheckScript(state *ValidationState, allowLargeOpReturn bool) (succeed bool, pubKeyType int)  {
	succeed, pubKeyType = txOut.scriptPubKey.CheckScriptPubKey(state)

	if pubKeyType == ScriptNullData {
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
		}
	}

	return
}

func (txOut *TxOut) GetValue() int64 {
	return txOut.value
}
func (txOut *TxOut) SetValue(v int64) {
	txOut.value=v
}
func (txOut *TxOut) GetScriptPubKey() *Script {
	return txOut.scriptPubKey
}
func (txOut *TxOut) SetScriptPubKey(s *Script)  {
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

func NewTxOut() *TxOut {
	txOut := TxOut{
		value:  -1,
		scriptPubKey: nil,
	}
	return &txOut
}
