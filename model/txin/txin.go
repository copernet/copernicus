package txin

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/util"
	"io"
)

type TxIn struct {
	PreviousOutPoint *outpoint.OutPoint
	scriptSig        *script.Script
	Sequence         uint32
}

func (txIn *TxIn) SerializeSize() uint32 {
	return txIn.EncodeSize()
}

func (txIn *TxIn) Unserialize(reader io.Reader) error {
	return txIn.Decode(reader)
}

func (txIn *TxIn) Serialize(writer io.Writer) error {
	return txIn.Encode(writer)
}

func (txIn *TxIn) EncodeSize() uint32 {
	// previousOutPoint EncodeSize + scriptSig EncodeSize + Sequence 4 bytes
	return txIn.PreviousOutPoint.EncodeSize() + txIn.scriptSig.EncodeSize() + 4
}

func (txIn *TxIn) Encode(writer io.Writer) error {
	err := txIn.PreviousOutPoint.Encode(writer)
	if err != nil {
		return err
	}
	err = txIn.scriptSig.Encode(writer)
	if err != nil {
		return err
	}

	err = util.BinarySerializer.PutUint32(writer, binary.LittleEndian, txIn.Sequence)
	return err

}

func (txIn *TxIn) Decode(reader io.Reader) error {
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

func (txIn *TxIn) SetScript(script *script.Script) {
	txIn.scriptSig = script
}

func NewTxIn(previousOutPoint *outpoint.OutPoint, scriptSig *script.Script, sequence uint32) *TxIn {
	txIn := TxIn{PreviousOutPoint: previousOutPoint, scriptSig: scriptSig, Sequence: sequence}
	return &txIn
}
