package tx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

/*
import (
	"bytes"
	"encoding/binary"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/utils"
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

}*/
/*
func GetScriptBytes(script *Script) (bytes []byte, err error) {
	stk, err := script.ParseScript()
	if err != nil {
		return
	}
	bytes = make([]byte, 0, len(stk))
	for i := 0; i < len(stk); i++ {
		/** Serialize the passed scriptCode, skipping OP_CODESEPARATORs */
/*
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
*/

func SignatureHash(transaction *Tx, s *script.Script, hashType uint32, nIn int,
	money amount.Amount, flags uint32) (result util.Hash, err error) {

	var hashBuffer bytes.Buffer
	var sigHashAnyOneCanPay bool = false
	if hashType&crypto.SigHashAnyoneCanpay == crypto.SigHashAnyoneCanpay {
		sigHashAnyOneCanPay = true
	}
	var sigHashNone bool = false
	if hashType&crypto.SigHashMask == crypto.SigHashNone {
		sigHashNone = true
	}
	var sigHashSingle bool = false
	if hashType&crypto.SigHashMask == crypto.SigHashSingle {
		sigHashSingle = true
	}
	if hashType&crypto.SigHashForkID == crypto.SigHashForkID &&
		flags&script.ScriptEnableSigHashForkId == script.ScriptEnableSigHashForkId {
		var hashPrevouts util.Hash
		var hashSequence util.Hash
		var hashOutputs util.Hash

		if !sigHashAnyOneCanPay {
			hashPrevouts = GetPreviousOutHash(transaction)
		}
		if !sigHashAnyOneCanPay && !sigHashSingle && !sigHashNone {
			hashSequence = GetSequenceHash(transaction)
		}
		if !sigHashSingle && !sigHashNone {
			hashOutputs, _ = GetOutputsHash(transaction.GetOuts())
		} else if sigHashSingle && nIn < len(transaction.GetOuts()) {
			hashOutputs, _ = GetOutputsHash(transaction.GetOuts()[nIn : nIn+1])
		}

		util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, uint32(transaction.GetVersion()))
		hashBuffer.Write(hashPrevouts[:])
		hashBuffer.Write(hashSequence[:])
		transaction.GetIns()[nIn].PreviousOutPoint.Encode(&hashBuffer)
		err = s.Serialize(&hashBuffer)
		if err != nil {
			return
		}
		//input preout amount
		util.BinarySerializer.PutUint64(&hashBuffer, binary.LittleEndian, uint64(money))
		util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetIns()[nIn].Sequence)
		hashBuffer.Write(hashOutputs[:])
		util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetLockTime())
		util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, hashType)

		result = util.Sha256Hash(hashBuffer.Bytes())
		return
	}
	// The SigHashSingle signature type signs only the corresponding input
	// and output (the output with the same index number as the input).
	//
	// Since transactions can have more inputs than outputs, this means it
	// is improper to use SigHashSingle on input indices that don't have a
	// corresponding output.
	//
	// A bug in the original Satoshi client implementation means specifying
	// an index that is out of range results in a signature hash of 1 (as a
	// uint256 little endian).  The original intent appeared to be to
	// indicate failure, but unfortunately, it was never checked and thus is
	// treated as the actual signature hash.  This buggy behavior is now
	// part of the consensus and a hard fork would be required to fix it.
	//
	// Due to this, care must be taken by software that creates transactions
	// which make use of SigHashSingle because it can lead to an extremely
	// dangerous situation where the invalid inputs will end up signing a
	// hash of 1.  This in turn presents an opportunity for attackers to
	// cleverly construct transactions which can steal those coins provided
	// they can reuse signatures.
	ins := transaction.GetIns()
	insLen := len(ins)
	outs := transaction.GetOuts()
	outsLen := len(outs)
	if sigHashSingle && nIn >= outsLen {
		return util.HashOne, nil
	}
	if nIn >= insLen {
		return util.HashOne, nil
	}

	util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, uint32(transaction.GetVersion()))
	var inputsCount int
	if sigHashAnyOneCanPay {
		inputsCount = 1
	} else {
		inputsCount = insLen
	}
	util.WriteVarInt(&hashBuffer, uint64(inputsCount))

	ss := s.RemoveOpcode(opcodes.OP_CODESEPARATOR)

	// encode tx.inputs
	var i int
	for i = 0; i < inputsCount; i++ {
		if sigHashAnyOneCanPay {
			ins[nIn].PreviousOutPoint.Encode(&hashBuffer)
			ss.Serialize(&hashBuffer)
			util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[nIn].Sequence)
		} else {
			ins[i].PreviousOutPoint.Encode(&hashBuffer)
			if i != nIn {
				// push empty script
				util.WriteVarInt(&hashBuffer, 0)
				if sigHashSingle || sigHashNone {
					// push empty sequence
					util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, 0)
				} else {
					util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[i].Sequence)
				}
			} else {
				ss.Serialize(&hashBuffer)
				util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[i].Sequence)
			}
		}
	}
	//log.Debug("SignatureHash: after inputs serialize, buf is: %s", hex.EncodeToString(hashBuffer.Bytes()))
	// encode tx.outs
	var outsCount int
	if sigHashNone {
		outsCount = 0
	} else {
		if sigHashSingle {
			outsCount = nIn + 1
		} else {
			outsCount = outsLen
		}
	}
	util.WriteVarInt(&hashBuffer, uint64(outsCount))
	for m := 0; m < outsCount; m++ {
		if sigHashSingle && m != nIn {
			to := txout.NewTxOut(-1, nil)
			to.Encode(&hashBuffer)
		} else {
			outs[m].Encode(&hashBuffer)
		}
	}

	// encode tx.locktime
	util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetLockTime())
	util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, hashType)
	//log.Debug("SignatureHash buf: %s", hex.EncodeToString(hashBuffer.Bytes()))
	result = util.DoubleSha256Hash(hashBuffer.Bytes())
	return
}

