package model

type TxOut struct {
	Address  string
	Value    uint64
	PKScript []byte
}
