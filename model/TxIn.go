package model

import "github.com/btccom/copernicus/utils"

type TxIn struct {
	PreviousOutPoint *OutPoint
	ScriptSig        []byte
	Sequence         uint32
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
