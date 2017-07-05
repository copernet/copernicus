package model

import (
	"encoding/binary"
	"github.com/btccom/copernicus/protocol"
	"github.com/btccom/copernicus/utils"
	"io"
)

type TxIn struct {
	PreviousOutPoint *OutPoint
	ScriptSig        []byte
	Sequence         uint32 //todo ?
}

func (txIn *TxIn) SerializeSize() int {
	// Outpoint Hash 32 bytes + Outpoint Index 4 bytes + Sequence 4 bytes +
	// serialized varint size for the length of SignatureScript +
	// SignatureScript bytes.
	return 40 + utils.VarIntSerializeSize(uint64(len(txIn.ScriptSig))) + len(txIn.ScriptSig)

}

func NewTxIn(prevOut *OutPoint, pkScript []byte) *TxIn {
	txIn := TxIn{PreviousOutPoint: prevOut, ScriptSig: pkScript, Sequence: MaxTxInSequenceNum}
	return &txIn
}

func (txIn *TxIn) ReadTxIn(reader io.Reader, pver uint32, version int32) error {
	err := txIn.PreviousOutPoint.ReadOutPoint(reader, pver, version)
	if err != nil {
		return err
	}
	txIn.ScriptSig, err = ReadScript(reader, pver, MaxmessagePayload, "tx input signature script")
	if err != nil {
		return err
	}
	return protocol.ReadElement(reader, &txIn.Sequence)

}
func (txIn *TxIn) WriteTxIn(writer io.Writer, pver uint32, version int32) error {
	err := txIn.PreviousOutPoint.WriteOutPoint(writer, pver, version)
	if err != nil {
		return err
	}
	err = utils.WriteVarBytes(writer, pver, txIn.ScriptSig)
	if err != nil {
		return err
	}

	return utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, txIn.Sequence)
}
