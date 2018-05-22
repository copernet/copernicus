package txin

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/util"
)


/*const (
	MaxTxInSequenceNum uint32 = 0xffffffff
)
*/

type TxIn struct {
	PreviousOutPoint *outpoint.OutPoint
	scriptSig        *script.Script
	Sequence         uint32 //todo ?
	SigOpCount       int
}

func (txIn *TxIn) SerializeSize() int {
	// Outpoint Hash 32 bytes + Outpoint Index 4 bytes + Sequence 4 bytes +
	// serialized VarInt size for the length of SignatureScript +
	// SignatureScript bytes.
	if txIn.scriptSig == nil {
		return 40
	}

	return 40 + util.VarIntSerializeSize(uint64(txIn.scriptSig.Size())) + txIn.scriptSig.Size()
}

func (txIn *TxIn) Unserialize(reader io.Reader) error {
	err := txIn.PreviousOutPoint.Decode(reader)
	if err != nil {
		return err
	}
	bytes, err := script.ReadScript(reader, script.MaxMessagePayload, "tx input signature script")
	if err != nil {
		return err
	}
	txIn.scriptSig = script.NewScriptRaw(bytes)
	return util.ReadElements(reader, &txIn.Sequence)
}

func (txIn *TxIn) Serialize(writer io.Writer) error {
	var err error
	if txIn.PreviousOutPoint != nil {
		err = txIn.PreviousOutPoint.Encode(writer)
		if err != nil {
			return err
		}
	}
	err = util.WriteVarBytes(writer, txIn.scriptSig.GetData())
	if err != nil {
		return err
	}

	err = util.BinarySerializer.PutUint32(writer, binary.LittleEndian, txIn.Sequence)
	return err
}

func (txIn *TxIn) GetScriptSig() *script.Script {
	return txIn.scriptSig
}

func (txIn *TxIn) CheckStandard() error {
	return txIn.scriptSig.CheckScriptSigStandard()
}

func (txIn *TxIn) String() string {
	str := fmt.Sprintf("PreviousOutPoint: %s ", txIn.PreviousOutPoint.String())
	if txIn.scriptSig == nil {
		return fmt.Sprintf("%s , script:  , Sequence:%d ", str, txIn.Sequence)
	}
	return fmt.Sprintf("%s , script:%s , Sequence:%d ", str, hex.EncodeToString(txIn.scriptSig.GetData()), txIn.Sequence)

}
/*
func (txIn *TxIn) Check() bool {
	return true
}
*/

func (txIn *TxIn) SetScript(script *script.Script) {
	txIn.scriptSig = script
}

func NewTxIn(previousOutPoint *outpoint.OutPoint, scriptSig *script.Script, sequence uint32) *TxIn {
	txIn := TxIn{PreviousOutPoint: previousOutPoint, scriptSig: scriptSig, Sequence: sequence}
	return &txIn
}
