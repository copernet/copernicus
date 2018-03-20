package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

const (
	TxOrphan = iota
	TxInvalid
	CoinAmount = 100000000
)

const (
	// SequenceLockTimeDisableFlag below flags apply in the context of BIP 68*/
	// If this flag set, CTxIn::nSequence is NOT interpreted as a
	// relative lock-time. */
	SequenceLockTimeDisableFlag = 1 << 31

	// SequenceLockTimeTypeFlag if CTxIn::nSequence encodes a relative lock-time and this flag
	// is set, the relative lock-time has units of 512 seconds,
	// otherwise it specifies blocks with a granularity of 1.
	SequenceLockTimeTypeFlag = 1 << 22

	// SequenceLockTimeMask if CTxIn::nSequence encodes a relative lock-time, this mask is
	// applied to extract that lock-time from the sequence field.
	SequenceLockTimeMask = 0x0000ffff

	// SequenceLockTimeQranularity in order to use the same number of bits to encode roughly the
	// same wall-clock duration, and because blocks are naturally
	// limited to occur every 600s on average, the minimum granularity
	// for time-based relative lock-time is fixed at 512 seconds.
	// Converting from CTxIn::nSequence to seconds is performed by
	// multiplying by 512 = 2^9, or equivalently shifting up by
	// 9 bits.
	SequenceLockTimeQranularity = 9

	MaxMoney = 21000000 * CoinAmount

	// MaxTxSigOpsCounts the maximum allowed number of signature check operations per transaction (network rule)
	MaxTxSigOpsCounts = 20000

	MaxStandardVersion = 2

	MaxTxInSequenceNum uint32 = 0xffffffff
	FreeListMaxItems          = 12500
	MaxMessagePayload         = 32 * 1024 * 1024
	MinTxInPayload            = 9 + utils.Hash256Size
	MaxTxInPerMessage         = (MaxMessagePayload / MinTxInPayload) + 1
	TxVersion                 = 1
)

type Tx struct {
	Hash     utils.Hash // Cached transaction hash	todo defined a pointer will be the optimization
	LockTime uint32
	Version  int32
	Ins      []*TxIn
	Outs     []*TxOut
	ValState int
}

var scriptPool ScriptFreeList = make(chan []byte, FreeListMaxItems)

func (tx *Tx) AddTxIn(txIn *TxIn) {
	tx.Ins = append(tx.Ins, txIn)
}

func (tx *Tx) AddTxOut(txOut *TxOut) {
	tx.Outs = append(tx.Outs, txOut)
}

func (tx *Tx) RemoveTxIn(txIn *TxIn) {
	ret := tx.Ins[:0]
	for _, e := range tx.Ins {
		if e != txIn {
			ret = append(ret, e)
		}
	}
	tx.Ins = ret
}

func (tx *Tx) RemoveTxOut(txOut *TxOut) {
	ret := tx.Outs[:0]
	for _, e := range tx.Outs {
		if e != txOut {
			ret = append(ret, e)
		}
	}
	tx.Outs = ret
}

func (tx *Tx) SerializeSize() int {
	// Version 4 bytes + LockTime 4 bytes + Serialized varint size for the
	// number of transaction inputs and outputs.
	n := 8 + utils.VarIntSerializeSize(uint64(len(tx.Ins))) + utils.VarIntSerializeSize(uint64(len(tx.Outs)))
	if tx == nil {
		fmt.Println("tx is nil")
	}
	for _, txIn := range tx.Ins {
		if txIn == nil {
			fmt.Println("txIn ins is nil")
		}
		n += txIn.SerializeSize()
	}
	for _, txOut := range tx.Outs {
		n += txOut.SerializeSize()
	}
	return n
}

func (tx *Tx) Serialize(writer io.Writer) error {
	err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, uint32(tx.Version))
	if err != nil {
		return err
	}
	count := uint64(len(tx.Ins))
	err = utils.WriteVarInt(writer, count)
	if err != nil {
		return err
	}
	for _, txIn := range tx.Ins {
		err := txIn.Serialize(writer, tx.Version)
		if err != nil {
			return err
		}
	}
	count = uint64(len(tx.Outs))
	err = utils.WriteVarInt(writer, count)
	if err != nil {
		return err
	}
	for _, txOut := range tx.Outs {
		err := txOut.Serialize(writer)
		if err != nil {
			return err
		}
	}
	return utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, tx.LockTime)

}

