package core

import (
	"bytes"
	"encoding/binary"

	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
)

type TxSignatureSerializer struct {
	txTo       *Tx
	script     *Script
	nIn        int
	hashSingle bool
	hashNone   bool
}

func GetPrevoutHash(tx *Tx) (utils.Hash, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 40*len(tx.Ins)))
	for i := 0; i < len(tx.Ins); i++ {
		outPoint := tx.Ins[i].PreviousOutPoint
		_, err := buf.Write(outPoint.Hash[:])
		if err != nil {
			return utils.Hash{}, err
		}
		utils.BinarySerializer.PutUint32(buf, binary.LittleEndian, outPoint.Index)
	}
	return crypto.DoubleSha256Hash(buf.Bytes()), nil

}

func GetSequenceHash(tx *Tx) (utils.Hash, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 8*len(tx.Ins)))
	for i := 0; i < len(tx.Ins); i++ {
		utils.BinarySerializer.PutUint32(buf, binary.LittleEndian, tx.Ins[i].Sequence)
	}
	return crypto.DoubleSha256Hash(buf.Bytes()), nil

}

func GetOutputsHash(tx *Tx) (utils.Hash, error) {
	size := 0
	for i := 0; i < len(tx.Outs); i++ {
		size += tx.Outs[i].SerializeSize()
	}
	buf := bytes.NewBuffer(make([]byte, 0, size))
	for i := 0; i < len(tx.Ins); i++ {
		tx.Outs[i].Serialize(buf)
	}
	return crypto.DoubleSha256Hash(buf.Bytes()), nil

}
func GetScriptBytes(script *Script) (bytes []byte, err error) {
	stk, err := script.ParseScript()
	if err != nil {
		return
	}
	bytes = make([]byte, 0, len(stk))
	for i := 0; i < len(stk); i++ {
		/** Serialize the passed scriptCode, skipping OP_CODESEPARATORs */
		parsedOpcode := stk[i]
		if parsedOpcode.opValue == OP_CODESEPARATOR {

		} else {
			bytes = append(bytes, parsedOpcode.opValue)
			bytes = append(bytes, parsedOpcode.data...)
		}

	}
	return
}

var NilScript = NewScriptRaw(make([]byte, 0))

func SignatureHash(tx *Tx, script *Script, hashType uint32, nIn int) (result utils.Hash, err error) {
	if (hashType&0x1f == crypto.SigHashSingle) &&
		nIn >= len(tx.Outs) {
		return utils.HashOne, nil
	}

	txCopy := tx.Copy()
	for i := range tx.Ins {
		if i == nIn {
			scriptBytes, _ := GetScriptBytes(script)
			txCopy.Ins[i].Script = NewScriptRaw(scriptBytes)
		} else {
			txCopy.Ins[i].Script = NilScript
		}
	}
	switch hashType & 0x1f {
	case crypto.SigHashNone:
		txCopy.Outs = make([]*TxOut, 0)
		for i := range txCopy.Ins {
			if nIn != i {
				txCopy.Ins[i].Sequence = 0
			}
		}
	case crypto.SigHashSingle:
		txCopy.Outs = txCopy.Outs[:nIn+1]
		for i := 0; i < nIn; i++ {
			txCopy.Outs[i].value = -1
			txCopy.Outs[i].scriptPubKey = NilScript
		}
		for i := range txCopy.Ins {
			if i != nIn {
				txCopy.Ins[i].Sequence = 0
			}
		}
	case crypto.SigHashAll:
	}
	if hashType&crypto.SigHashAnyoneCanpay != 0 {
		txCopy.Ins = txCopy.Ins[nIn : nIn+1]
	}
	buf := bytes.NewBuffer(make([]byte, 0, txCopy.SerializeSize()+4))
	txCopy.Serialize(buf)
	binary.Write(buf, binary.LittleEndian, hashType) //todo can't write int
	sha256 := crypto.DoubleSha256Bytes(buf.Bytes())
	result = utils.Hash{}
	result.SetBytes(sha256)
	return
}
