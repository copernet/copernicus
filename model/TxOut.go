package model

import (
	"encoding/binary"
	"github.com/btccom/copernicus/protocol"
	"github.com/btccom/copernicus/utils"
	"io"
)

type TxOut struct {
	Address   string
	Value     int64
	OutScript []byte
}

func (txOut *TxOut) SerializeSize() int {
	return 8 + utils.VarIntSerializeSize(uint64(len(txOut.OutScript))) + len(txOut.OutScript)
}
func NewTxOut(value int64, pkScript []byte) *TxOut {
	txOut := TxOut{
		Value:     value,
		OutScript: pkScript,
	}
	return &txOut
}
func (txOut *TxOut) ReadTxOut(reader io.Reader, pver uint32, version int32) error {
	err := protocol.ReadElement(reader, &txOut.Value)
	if err != nil {
		return err
	}
	txOut.OutScript, err = ReadScript(reader, pver, MaxmessagePayload, "tx output script")
	return err
}
func (txOut *TxOut) WriteTxOut(writer io.Writer, pver uint32, version int32) error {
	err := utils.BinarySerializer.PutUint64(writer, binary.LittleEndian, uint64(txOut.Value))
	if err != nil {
		return err
	}
	return utils.WriteVarBytes(writer, pver, txOut.OutScript)

}
