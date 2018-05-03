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
	"copernicus/conf"
	utils2 "copernicus/utils"
	"copernicus/blockchain"
	"btcd/chaincfg"
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
	ins      []*TxIn
	outs     []*TxOut
	//ValState int
}

var scriptPool ScriptFreeList = make(chan []byte, FreeListMaxItems)


func (tx *Tx) AddTxIn(txIn *TxIn) {
	tx.ins = append(tx.ins, txIn)
}

func (tx *Tx) AddTxOut(txOut *TxOut) {
	tx.outs = append(tx.outs, txOut)
}

func (tx *Tx) RemoveTxIn(txIn *TxIn) {
	ret := tx.ins[:0]
	for _, e := range tx.ins {
		if e != txIn {
			ret = append(ret, e)
		}
	}
	tx.ins = ret
}

func (tx *Tx) RemoveTxOut(txOut *TxOut) {
	ret := tx.outs[:0]
	for _, e := range tx.outs {
		if e != txOut {
			ret = append(ret, e)
		}
	}
	tx.outs = ret
}

func (tx *Tx) SerializeSize() uint {
	// Version 4 bytes + LockTime 4 bytes + Serialized varint size for the
	// number of transaction inputs and outputs.
	n := 8 + utils.VarIntSerializeSize(uint64(len(tx.Ins))) + utils.VarIntSerializeSize(uint64(len(tx.outs)))
	//if tx == nil {
	//	fmt.Println("tx is nil")
	//}
	for _, txIn := range tx.ins {
		if txIn == nil {
			fmt.Println("txIn ins is nil")
		}
		n += txIn.SerializeSize()
	}
	for _, txOut := range tx.outs {
		n += txOut.SerializeSize()
	}
	return n
}

func (tx *Tx) Serialize(writer io.Writer) error {
	err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, tx.Version)
	if err != nil {
		return err
	}
	count := uint64(len(tx.Ins))
	err = utils.WriteVarInt(writer, count)
	if err != nil {
		return err
	}
	for _, txIn := range tx.Ins {
		err := txIn.Serialize(writer)
		if err != nil {
			return err
		}
	}
	count = uint64(len(tx.outs))
	err = utils.WriteVarInt(writer, count)
	if err != nil {
		return err
	}
	for _, txOut := range tx.outs {
		err := txOut.Serialize(writer)
		if err != nil {
			return err
		}
	}
	return utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, tx.LockTime)

}

func (tx *Tx)Deserialize(reader io.Reader) error {
	version, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return err
	}
	count, err := utils.ReadVarInt(reader)
	if err != nil {
		return err
	}
	if count > uint64(MaxTxInPerMessage) {
		err = errors.Errorf("too many input tx to fit into max message size [count %d , max %d]", count, MaxTxInPerMessage)
		return err
	}

	tx.Version = int32(version)
	tx.ins = make([]*TxIn, count)

	for i := uint64(0); i < count; i++ {
		txIn := new(TxIn)
		txIn.PreviousOutPoint = new(OutPoint)
		txIn.PreviousOutPoint.Hash = *new(utils.Hash)
		err = txIn.Deserialize(reader)
		if err != nil {
			return err
		}
		tx.ins[i] = txIn
	}
	count, err = utils.ReadVarInt(reader)
	if err != nil {
		return err
	}

	tx.outs = make([]*TxOut, count)
	for i := uint64(0); i < count; i++ {
		// The pointer is set now in case a script buffer is borrowed
		// and needs to be returned to the pool on error.
		txOut := new(TxOut)
		err = txOut.Deserialize(reader)
		if err != nil {
			return err
		}
		tx.outs[i] = txOut
	}

	tx.LockTime, err = utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return err
	}
	return err
}

func (tx *Tx) IsCoinBase() bool {
	return len(tx.ins) == 1 && tx.ins[0].PreviousOutPoint == nil
}

func (tx *Tx) GetSigOpCountWithoutP2SH() int {
	n := 0
	for _, in := range tx.ins {
		if c, err := in.Script.GetSigOpCount(false); err == nil {
			n += c
		}
	}
	for _, out := range tx.outs {
		if c, err := out.Script.GetSigOpCount(false); err == nil {
			n += c
		}
	}
	return n
}

// starting BIP16(Apr 1 2012), we should check p2sh
func (tx *Tx) GetSigOpCountWithP2SH() (int, error) {
	n := tx.GetSigOpCountWithoutP2SH()
	if tx.IsCoinBase() {
		return n, nil
	}
	for _, e := range tx.ins {
		coin := utxo.GetCoins(e.PreviousOutPoint)
		if !coin {
			coin = mempool.GetCoins(e.PreviousOutPoint)
			if !coin {
				err := errors.New("TX has no Previous coin")
				return 0, err
			}
		}
		if !coin.Vout.ScriptPubkey.IsPayToScriptHash() {
			n += coin.Vout.ScriptPubkey.GetSigOpCount(true)
		} else {
			n += e.scriptSigcript.GetP2SHSigOpCount()
		}
	}
	return n, nil
}

