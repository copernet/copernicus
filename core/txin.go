package core

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/net/protocol"
	"github.com/btcboost/copernicus/utils"
)

type TxIn struct {
	PreviousOutPoint *OutPoint
	scriptSig        *Script
	Sequence         uint32 //todo ?
	SigOpCount       int
}

func (txIn *TxIn) SerializeSize() int {
	// Outpoint Hash 32 bytes + Outpoint Index 4 bytes + Sequence 4 bytes +
	// serialized VarInt size for the length of SignatureScript +
	// SignatureScript bytes.
	if txIn.scriptSig == nil {
		return 40
	}

	return 40 + utils.VarIntSerializeSize(uint64(txIn.scriptSig.Size())) + txIn.scriptSig.Size()
}

func (txIn *TxIn) Deserialize(reader io.Reader) error {
	err := txIn.PreviousOutPoint.Deserialize(reader)
	if err != nil {
		return err
	}
	bytes, err := ReadScript(reader, MaxMessagePayload, "tx input signature script")
	if err != nil {
		return err
	}
	txIn.scriptSig = NewScriptRaw(bytes)
	return protocol.ReadElement(reader, &txIn.Sequence)
}
func (txIn *TxIn) Serialize(writer io.Writer) error {
	var err error
	if txIn.PreviousOutPoint != nil {
		err = txIn.PreviousOutPoint.WriteOutPoint(writer)
		if err != nil {
			return err
		}
	}
	err = utils.WriteVarBytes(writer, txIn.scriptSig.bytes)
	if err != nil {
		return err
	}

	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, txIn.Sequence)
	return err
}

func (txIn *TxIn) CheckScript(state *ValidationState) bool {
	return txIn.scriptSig.CheckScriptSig(state)
}

func (txIn *TxIn) String() string {
	str := fmt.Sprintf("PreviousOutPoint: %s ", txIn.PreviousOutPoint.String())
	if txIn.scriptSig == nil {
		return fmt.Sprintf("%s , script:  , Sequence:%d ", str, txIn.Sequence)
	}
	return fmt.Sprintf("%s , script:%s , Sequence:%d ", str, hex.EncodeToString(txIn.script.bytes), txIn.Sequence)

}
/*
func (txIn *TxIn) Check() bool {
	return true
}
*/
func NewTxIn() *TxIn {
	txIn := TxIn{PreviousOutPoint: nil, scriptSig: nil, Sequence: MaxTxInSequenceNum}
	return &txIn
}
