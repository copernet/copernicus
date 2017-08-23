package model

import (
	"encoding/binary"
	"github.com/btcboost/copernicus/protocol"
	"github.com/btcboost/copernicus/utils"
	"io"
)

type TxIn struct {
	PreviousOutPoint *OutPoint
	Script           []byte
	Sequence         uint32 //todo ?
	SigOpCount       int
}

func (txIn *TxIn) SerializeSize() int {
	// Outpoint Hash 32 bytes + Outpoint Index 4 bytes + Sequence 4 bytes +
	// serialized VarInt size for the length of SignatureScript +
	// SignatureScript bytes.
	return 40 + utils.VarIntSerializeSize(uint64(len(txIn.Script))) + len(txIn.Script)

}

func (txIn *TxIn) Deserialize(reader io.Reader, version int32) error {
	err := txIn.PreviousOutPoint.ReadOutPoint(reader, version)
	if err != nil {
		return err
	}
	txIn.Script, err = ReadScript(reader, MaxMessagePayload, "tx input signature script")
	if err != nil {
		return err
	}
	return protocol.ReadElement(reader, &txIn.Sequence)

}
func (txIn *TxIn) Serialize(writer io.Writer, version int32) error {
	err := txIn.PreviousOutPoint.WriteOutPoint(writer, 0, version)
	if err != nil {
		return err
	}
	err = utils.WriteVarBytes(writer, txIn.Script)
	if err != nil {
		return err
	}

	return utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, txIn.Sequence)
}

func (txIn *TxIn) Check() bool {
	return true
}

func NewTxIn(prevOut *OutPoint, pkScript []byte) *TxIn {
	txIn := TxIn{PreviousOutPoint: prevOut, Script: pkScript, Sequence: MaxTxInSequenceNum}
	return &txIn
}