func (tx *Tx) CheckCoinbaseTransaction(state *ValidationState) bool {
	if !tx.IsCoinBase() {
		return state.Dos(100, false, RejectInvalid, "bad-cb-missing", false,
			"first tx is not coinbase")
	}
	if !tx.checkTransactionCommon(state, false) {
		return false
	}
	if tx.ins[0].Script.Size() < 2 || tx.ins[0].Script.Size() > 100 {
		return state.Dos(100, false, RejectInvalid, "bad-cb-length", false, "")
	}
	return true
}

func (tx *Tx) CheckRegularTransaction(state *ValidationState, allowLargeOpReturn bool) bool {
	if tx.IsCoinBase() {
		state.Dos(100, false, RejectInvalid, "bad-tx-coinbase", false, "")
		return false
	}

	if !tx.checkTransactionCommon(state, true) {
		return false
	}

	// all inputs should have preout
	for _, in := range tx.ins {
		if in.PreviousOutPoint.IsNull() {
			state.Dos(10, false, RejectInvalid, "bad-txns-prevout-null", false, "")
			return false
		}
	}

	// check standard
	if RequireStandard && !tx.checkStandard(state, allowLargeOpReturn) {
		return false
	}

	//check locktime
	if !tx.ContextualCheckTransaction(state, StandardLockTimeVerifyFlags) {
		return false
	}

	// check duplicate tx
	if tx.isOutputAlreadyExist() {
		return state.Dos(10, false, RejectInvalid, "bad-txns-output-already-exist", false, "")
	}

	// check duble-spending
	if !tx.isInputAvailable() {
		return state.Dos(10, false, RejectInvalid, "bad-txns-input-already-spended", false, "")
	}

	//check inputs
	if !tx.checkInputs() {
		return false
	}

	return true
}

func (tx *Tx) checkTransactionCommon(state *ValidationState, checkDupInput bool) bool {
	//check inputs and outputs
	if len(tx.ins) == 0 {
		state.Dos(10, false, RejectInvalid, "bad-txns-vin-empty", false, "")
		return false
	}
	if len(tx.outs) == 0 {
		state.Dos(10, false, RejectInvalid, "bad-txns-vout-empty", false, "")
		return false
	}

	// check size
	if tx.SerializeSize() > MaxTxSize {
		state.Dos(100, false, RejectInvalid, "bad-txns-oversize", false, "")
		return false
	}

	// check outputs money
	totalOut := int64(0)
	for _, out := range tx.outs {
		if !out.CheckValue(state) {
			return false
		}
		totalOut += out.GetValue()
		if totalOut < 0 || totalOut > MaxMoney {
			state.Dos(100, false, RejectInvalid, "bad-txns-txouttotal-toolarge", false, "")
			return false
		}
	}

	// check sigopcount
	if tx.GetSigOpCountWithoutP2SH() > MaxTxSigOpsCount {
		return state.Dos(100, false, RejectInvalid, "bad-txn-sigops", false, "")
	}

	// check dup input
	if checkDupInput {
		outPointSet := make(map[*OutPoint]struct{})
		for _, in := range tx.ins {
			if _, ok := outPointSet[in.PreviousOutPoint]; !ok {
				outPointSet[in.PreviousOutPoint] = struct{}{}
			} else {
				return state.Dos(100, false, RejectInvalid, "bad-txns-inputs-duplicate", false, "")
			}
		}
	}

	return true
}

func (tx *Tx) checkStandard(state *ValidationState, allowLargeOpReturn bool) bool {
	// check version
	if tx.Version > MaxStandardVersion || tx.Version < 1 {
		state.Dos(10, false, RejectInvalid, "bad-tx-version", false, "")
		return false
	}

	// check size
	if tx.SerializeSize() > MaxStandardTxSize {
		state.Dos(100, false, RejectInvalid, "bad-txns-oversize", false, "")
		return false
	}

	// check inputs script
	for _, in := range tx.ins {
		if in.CheckScript(state) {
			return false
		}
	}

	// check output scriptpubkey and inputs scriptsig
	nDataOut := 0
	for _, out := range tx.outs {
		succeed, pubKeyType := out.CheckScript(state, allowLargeOpReturn)
		if !succeed {
			state.Dos(100, false, RejectInvalid, "scriptpubkey", false, "")
			return false
		}
		if pubKeyType == SCRIPT_MULTISIG && !IsBareMultiSigStd {
			state.Dos(100, false, RejectInvalid, "bare-multisig", false, "")
			return false
		}
		if pubKeyType == SCRIPT_NULL_DATA {
			nDataOut++
		}
		// only one OP_RETURN txout is permitted
		if nDataOut > 1 {
			state.Dos(100, false, RejectInvalid, "multi-op-return", false, "")
			return false
		}
		if out.IsDust(conf.GlobalValueInstance.GetDustRelayFee()) {
			state.Dos(100, false, RejectInvalid, "dust out", false, "")
			return false
		}
	}

	return true
}

