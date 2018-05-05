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
	Script           *Script
	Sequence         uint32 //todo ?
	SigOpCount       int
}

func (txIn *TxIn) SerializeSize() int {
	// Outpoint Hash 32 bytes + Outpoint Index 4 bytes + Sequence 4 bytes +
	// serialized VarInt size for the length of SignatureScript +
	// SignatureScript bytes.
	if txIn.Script == nil {
		return 40
	}
	return 40 + utils.VarIntSerializeSize(uint64(txIn.Script.Size())) + txIn.Script.Size()
}

func (txIn *TxIn) Deserialize(reader io.Reader, version int32) error {
	err := txIn.PreviousOutPoint.Deserialize(reader)
	if err != nil {
		return err
	}
	bytes, err := ReadScript(reader, MaxMessagePayload, "tx input signature script")
	if err != nil {
		return err
	}
	txIn.Script = NewScriptRaw(bytes)
	return protocol.ReadElement(reader, &txIn.Sequence)
}
func (txIn *TxIn) Serialize(writer io.Writer, version int32) error {
	var err error
	if txIn.PreviousOutPoint != nil {
		err = txIn.PreviousOutPoint.WriteOutPoint(writer)
		if err != nil {
			return err
		}
	}
	err = utils.WriteVarBytes(writer, txIn.Script.bytes)
	if err != nil {
		return err
	}

	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, txIn.Sequence)
	return err
}

func (txIn *TxIn) String() string {
	str := fmt.Sprintf("PreviousOutPoint: %s ", txIn.PreviousOutPoint.String())
	if txIn.Script == nil {
		return fmt.Sprintf("%s , script:  , Sequence:%d ", str, txIn.Sequence)
	}
	return fmt.Sprintf("%s , script:%s , Sequence:%d ", str, hex.EncodeToString(txIn.Script.bytes), txIn.Sequence)

}
func (txIn *TxIn) Check() bool {
	return true
}

func NewTxIn(prevOut *OutPoint, pkScript []byte) *TxIn {
	txIn := TxIn{PreviousOutPoint: prevOut, Script: NewScriptRaw(pkScript), Sequence: MaxTxInSequenceNum}
	return &txIn
}
