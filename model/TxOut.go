package model

import (
	"encoding/binary"
	"github.com/btcboost/copernicus/protocol"
	"github.com/btcboost/copernicus/utils"
	"io"
)

type TxOut struct {
	Value     int64
	OutScript []byte
}

func (txOut *TxOut) SerializeSize() int {
	return 8 + utils.VarIntSerializeSize(uint64(len(txOut.OutScript))) + len(txOut.OutScript)
}

func (txOut *TxOut) Deserialize(reader io.Reader, pver uint32, version int32) error {
	err := protocol.ReadElement(reader, &txOut.Value)
	if err != nil {
		return err
	}
	txOut.OutScript, err = ReadScript(reader, pver, MaxMessagePayload, "tx output script")
	return err
}
func (txOut *TxOut) Serialize(writer io.Writer, pver uint32, version int32) error {
	err := utils.BinarySerializer.PutUint64(writer, binary.LittleEndian, uint64(txOut.Value))
	if err != nil {
		return err
	}
	return utils.WriteVarBytes(writer, pver, txOut.OutScript)

}
func (txOut *TxOut) Check() bool {
	return true
}

func NewTxOut(value int64, pkScript []byte) *TxOut {
	txOut := TxOut{
		Value:     value,
		OutScript: pkScript,
	}
	return &txOut
}