func (tx *Tx) ContextualCheckTransaction(state *ValidationState, flag int) {
	flags := 0
	if flag > 0 {
		flags = flag
	}

	nBlockHeight := ActiveChain.Height() + 1

	var nLockTimeCutoff int64 = 0

	if flags & LocktimeMedianTimePast {
		nLockTimeCutoff = ActiveChain.Tip()->GetMedianTimePast()
	} else {
		nLockTimeCutoff = utils2.GetAdjustedTime()
	}

	if !tx.isFinal(nBlockHeight, nLockTimeCutoff) {
		return state.Dos(100, false, RejectInvalid, "bad-txns-nonfinal", false, "")
	}

	if blockchain.IsUAHFEnabled(nBlockHeight) && nBlockHeight <= consensusParams.antiReplayOpReturnSunsetHeight {
		for _, e := range tx.outs {
			if e.scriptPubKey.IsCommitment(consensusParams.antiReplayOpReturnCommitment) {
				return state.Dos(100, false, RejectInvalid, "bad-txns-replay", false, "")
			}
		}
	}

	return true
}

func (tx *Tx) isOutputAlreadyExist() bool {
	for i, e := range tx.outs {
		outPoint := NewOutPoint(tx.GetID(), i)
		if GMempool.GetCoin(outPoint) {
			return false
		}
		if GUtxo.GetCoin(outPoint) {
			return false
		}
	}

	return true
}

func (tx *Tx) isInputsAvailable() bool {
	for e := range tx.ins {
		outPoint := e.PreviousOutPoint
		if !GMempool.GetCoin(outPoint) {
			return false
		}
		if !GUtxo.GetCoin(outPoint) {
			return false
		}
	}

	return true
}

func (tx *Tx) checkInputs() bool {

	return true
}

func (tx *Tx) returnScriptBuffers() {
	for _, txIn := range tx.ins {
		if txIn == nil || txIn.scriptSig == nil {
			continue
		}
		scriptPool.Return(txIn.scriptSig.bytes)
	}
	for _, txOut := range tx.outs {
		if txOut == nil || txOut.scriptPubKey == nil {
			continue
		}
		scriptPool.Return(txOut.scriptPubKey.bytes)
	}
}

func (tx *Tx) GetValueOut() int64 {
	var valueOut int64
	for _, out := range tx.outs {
		valueOut += out.Value
		if !utils.MoneyRange(out.Value) || !utils.MoneyRange(valueOut) {
			panic("value out of range")
		}
	}
	return valueOut
}

/*
func (tx *Tx) Copy() *Tx {
	newTx := Tx{
		Version:  tx.Version,
		LockTime: tx.LockTime,
		ins:      make([]*TxIn, 0, len(tx.ins)),
		outs:     make([]*TxOut, 0, len(tx.outs)),
	}
	newTx.Hash = tx.Hash

	for _, txOut := range tx.outs {
		scriptLen := len(txOut.Script.bytes)
		newOutScript := make([]byte, scriptLen)
		copy(newOutScript, txOut.Script.bytes[:scriptLen])

		newTxOut := TxOut{
			Value:  txOut.Value,
			Script: NewScriptRaw(newOutScript),
		}
		newTx.outs = append(newTx.outs, &newTxOut)
	}
	for _, txIn := range tx.ins {
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
		newTx.ins = append(newTx.ins, &newTxTmp)
	}
	return &newTx

}
*/
/*
func (tx *Tx) Equal(dstTx *Tx) bool {
	originBuf := bytes.NewBuffer(nil)
	tx.Serialize(originBuf)

	dstBuf := bytes.NewBuffer(nil)
	dstTx.Serialize(dstBuf)

	return bytes.Equal(originBuf.Bytes(), dstBuf.Bytes())
}
*/

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
	for _, in := range tx.ins {
		InscriptModifiedSize := math.Min(110, float64(len(in.Script.bytes)))
		offset := 41 + int(InScriptModifiedSize)
		if txSize > offset {
			txSize -= offset
		}
	}
	return txSize

}

func (tx *Tx) isFinal(Height int, time int64) bool {
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

	for _, txin := range tx.ins {
		if txin.Sequence != SequenceFinal {
			return false
		}
	}

	return true
}

func (tx *Tx) String() string {
	str := ""
	str = fmt.Sprintf(" hash :%s version : %d  lockTime: %d , ins:%d outs:%d \n", tx.Hash.ToString(), tx.Version, tx.LockTime, len(tx.ins), len(tx.outs))
	inStr := "ins:\n"
	for i, in := range tx.ins {
		if in == nil {
			inStr = fmt.Sprintf("  %s %d , nil \n", inStr, i)
		} else {
			inStr = fmt.Sprintf("  %s %d , %s\n", inStr, i, in.String())
		}
	}
	outStr := "outs:\n"
	for i, out := range tx.outs {
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
/*
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
}*/
