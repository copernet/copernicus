package scripts

import (
	"bytes"
	"encoding/binary"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type TxSignatureSerializer struct {
	txTo       *model.Tx
	script     *CScript
	nIn        int
	hashSingle bool
	hashNone   bool
}

func GetPrevoutHash(tx *model.Tx) (utils.Hash, error) {
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

func GetSequenceHash(tx *model.Tx) (utils.Hash, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 8*len(tx.Ins)))
	for i := 0; i < len(tx.Ins); i++ {
		utils.BinarySerializer.PutUint32(buf, binary.LittleEndian, tx.Ins[i].Sequence)
	}
	return core.DoubleSha256Hash(buf.Bytes()), nil

}

func GetOutputsHash(tx *model.Tx) (utils.Hash, error) {
	size := 0
	for i := 0; i < len(tx.Outs); i++ {
		size += tx.Outs[i].SerializeSize()
	}
	buf := bytes.NewBuffer(make([]byte, 0, size))
	for i := 0; i < len(tx.Ins); i++ {
		tx.Outs[i].Serialize(buf, 0, 1) //todo pver and version
	}
	return core.DoubleSha256Hash(buf.Bytes()), nil

}
func GetScriptBytes(script *CScript) (bytes []byte, err error) {
	stk, err := script.ParseScript()
	if err != nil {
		return
	}
	bytes = make([]byte, 0, len(stk))
	for i := 0; i < len(stk); i++ {
		/** Serialize the passed scriptCode, skipping OP_CODESEPARATORs */
		if stk[i][0] == OP_CODESEPARATOR {

		} else {
			bytes = append(bytes, stk[i]...)
		}

	}
	return
}

func SignatureHash(tx *model.Tx, script *CScript, hashType int, nIn int) (result utils.Hash, err error) {
	if (hashType&0x1f == core.SIGHASH_SINGLE) &&
		nIn >= len(tx.Outs) {
		return utils.HashOne, nil
	}

	return
}
