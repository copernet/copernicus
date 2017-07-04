package model

type Tx struct {
	Hash     string
	Size     uint32
	LockTime uint32
	Version  uint32
	TxInCnt  uint32
	TxOutCnt uint32
	Ins      []*TxIn
	Outs     []*TxOut
}

const (
	MaxTxInSequenceNum uint32 = 0xffffffff
)

//
//func ParseTranscations(raws [] byte) (txs []*Tx, err error) {
//	offset := int(0)
//	txCnt, txCntSize := utils.DecodeVariableLengthInteger(raws[offset:])
//	offset += txCntSize
//
//	txs = make([]*Tx, txCnt)
//
//	txOffset := int(0)
//	for i := range txs {
//		txs[i], txOffset = ParseTranscation(txs[offset:])
//		txs[i].Hash = utils.ToHash256String(raws[offset:offset + txOffset])
//		txs[i].Size = uint32(txOffset)
//		offset += txOffset
//	}
//	return
//
//}
//
//func ParseTranscation(raw [] byte) (txs *[]Tx, offset int) {
//	tx = new(Tx)
//	tx.Version = binary.LittleEndian.Uint32(raw[0:4])
//	offset = 4
//
//	inCnt, inCntSize := utils.DecodeVariableLengthInteger(raw[offset:])
//	offset += inCntSize
//
//	tx.TxInCnt = uint32((inCnt))
//	tx.Ins = make([]*TxIn, inCnt)
//
//	txInOffset := int(0)
//
//	for i := range tx.Ins {
//		tx.Ins[i], txInOffset = ParseTranscationIn(raw[offset:])
//		offset += txInOffset
//	}
//
//	txOutCnt, txOutCntSize := utils.DecodeVariableLengthInteger(raw[offset:])
//	offset += txOutCntSize
//
//	tx.TxOutCnt = uint32(txOutCnt)
//	tx.Outs = make([]*TxOut, txOutCnt)
//
//	txOutOffset := int(0)
//	for i := range tx.Outs {
//		tx.Outs[i], txOutOffset = ParseTranscationOut(raw[offset:])
//		offset += txOutOffset
//	}
//	tx.LockTime = binary.LittleEndian.Uint32(raw[offset:offset + 4])
//	offset += 4
//	return
//}
//
//func ParseTranscationIn(raw[] byte) (txIn*TxIn, offset int) {
//	txIn = new(TxIn)
//	txIn.InputHash = utils.ToHash256String(raw[0:32])
//	txIn.InputVout = binary.LittleEndian.Uint32(raw[32:36])
//	offset = 36
//
//	scriptSigCnt, scriptSigSzie := utils.DecodeVariableLengthInteger(raw[offset:])
//	offset += scriptSigSzie
//	txIn.ScriptSig = raw[offset:offset + scriptSigCnt]
//	offset += scriptSigCnt
//
//	txIn.Sequence = binary.LittleEndian.Uint32(raw[offset:offset + 4])
//	offset + 4
//	return
//}
//func ParseTranscationOut(rawOut []byte) (txOut*TxOut, offset int) {
//
//	txOut = new(TxOut)
//	offset = 8
//	txOut.Value = binary.LittleEndian.Uint64(rawOut[0:offset])
//
//	pkScriptCnt, pkScriptSize := utils.DecodeVariableLengthInteger(rawOut[offset:])
//	offset += pkScriptSize
//	txOut.OutScript = rawOut[offset:offset + pkScriptCnt]
//	offset += pkScriptCnt
//
//	_, addressHash, _, err := txscript.ExtractPkScriptAddrs(txOut.OutScript, &chaincfg.MainNetParams)
//
//	if err != nil {
//		return
//
//	}
//	if len(addressHash) != 0 {
//		txOut.Address = addressHash[0].EncodeAddress()
//
//	} else {
//		txOut.Address = ""
//	}
//	return
//
//}
