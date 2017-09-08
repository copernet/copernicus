package model

import (
	"encoding/binary"
	"io"

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

func (txOut *TxOut) Deserialize(reader io.Reader, version int32) error {
	err := protocol.ReadElement(reader, &txOut.Value)
	if err != nil {
		return err
	}
	bytes, err := ReadScript(reader, MaxMessagePayload, "tx output script")
	txOut.Script = NewScriptRaw(bytes)
	return err
}
func (txOut *TxOut) Serialize(writer io.Writer, version int32) error {
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

func NewTxOut(value int64, pkScript []byte) *TxOut {
	txOut := TxOut{
		Value:  value,
		Script: NewScriptRaw(pkScript),
	}
	return &txOut
}