func GetPreviousOutHash(tx *Tx) (h util.Hash) {
	ins := tx.GetIns()
	var bPreOut bytes.Buffer
	for _, e := range ins {
		e.PreviousOutPoint.Encode(&bPreOut)
	}
	h = util.Sha256Hash(bPreOut.Bytes())
	return
}

func GetSequenceHash(tx *Tx) (h util.Hash) {
	ins := tx.GetIns()
	buf := make([]byte, 0, 4*len(ins))
	for _, e := range ins {
		tempbuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(tempbuf, e.Sequence)
		buf = append(buf, tempbuf...)
	}
	h = util.Sha256Hash(buf)
	return
}

func GetOutputsHash(outs []*txout.TxOut) (h util.Hash, err error) {
	var bOut bytes.Buffer
	for _, e := range outs {
		err = e.Serialize(&bOut)
		if err != nil {
			return
		}
	}
	h = util.Sha256Hash(bOut.Bytes())
	return
}

func CheckSig(signHash util.Hash, vchSigIn []byte, vchPubKey []byte) bool {
	if len(vchPubKey) == 0 {
		return false
	}
	if len(vchSigIn) == 0 {
		return false
	}
	publicKey, err := crypto.ParsePubKey(vchPubKey)
	if err != nil {
		return false
	}

	sign, err := crypto.ParseDERSignature(vchSigIn)
	if err != nil {
		return false
	}
	uncompressedPubKey := publicKey.SerializeUncompressed()
	log.Debug("sig:%s, hash:%s, pubkey:%s, uncompressedPubKey:%s", hex.EncodeToString(vchSigIn),
		hex.EncodeToString(signHash[:]), hex.EncodeToString(vchPubKey), hex.EncodeToString(uncompressedPubKey))
	if !sign.EcdsaNormalize() {
		return false
	}
	ret := sign.Verify(signHash.GetCloneBytes(), publicKey)
	if !ret {
		return false
	}

	return true

}

//
//func verifySignature(vchSig []byte, pubkey *crypto.PublicKey, sigHash util.Hash) (bool, error) {
//	sign, err := crypto.ParseDERSignature(vchSig)
//	if err != nil {
//		return false, err
//	}
//	result := sign.Verify(sigHash.GetCloneBytes(), pubkey)
//	return result, nil
//}
