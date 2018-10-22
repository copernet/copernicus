package tx

import (
	"bytes"
	"encoding/binary"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

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
			log.Error("txSignature:push transaction.GetVersion() to hashBuffer failed: %v", err)
			return util.HashOne, err
		}
		_, err = hashBuffer.Write(hashPrevouts[:])
		if err != nil {
			log.Error("txSignature:write hashPrevouts failed: %v", err)
			return util.HashOne, err
		}
		_, err = hashBuffer.Write(hashSequence[:])
		if err != nil {
			log.Error("txSignature:write hashSequence failed: %v", err)
			return util.HashOne, err
		}
		err = transaction.GetIns()[nIn].PreviousOutPoint.Encode(&hashBuffer)
		if err != nil {
			log.Error("txSignature:Previous OutPoint encode failed: %v", err)
			return util.HashOne, err
		}
		err = s.Serialize(&hashBuffer)
		if err != nil {
			log.Error("txSignature:serialize hashBuffer failed: %v", err)
			return util.HashOne, err
		}
		//input preout amount
		err = util.BinarySerializer.PutUint64(&hashBuffer, binary.LittleEndian, uint64(money))
		if err != nil {
			log.Error("txSignature:push money to hashBuffer failed: %v", err)
			return util.HashOne, err
		}
		err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetIns()[nIn].Sequence)
		if err != nil {
			log.Error("txSignature:push transaction.GetIns()[nIn].Sequence to hashBuffer failed: %v", err)
			return util.HashOne, err
		}
		_, err = hashBuffer.Write(hashOutputs[:])
		if err != nil {
			log.Error("txSignature:write hashOutputs failed: %v", err)
			return util.HashOne, err
		}
		err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetLockTime())
		if err != nil {
			log.Error("txSignature:push transaction.GetLockTime() to hashBuffer failed: %v", err)
			return util.HashOne, err
		}
		err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, hashType)
		if err != nil {
			log.Error("txSignature:push hashType to hashBuffer failed: %v", err)
			return util.HashOne, err
		}
		result = util.DoubleSha256Hash(hashBuffer.Bytes())
		return result, nil
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
		log.Error("txSignature:push transaction.GetVersion() to hashBuffer failed: %v", err)
		return util.HashOne, err
	}
	var inputsCount int
	if sigHashAnyOneCanPay {
		inputsCount = 1
	} else {
		inputsCount = insLen
	}
	err = util.WriteVarInt(&hashBuffer, uint64(inputsCount))
	if err != nil {
		log.Error("txSignature:push inputsCount to hashBuffer failed: %v", err)
		return util.HashOne, err
	}

	ss := s.RemoveOpcode(opcodes.OP_CODESEPARATOR)

	// encode tx.inputs
	var i int
	for i = 0; i < inputsCount; i++ {
		if sigHashAnyOneCanPay {
			ins[nIn].PreviousOutPoint.Encode(&hashBuffer)
			err = ss.Serialize(&hashBuffer)
			if err != nil {
				log.Error("txSignature:serialize hashBuffer failed: %v", err)
				return util.HashOne, err
			}
			err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[nIn].Sequence)
			if err != nil {
				log.Error("txSignature:push ins[nIn].Sequence to hashBuffer failed: %v", err)
				return util.HashOne, err
			}
		} else {
			ins[i].PreviousOutPoint.Encode(&hashBuffer)
			if i != nIn {
				// push empty script
				err = util.WriteVarInt(&hashBuffer, 0)
				if err != nil {
					log.Error("txSignature:push empty script to hashBuffer failed: %v", err)
					return util.HashOne, err
				}
				if sigHashSingle || sigHashNone {
					// push empty sequence
					err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, 0)
					if err != nil {
						log.Error("txSignature:push empty sequence to hashBuffer failed: %v", err)
						return util.HashOne, err
					}
				} else {
					err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[i].Sequence)
					if err != nil {
						log.Error("txSignature:ins[i].Sequence put hashBuffer failed: %v", err)
						return util.HashOne, err
					}
				}
			} else {
				err = ss.Serialize(&hashBuffer)
				if err != nil {
					log.Error("txSignature:hashBuffer serialize failed: %v", err)
					return util.HashOne, err
				}
				err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, ins[i].Sequence)
				if err != nil {
					log.Error("txSignature:ins[i].Sequence put hashBuffer failed: %v", err)
					return util.HashOne, err
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
		log.Error("txSignature:write outsCount failed: %v", err)
		return util.HashOne, err
	}
	for m := 0; m < outsCount; m++ {
		if sigHashSingle && m != nIn {
			to := txout.NewTxOut(-1, nil)
			err = to.Encode(&hashBuffer)
			if err != nil {
				log.Error("txSignature:txOut encode failed: %v", err)
				return util.HashOne, err
			}
		} else {
			err = outs[m].Encode(&hashBuffer)
			if err != nil {
				log.Error("txSignature:tx's out encode failed: %v", err)
				return util.HashOne, err
			}
		}
	}

	// encode tx.locktime
	err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, transaction.GetLockTime())
	if err != nil {
		log.Error("txSignature:tx's lockTime put hashBuffer failed: %v", err)
		return util.HashOne, err
	}
	err = util.BinarySerializer.PutUint32(&hashBuffer, binary.LittleEndian, hashType)
	if err != nil {
		log.Error("txSignature:hashType put hashBuffer failed: %v", err)
		return util.HashOne, err
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
			log.Error("txSignature:previous outPoint encode failed: %v", err)
			return util.HashOne
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
