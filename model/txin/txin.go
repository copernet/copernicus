package txin

import (
	"io"

	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"math"
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

	bCoinBase := false
	if txIn.PreviousOutPoint.Index == 0xffffffff || txIn.PreviousOutPoint.Hash == util.HashZero {
		bCoinBase = true
	}
	scriptSig := script.NewEmptyScript()
	err = scriptSig.Decode(reader, bCoinBase)
	if err != nil {
		return err
	}
	txIn.scriptSig = scriptSig
	//log.Debug("txIn's Script is %v", txIn.scriptSig.GetData())
	return util.ReadElements(reader, &txIn.Sequence)
}

func (txIn *TxIn) GetScriptSig() *script.Script {
	return txIn.scriptSig
}

func (txIn *TxIn) SetScriptSig(scriptSig *script.Script) {
	txIn.scriptSig = scriptSig
}

func (txIn *TxIn) CheckStandard() (bool, string) {
	return txIn.scriptSig.CheckScriptSigStandard()
}

func (txIn *TxIn) String() string {
	str := fmt.Sprintf("PreviousOutPoint: %s ", txIn.PreviousOutPoint.String())
	if txIn.scriptSig == nil {
		return fmt.Sprintf("%s , script:  , Sequence:%d ", str, txIn.Sequence)
	}
	return fmt.Sprintf("%s , script:%s , Sequence:%d ", str, hex.EncodeToString(txIn.scriptSig.GetData()), txIn.Sequence)

}

func NewTxIn(previousOutPoint *outpoint.OutPoint, scriptSig *script.Script, sequence uint32) *TxIn {
	txIn := TxIn{PreviousOutPoint: previousOutPoint, scriptSig: scriptSig, Sequence: sequence}
	if txIn.PreviousOutPoint == nil {
		txIn.PreviousOutPoint = outpoint.NewOutPoint(util.Hash{}, math.MaxUint32)
	}
	return &txIn
}