func DeserializeTx(reader io.Reader) (tx *Tx, err error) {
	tx = new(Tx)
	version, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return tx, err
	}
	tx.Version = int32(version)
	count, err := utils.ReadVarInt(reader)
	if err != nil {
		return tx, err
	}
	if count > uint64(MaxTxInPerMessage) {
		err = errors.Errorf("too many input tx to fit into max message size [count %d , max %d]", count, MaxTxInPerMessage)
		return
	}

	txIns := make([]TxIn, count)
	tx.Ins = make([]*TxIn, count)
	for i := uint64(0); i < count; i++ {
		txIns[i].PreviousOutPoint = new(OutPoint)
		txIns[i].PreviousOutPoint.Hash = *new(utils.Hash)

		txIn := &txIns[i]
		tx.Ins[i] = txIn
		err = txIn.Deserialize(reader, tx.Version)
		if err != nil {
			tx.returnScriptBuffers()
			return
		}
	}
	count, err = utils.ReadVarInt(reader)
	if err != nil {
		tx.returnScriptBuffers()
		return
	}

	txOuts := make([]TxOut, count)
	tx.Outs = make([]*TxOut, count)
	for i := uint64(0); i < count; i++ {
		// The pointer is set now in case a script buffer is borrowed
		// and needs to be returned to the pool on error.

		txOut := &txOuts[i]
		tx.Outs[i] = txOut
		err = txOut.Deserialize(reader)
		if err != nil {
			tx.returnScriptBuffers()
			return
		}
	}

	tx.LockTime, err = utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		tx.returnScriptBuffers()
		return
	}
	tx.returnScriptBuffers()
	return

}

func (tx *Tx) IsCoinBase() bool {
	return len(tx.Ins) == 1 && tx.Ins[0].PreviousOutPoint == nil
}

func (tx *Tx) GetSigOpCountWithoutP2SH() int {
	n := 0
	for _, in := range tx.Ins {
		if c, err := in.Script.GetSigOpCountWithAccurate(false); err == nil {
			n += c
		}
	}
	for _, out := range tx.Outs {
		if c, err := out.Script.GetSigOpCountWithAccurate(false); err == nil {
			n += c
		}
	}
	return n
}

func (tx *Tx) CheckCoinbase(state *ValidationState, checkDupInput bool) bool {
	if !tx.IsCoinBase() {
		return state.Dos(100, false, RejectInvalid, "bad-cb-missing", false,
			"first tx is not coinbase")
	}
	if !tx.CheckTransactionCommon(state, checkDupInput) {
		return false
	}
	if tx.Ins[0].Script.Size() < 2 || tx.Ins[0].Script.Size() > 100 {
		return state.Dos(100, false, RejectInvalid, "bad-cb-length", false, "")
	}
	return true
}

func (tx *Tx) CheckRegularTransaction(state *ValidationState, checkDupInput bool) bool {
	if tx.IsCoinBase() {
		return state.Dos(100, false, RejectInvalid, "bad-tx-coinbase", false, "")
	}
	if !tx.CheckTransactionCommon(state, checkDupInput) {
		return false
	}
	for _, in := range tx.Ins {
		if in.PreviousOutPoint.IsNull() {
			return state.Dos(10, false, RejectInvalid, "bad-txns-prevout-null", false, "")
		}
	}
	return true
}

func (tx *Tx) CheckTransactionCommon(state *ValidationState, checkDupInput bool) bool {
	if len(tx.Ins) == 0 {
		return state.Dos(10, false, RejectInvalid, "bad-txns-vin-empty", false, "")
	}
	if len(tx.Outs) == 0 {
		return state.Dos(10, false, RejectInvalid, "bad-txns-vout-empty", false, "")
	}
	if tx.SerializeSize() > MaxTxSize {
		return state.Dos(100, false, RejectInvalid, "bad-txns-oversize", false, "")
	}
	totalOut := int64(0)
	for _, out := range tx.Outs {
		if out.Value < 0 {
			return state.Dos(100, false, RejectInvalid, "bad-txns-vout-negative", false, "")
		}
		if out.Value > MaxMoney {
			return state.Dos(100, false, RejectInvalid, "bad-txns-vout-toolarge", false, "")
		}
		totalOut += out.Value
		if totalOut < 0 || totalOut > MaxMoney {
			return state.Dos(100, false, RejectInvalid, "bad-txns-txouttotal-toolarge", false, "")
		}
	}
	if tx.GetSigOpCountWithoutP2SH() > 100 {
		return state.Dos(100, false, RejectInvalid, "bad-txn-sigops", false, "")
	}
	if checkDupInput {
		outPointSet := make(map[*OutPoint]struct{})
		for _, in := range tx.Ins {
			if _, ok := outPointSet[in.PreviousOutPoint]; !ok {
				outPointSet[in.PreviousOutPoint] = struct{}{}
			} else {
				return state.Dos(100, false, RejectInvalid, "bad-txns-inputs-duplicate", false, "")
			}
		}
	}
	return true
}

