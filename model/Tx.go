package model

import (
	"encoding/binary"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
	"io"
)

type Tx struct {
	Hash     utils.Hash
	LockTime uint32
	Version  int32
	Ins      []*TxIn
	Outs     []*TxOut
}

const (
	MaxTxInSequenceNum uint32 = 0xffffffff
	FreeListMaxItems          = 12500
	MaxMessagePayload         = 32 * 1024 * 1024
	MinTxInPayload            = 9 + utils.HashSize
	MaxTxInPerMessage         = (MaxMessagePayload / MinTxInPayload) + 1
	TxVersion                 = 1
)

var scriptPool ScriptFreeList = make(chan []byte, FreeListMaxItems)

func (tx *Tx) AddTxIn(txIn *TxIn) {
	tx.Ins = append(tx.Ins, txIn)
}

func (tx *Tx) AddTxOut(txOut *TxOut) {
	tx.Outs = append(tx.Outs, txOut)
}

func (tx *Tx) RemoveTxIn(txIn *TxIn) {

}

func (tx *Tx) RemoveTxOut(txOut *TxOut) {

}

func (tx *Tx) SerializeSize() int {
	// Version 4 bytes + LockTime 4 bytes + Serialized varint size for the
	// number of transaction inputs and outputs.
	n := 8 + utils.VarIntSerializeSize(uint64(len(tx.Ins))) + utils.VarIntSerializeSize(uint64(len(tx.Outs)))
	for _, txIn := range tx.Ins {
		n += txIn.SerializeSize()
	}
	for _, txOut := range tx.Outs {
		n += txOut.SerializeSize()
	}
	return n
}

func (tx *Tx) Serialize(writer io.Writer, pver uint32) error {
	err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, uint32(tx.Version))
	if err != nil {
		return err
	}
	count := uint64(len(tx.Ins))
	err = utils.WriteVarInt(writer, pver, count)
	if err != nil {
		return err
	}
	for _, ti := range tx.Ins {
		err := ti.Serialize(writer, pver, tx.Version)
		if err != nil {
			return err
		}
	}
	count = uint64(len(tx.Outs))
	err = utils.WriteVarInt(writer, pver, count)
	if err != nil {
		return err
	}
	for _, txOut := range tx.Outs {
		err := txOut.Serialize(writer, pver, tx.Version)
		if err != nil {
			return err
		}
	}
	return utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, tx.LockTime)

}

func (tx *Tx) Deserialize(reader io.Reader, pver uint32) (err error) {
	version, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return err
	}
	tx.Version = int32(version)
	count, err := utils.ReadVarInt(reader, pver)
	if err != nil {
		return err
	}
	if count > uint64(MaxTxInPerMessage) {
		err = errors.Errorf("too many input tx to fit into max message size [count %d , max %d]", count, MaxTxInPerMessage)
		return
	}
	var totalScriptSize uint64

	txIns := make([]TxIn, count)
	tx.Ins = make([]*TxIn, count)
	for i := uint64(0); i < count; i++ {
		txIn := &txIns[i]
		tx.Ins[i] = txIn
		err = txIn.Deserialize(reader, pver, tx.Version)
		if err != nil {
			tx.returnScriptBuffers()
			return
		}
		totalScriptSize += uint64(len(txIn.ScriptSig))
	}
	count, err = utils.ReadVarInt(reader, pver)
	if err != nil {
		tx.returnScriptBuffers()
		return
	}

	txOuts := make([]TxOut, count)
	tx.Outs = make([]*TxOut, count)
	for i := uint64(0); i < count; i++ {
		// The pointer is set now in case a script buffer is borrowed
		// and needs to be returned to the pool on error.
		txOut := &txOuts[i]
		tx.Outs[i] = txOut
		err = txOut.Deserialize(reader, pver, tx.Version)
		if err != nil {
			tx.returnScriptBuffers()
			return
		}
		totalScriptSize += uint64(len(txOut.OutScript))
	}

	tx.LockTime, err = utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		tx.returnScriptBuffers()
		return
	}
	tx.returnScriptBuffers()
	return

}
func (tx *Tx) Check() bool {
	return true
}

func (tx *Tx) returnScriptBuffers() {
	for _, txIn := range tx.Ins {
		if txIn == nil || txIn.ScriptSig == nil {
			continue
		}
		scriptPool.Return(txIn.ScriptSig)
	}
	for _, txOut := range tx.Outs {
		if txOut == nil || txOut.OutScript == nil {
			continue
		}
		scriptPool.Return(txOut.OutScript)
	}
}

func NewTx() *Tx {
	return &Tx{LockTime: 0, Version: TxVersion}
}
