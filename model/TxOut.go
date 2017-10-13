package model

import (
	"encoding/binary"
	"io"

	"encoding/hex"
	"fmt"

	"github.com/btcboost/copernicus/protocol"
	"github.com/btcboost/copernicus/utils"
)

type TxOut struct {
	Value      int64
	SigOpCount int
	Script     *Script
}

func (txOut *TxOut) SerializeSize() int {
	if txOut.Script == nil {
		return 8
	}
	return 8 + utils.VarIntSerializeSize(uint64(txOut.Script.Size())) + txOut.Script.Size()
}

func (txOut *TxOut) IsDust(minRelayTxFee utils.FeeRate) bool {
	return txOut.Value < txOut.GetDustThreshold(minRelayTxFee)
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
	if txOut.Script.IsUnspendable() {
		return 0
	}
	size := txOut.SerializeSize()
	size += (32 + 4 + 1 + 107 + 4)
	return 3 * minRelayTxFee.GetFee(size)
}

func (txOut *TxOut) Deserialize(reader io.Reader) error {
	err := protocol.ReadElement(reader, &txOut.Value)
	if err != nil {
		return err
	}
	bytes, err := ReadScript(reader, MaxMessagePayload, "tx output script")
	txOut.Script = NewScriptRaw(bytes)
	return err
}

func (txOut *TxOut) Serialize(writer io.Writer) error {
	if txOut.Script == nil {
		return nil
	}
	err := utils.BinarySerializer.PutUint64(writer, binary.LittleEndian, uint64(txOut.Value))
	if err != nil {
		return err
	}
	return utils.WriteVarBytes(writer, txOut.Script.bytes)
}

func (txOut *TxOut) Check() bool {
	return true
}

func (txOut *TxOut) SetNull() {
	txOut.Value = -1
	txOut.Script = nil
}

func (txOut *TxOut) IsNull() bool {
	return txOut.Value == -1
}

func (txOut *TxOut) String() string {
	return fmt.Sprintf("Value :%d Script:%s", txOut.Value, hex.EncodeToString(txOut.Script.bytes))

}

func NewTxOut(value int64, pkScript []byte) *TxOut {
	txOut := TxOut{
		Value:  value,
		Script: NewScriptRaw(pkScript),
	}
	return &txOut
}
