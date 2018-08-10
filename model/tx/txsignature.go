package tx

import (
	"bytes"
	"encoding/binary"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/copernet/copernicus/log"
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
	var sigHashAnyOneCanPay = false
	if hashType&crypto.SigHashAnyoneCanpay == crypto.SigHashAnyoneCanpay {
		sigHashAnyOneCanPay = true
	}
	var sigHashNone = false
	if hashType&crypto.SigHashMask == crypto.SigHashNone {
		sigHashNone = true
	}
	var sigHashSingle = false
	if hashType&crypto.SigHashMask == crypto.SigHashSingle {
		sigHashSingle = true
	}
	if flags&script.ScriptEnableReplayProtection == script.ScriptEnableReplayProtection {
		// Legacy chain's value for fork id must be of the form 0xffxxxx.
		// By xoring with 0xdead, we ensure that the value will be different
		// from the original one, even if it already starts with 0xff.
		newForkValue := (hashType >> 8) ^ 0xdead
		hashType = hashType&0xff | ((0xff0000 | newForkValue) << 8)
	}

	if hashType&crypto.SigHashForkID == crypto.SigHashForkID &&
		flags&script.ScriptEnableSigHashForkID == script.ScriptEnableSigHashForkID {
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

		err := util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, uint32(transaction.GetVersion()))
		if err != nil {
			log.Error("txSignature:push transaction.GetVersion() to hashBuffer failed.")
		}
		_, err = hashBuffer.Write(hashPrevouts[:])
		if err != nil {
			log.Error("txSignature:write hashPrevouts failed.")
		}
		_, err = hashBuffer.Write(hashSequence[:])
		if err != nil {
			log.Error("txSignature:write hashSequence failed.")
		}
		err = transaction.GetIns()[nIn].PreviousOutPoint.Encode(&hashBuffer)
		if err != nil {
			log.Error("txSignature:Previous OutPoint encode failed.")
		}
		err = s.Serialize(&hashBuffer)
		if err != nil {
			return
		}
		//input preout amount
		err = util.BinarySerializer.PutUint64(&hashBuffer, binary.LittleEndian, uint64(money))
		if err != nil {
			log.Error("txSignature:push money to hashBuffer failed.")
		}
		err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetIns()[nIn].Sequence)
		if err != nil {
			log.Error("txSignature:push transaction.GetIns()[nIn].Sequence to hashBuffer failed.")
		}
		_, err = hashBuffer.Write(hashOutputs[:])
		if err != nil {
			log.Error("txSignature:write hashOutputs failed.")
		}
		err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetLockTime())
		if err != nil {
			log.Error("txSignature:push transaction.GetLockTime() to hashBuffer failed.")
		}
		err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, hashType)
		if err != nil {
			log.Error("txSignature:push hashType to hashBuffer failed.")
		}
		result = util.DoubleSha256Hash(hashBuffer.Bytes())
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

	err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, uint32(transaction.GetVersion()))
	if err != nil {
		log.Error("txSignature:push transaction.GetVersion() to hashBuffer failed.")
	}
	var inputsCount int
	if sigHashAnyOneCanPay {
		inputsCount = 1
	} else {
		inputsCount = insLen
	}
	err = util.WriteVarInt(&hashBuffer, uint64(inputsCount))
	if err != nil {
		log.Error("txSignature:push inputsCount to hashBuffer failed.")
	}

	ss := s.RemoveOpcode(opcodes.OP_CODESEPARATOR)

	// encode tx.inputs
	var i int
	for i = 0; i < inputsCount; i++ {
		if sigHashAnyOneCanPay {
			ins[nIn].PreviousOutPoint.Encode(&hashBuffer)
			err = ss.Serialize(&hashBuffer)
			if err != nil {
				log.Error("txSignature:serialize hashBuffer failed.")
			}
			err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[nIn].Sequence)
			if err != nil {
				log.Error("txSignature:push ins[nIn].Sequence to hashBuffer failed.")
			}
		} else {
			ins[i].PreviousOutPoint.Encode(&hashBuffer)
			if i != nIn {
				// push empty script
				err = util.WriteVarInt(&hashBuffer, 0)
				if err != nil {
					log.Error("txSignature:push empty script to hashBuffer failed.")
				}
				if sigHashSingle || sigHashNone {
					// push empty sequence
					err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, 0)
					if err != nil {
						log.Error("txSignature:push empty sequence to hashBuffer failed.")
					}
				} else {
					err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[i].Sequence)
					if err != nil {
						log.Error("txSignature:ins[i].Sequence put hashBuffer failed.")
					}
				}
			} else {
				err = ss.Serialize(&hashBuffer)
				if err != nil {
					log.Error("txSignature:hashBuffer serialize failed.")
				}
				err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[i].Sequence)
				if err != nil {
					log.Error("txSignature:ins[i].Sequence put hashBuffer failed.")
				}
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
	err = util.WriteVarInt(&hashBuffer, uint64(outsCount))
	if err != nil {
		log.Error("txSignature:write outsCount failed.")
	}
	for m := 0; m < outsCount; m++ {
		if sigHashSingle && m != nIn {
			to := txout.NewTxOut(-1, nil)
			err = to.Encode(&hashBuffer)
			if err != nil {
				log.Error("txSignature:txOut encode failed.")
			}
		} else {
			err = outs[m].Encode(&hashBuffer)
			if err != nil {
				log.Error("txSignature:tx's out encode failed.")
			}
		}
	}

	// encode tx.locktime
	err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetLockTime())
	if err != nil {
		log.Error("txSignature:tx's lockTime put hashBuffer failed.")
	}
	err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, hashType)
	if err != nil {
		log.Error("txSignature:hashType put hashBuffer failed.")
	}
	//log.Debug("SignatureHash buf: %s", hex.EncodeToString(hashBuffer.Bytes()))
	result = util.DoubleSha256Hash(hashBuffer.Bytes())
	return
}

func GetPreviousOutHash(tx *Tx) (h util.Hash) {
	ins := tx.GetIns()
	var bPreOut bytes.Buffer
	for _, e := range ins {
		err := e.PreviousOutPoint.Encode(&bPreOut)
		if err != nil {
			log.Error("txSignature:previous outPoint encode failed.")
		}
	}
	h = util.DoubleSha256Hash(bPreOut.Bytes())
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
	h = util.DoubleSha256Hash(buf)
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
	h = util.DoubleSha256Hash(bOut.Bytes())
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
	if !sign.EcdsaNormalize() {
		return false
	}
	ret := sign.Verify(signHash.GetCloneBytes(), publicKey)
	return ret
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
