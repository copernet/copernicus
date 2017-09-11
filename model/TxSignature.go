package model

import (
	"bytes"
	"encoding/binary"

	"encoding/hex"
	"fmt"

	"github.com/btcboost/copernicus/core"
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
	return core.DoubleSha256Hash(buf.Bytes()), nil

}

func GetSequenceHash(tx *Tx) (utils.Hash, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 8*len(tx.Ins)))
	for i := 0; i < len(tx.Ins); i++ {
		utils.BinarySerializer.PutUint32(buf, binary.LittleEndian, tx.Ins[i].Sequence)
	}
	return core.DoubleSha256Hash(buf.Bytes()), nil

}

func GetOutputsHash(tx *Tx) (utils.Hash, error) {
	size := 0
	for i := 0; i < len(tx.Outs); i++ {
		size += tx.Outs[i].SerializeSize()
	}
	buf := bytes.NewBuffer(make([]byte, 0, size))
	for i := 0; i < len(tx.Ins); i++ {
		tx.Outs[i].Serialize(buf, 1) //todo pver and version
	}
	return core.DoubleSha256Hash(buf.Bytes()), nil

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

func SignatureHash(tx *Tx, script *Script, hashType uint32, nIn int) (result utils.Hash, err error) {
	if (hashType&0x1f == core.SIGHASH_SINGLE) &&
		nIn >= len(tx.Outs) {
		return utils.HashOne, nil
	}
	txCopy := tx.Copy()
	for i := range tx.Ins {
		if i == nIn {
			scriptBytes, _ := GetScriptBytes(script)
			txCopy.Ins[i].Script = NewScriptRaw(scriptBytes)
		} else {
			txCopy.Ins[i].Script = nil
		}
	}
	switch hashType & 0x1f {
	case core.SIGHASH_NONE:
		txCopy.Outs = make([]*TxOut, 0)
		for i := range txCopy.Ins {
			if nIn != i {
				txCopy.Ins[i].Sequence = 0
			}
		}
	case core.SIGHASH_SINGLE:
		txCopy.Outs = txCopy.Outs[:nIn+1]
		for i := 0; i < nIn; i++ {
			txCopy.Outs[i].Value = -1
			txCopy.Outs[i].Script = nil
		}
		for i := range txCopy.Ins {
			if i != nIn {
				txCopy.Ins[i].Sequence = 0
			}
		}
	case core.SIGHASH_ALL:

	}
	if hashType&core.SIGHASH_ANYONECANPAY != 0 {
		txCopy.Ins = tx.Ins[nIn : nIn+1]
	}

	buf := bytes.NewBuffer(make([]byte, 0, txCopy.SerializeSize()+4))
	txCopy.Serialize(buf)
	binary.Write(buf, binary.LittleEndian, hashType) //todo can't write int
	for i, in := range tx.Ins {
		fmt.Printf("in: %d , %s , %s \n", i, hex.EncodeToString(in.Script.bytes), hex.EncodeToString(in.PreviousOutPoint.Hash.GetCloneBytes()))
	}
	fmt.Printf("tx :%s\n", tx)
	fmt.Printf("tx string :%s\n", tx.String())
	fmt.Printf("tx :%s\n", hex.EncodeToString(buf.Bytes()))
	sha256 := core.DoubleSha256Bytes(buf.Bytes())
	fmt.Printf("tx hash :%s\n", hex.EncodeToString(sha256))
	result = utils.Hash{}
	result.SetBytes(sha256)
	return
}
