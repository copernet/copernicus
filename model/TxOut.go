package model

import "github.com/btccom/copernicus/utils"

type TxOut struct {
	Address   string
	Value     int64
	OutScript []byte
}

func (txOut *TxOut) SerializeSize() int {
	return 8 + utils.VarIntSerializeSize(uint64(len(txOut.OutScript))) + len(txOut.OutScript)
}
func NewTxOut(value int64, pkScript []byte) *TxOut {
	txOut := TxOut{
		Value:     value,
		OutScript: pkScript,
	}
	return &txOut
}
