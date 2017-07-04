package model

type TxIn struct {
	InputHash string
	InputVout uint32
	ScriptSig []byte
	Sequence  uint32
}