func (tx *Tx) CheckSelf() (bool, error) {
	if tx.Version > MaxStandardVersion || tx.Version < 1 {
		return false, errors.New("error version")
	}
	if len(tx.Ins) == 0 || len(tx.Outs) == 0 {
		return false, errors.New("no inputs or outputs")
	}
	size := tx.SerializeSize()
	if size > MaxTxSize {
		return false, errors.Errorf("tx size %d > max size %d", size, MaxTxSize)
	}

	TotalOutValue := int64(0)
	TotalSigOpCount := int64(0)
	TxOutsLen := len(tx.Outs)
	//to do: check txOut's script is
	for i := 0; i < TxOutsLen; i++ {
		txOut := tx.Outs[i]
		if txOut.Value < 0 {
			return false, errors.Errorf("tx out %d's value:%d invalid", i, txOut.Value)
		}
		if txOut.Value > MaxMoney {
			return false, errors.Errorf("tx out %d's value:%d invalid", i, txOut.Value)
		}

		TotalOutValue += txOut.Value
		if TotalOutValue > MaxMoney {
			return false, errors.Errorf("tx outs' total value:%d from 0 to %d is too large", TotalOutValue, i)
		}

		TotalSigOpCount += int64(txOut.SigOpCount)
		if TotalSigOpCount > int64(MaxTxSigOpsCounts) {
			return false, errors.Errorf("tx outs' total SigOpCount:%d from 0 to %d is too large", TotalSigOpCount, i)
		}
	}

	//todo: check ins' preout duplicate at the same time
	TxInsLen := len(tx.Ins)
	for i := 0; i < TxInsLen; i++ {
		txIn := tx.Ins[i]
		TotalSigOpCount += int64(txIn.SigOpCount)
		if TotalSigOpCount > int64(MaxTxSigOpsCounts) {
			return false, errors.Errorf("tx total SigOpCount:%d of all Outs and partial Ins from 0 to %d is too large", TotalSigOpCount, i)
		}
		if txIn.Script.Size() > 1650 {
			return false, errors.Errorf("txIn %d has too long script", i)
		}
		if !txIn.Script.IsPushOnly() {
			return false, errors.Errorf("txIn %d's script is not push script", i)
		}
	}
	return true, nil
}

func (tx *Tx) returnScriptBuffers() {
	for _, txIn := range tx.Ins {
		if txIn == nil || txIn.Script == nil {
			continue
		}
		scriptPool.Return(txIn.Script.bytes)
	}
	for _, txOut := range tx.Outs {
		if txOut == nil || txOut.Script == nil {
			continue
		}
		scriptPool.Return(txOut.Script.bytes)
	}
}
func (tx *Tx) GetValueOut() int64 {
	var valueOut int64
	for _, out := range tx.Outs {
		valueOut += out.Value
		if !utils.MoneyRange(out.Value) || !utils.MoneyRange(valueOut) {
			panic("value out of range")
		}
	}
	return valueOut
}

func (tx *Tx) Copy() *Tx {
	newTx := Tx{
		Version:  tx.Version,
		LockTime: tx.LockTime,
		Ins:      make([]*TxIn, 0, len(tx.Ins)),
		Outs:     make([]*TxOut, 0, len(tx.Outs)),
	}
	newTx.Hash = tx.Hash

	for _, txOut := range tx.Outs {
		scriptLen := len(txOut.Script.bytes)
		newOutScript := make([]byte, scriptLen)
		copy(newOutScript, txOut.Script.bytes[:scriptLen])

		newTxOut := TxOut{
			Value:  txOut.Value,
			Script: NewScriptRaw(newOutScript),
		}
		newTx.Outs = append(newTx.Outs, &newTxOut)
	}
	for _, txIn := range tx.Ins {
		var hashBytes [32]byte
		copy(hashBytes[:], txIn.PreviousOutPoint.Hash[:])
		preHash := new(utils.Hash)
		preHash.SetBytes(hashBytes[:])
		newOutPoint := OutPoint{Hash: *preHash, Index: txIn.PreviousOutPoint.Index}
		scriptLen := txIn.Script.Size()
		newScript := make([]byte, scriptLen)
		copy(newScript[:], txIn.Script.bytes[:scriptLen])
		newTxTmp := TxIn{
			Sequence:         txIn.Sequence,
			PreviousOutPoint: &newOutPoint,
			Script:           NewScriptRaw(newScript),
		}
		newTx.Ins = append(newTx.Ins, &newTxTmp)
	}
	return &newTx

}

func (tx *Tx) Equal(dstTx *Tx) bool {
	originBuf := bytes.NewBuffer(nil)
	tx.Serialize(originBuf)

	dstBuf := bytes.NewBuffer(nil)
	dstTx.Serialize(dstBuf)

	return bytes.Equal(originBuf.Bytes(), dstBuf.Bytes())
}

func (tx *Tx) ComputePriority(priorityInputs float64, txSize int) float64 {
	txModifiedSize := tx.CalculateModifiedSize()
	if txModifiedSize == 0 {
		return 0
	}
	return priorityInputs / float64(txModifiedSize)

}

func (tx *Tx) CalculateModifiedSize() int {
	// In order to avoid disincentivizing cleaning up the UTXO set we don't
	// count the constant overhead for each txin and up to 110 bytes of
	// scriptSig (which is enough to cover a compressed pubkey p2sh redemption)
	// for priority. Providing any more cleanup incentive than making additional
	// inputs free would risk encouraging people to create junk outputs to
	// redeem later.
	txSize := tx.SerializeSize()
	for _, in := range tx.Ins {
		inScriptModifiedSize := math.Min(110, float64(len(in.Script.bytes)))
		offset := 41 + int(inScriptModifiedSize)
		if txSize > offset {
			txSize -= offset
		}
	}
	return txSize

}

func (tx *Tx) IsFinalTx(Height int, time int64) bool {
	if tx.LockTime == 0 {
		return true
	}

	lockTimeLimit := int64(0)
	if tx.LockTime < LockTimeThreshold {
		lockTimeLimit = int64(Height)
	} else {
		lockTimeLimit = time
	}

	if int64(tx.LockTime) < lockTimeLimit {
		return true
	}

	for _, txin := range tx.Ins {
		if txin.Sequence != SequenceFinal {
			return false
		}
	}

	return true
}

func (tx *Tx) String() string {
	str := ""
	str = fmt.Sprintf(" hash :%s version : %d  lockTime: %d , ins:%d outs:%d \n", tx.Hash.ToString(), tx.Version, tx.LockTime, len(tx.Ins), len(tx.Outs))
	inStr := "ins:\n"
	for i, in := range tx.Ins {
		if in == nil {
			inStr = fmt.Sprintf("  %s %d , nil \n", inStr, i)
		} else {
			inStr = fmt.Sprintf("  %s %d , %s\n", inStr, i, in.String())
		}
	}
	outStr := "outs:\n"
	for i, out := range tx.Outs {
		outStr = fmt.Sprintf("  %s %d , %s\n", outStr, i, out.String())
	}
	return fmt.Sprintf("%s%s%s", str, inStr, outStr)
}

func (tx *Tx) TxHash() utils.Hash {
	// cache hash
	if !tx.Hash.IsNull() {
		return tx.Hash
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	_ = tx.Serialize(buf)
	hash := crypto.DoubleSha256Hash(buf.Bytes())
	tx.Hash = hash
	return hash
}

func NewTx() *Tx {
	return &Tx{LockTime: 0, Version: TxVersion}
}

// PrecomputedTransactionData Precompute sighash midstate to avoid quadratic hashing
type PrecomputedTransactionData struct {
	HashPrevout  *utils.Hash
	HashSequence *utils.Hash
	HashOutputs  *utils.Hash
}

func NewPrecomputedTransactionData(tx *Tx) *PrecomputedTransactionData {
	hashPrevout, _ := GetPrevoutHash(tx)
	hashSequence, _ := GetSequenceHash(tx)
	hashOutputs, _ := GetOutputsHash(tx)

	return &PrecomputedTransactionData{
		HashPrevout:  &hashPrevout,
		HashSequence: &hashSequence,
		HashOutputs:  &hashOutputs,
	}
}
