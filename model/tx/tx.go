package tx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/model/txin"
	"github.com/btcboost/copernicus/model/txout"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/util/amount"
	"github.com/pkg/errors"
	"io"
)

const (
	TxOrphan = iota
	TxInvalid
	CoinAmount = 100000000
)

const (
	RequireStandard = 1
)

const (
	MaxMoney = 21000000 * CoinAmount

	// MaxTxSigOpsCounts the maximum allowed number of signature check operations per transaction (network rule)
	MaxTxSigOpsCounts = 20000

	FreeListMaxItems  = 12500
	MaxMessagePayload = 32 * 1024 * 1024
	MinTxInPayload    = 9 + util.Hash256Size
	MaxTxInPerMessage = (MaxMessagePayload / MinTxInPayload) + 1
	TxVersion         = 1
)

const (
	/*DefaultMaxGeneratedBlockSize default for -blockMaxsize, which controls the maximum size of block the
	 * mining code will create **/
	//DefaultMaxGeneratedBlockSize uint64 = 2 * consensus.OneMegaByte
	/** Default for -blockprioritypercentage, define the amount of block space
	 * reserved to high priority transactions **/

	//DefaultBlockPriorityPercentage uint64= 5

	/*DefaultBlockMinTxFee default for -blockMinTxFee, which sets the minimum feeRate for a transaction
	 * in blocks created by mining code **/
	//DefaultBlockMinTxFee uint = 1000

	MaxStandardVersion = 2

	/*MaxStandardTxSize the maximum size for transactions we're willing to relay/mine */
	MaxStandardTxSize uint = 100000

	/*MaxP2SHSigOps maximum number of signature check operations in an IsStandard() P2SH script*/
	MaxP2SHSigOps uint = 15

	/*MaxStandardTxSigOps the maximum number of sigops we're willing to relay/mine in a single tx */
	MaxStandardTxSigOps = uint(consensus.MaxTxSigOpsCount / 5)

	/*DefaultMaxMemPoolSize default for -maxMemPool, maximum megabytes of memPool memory usage */
	//DefaultMaxMemPoolSize uint = 300

	/** Default for -incrementalrelayfee, which sets the minimum feerate increase
	* for mempool limiting or BIP 125 replacement **/
	DefaultIncrementalRelayFee int64 = 1000

	/** Default for -bytespersigop */
	DefaultBytesPerSigop uint = 20

	/** The maximum number of witness stack items in a standard P2WSH script */
	MaxStandardP2WSHStackItems uint = 100

	/*MaxStandardP2WSHStackItemSize the maximum size of each witness stack item in a standard P2WSH script */
	MaxStandardP2WSHStackItemSize uint = 80

	/*MaxStandardP2WSHScriptSize the maximum size of a standard witnessScript */
	MaxStandardP2WSHScriptSize uint = 3600

	/*StandardLockTimeVerifyFlags used as the flags parameter to sequence and LockTime checks in
	 * non-core code. */
	StandardLockTimeVerifyFlags uint = consensus.LocktimeVerifySequence | consensus.LocktimeMedianTimePast
)

type Tx struct {
	hash     util.Hash // Cached transaction hash	todo defined a pointer will be the optimization
	lockTime uint32
	version  int32
	ins      []*txin.TxIn
	outs     []*txout.TxOut
}

//var scriptPool ScriptFreeList = make(chan []byte, FreeListMaxItems)
func (tx *Tx) AddTxIn(txIn *txin.TxIn) {
	tx.ins = append(tx.ins, txIn)
}

func (tx *Tx) AddTxOut(txOut *txout.TxOut) {
	tx.outs = append(tx.outs, txOut)
}

func (tx *Tx) GetTxOut(index int) (out *txout.TxOut) {
	if index < 0 || index > len(tx.outs) {
		return nil
	}

	return tx.outs[index]
}

func (tx *Tx) GetAllPreviousOut() (outs []outpoint.OutPoint) {
	return

}

func (tx *Tx) GetOutsCount() int {
	return len(tx.outs)

}

func (tx *Tx) RemoveTxIn(txIn *txin.TxIn) {
	ret := tx.ins[:0]
	for _, e := range tx.ins {
		if e != txIn {
			ret = append(ret, e)
		}
	}
	tx.ins = ret
}

func (tx *Tx) RemoveTxOut(txOut *txout.TxOut) {
	ret := tx.outs[:0]
	for _, e := range tx.outs {
		if e != txOut {
			ret = append(ret, e)
		}
	}
	tx.outs = ret
}

func (tx *Tx) EncodeSize() uint {
	// Version 4 bytes + LockTime 4 bytes + Serialized varint size for the
	// number of transaction inputs and outputs.
	n := 8 + util.VarIntSerializeSize(uint64(len(tx.ins))) + util.VarIntSerializeSize(uint64(len(tx.outs)))

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
	return uint(n)
}

func (tx *Tx) Encode(writer io.Writer) error {
	err := util.BinarySerializer.PutUint32(writer, binary.LittleEndian, uint32(tx.version))
	if err != nil {
		return err
	}
	count := uint64(len(tx.ins))
	err = util.WriteVarInt(writer, count)
	if err != nil {
		return err
	}
	for _, txIn := range tx.ins {
		err := txIn.Serialize(writer)
		if err != nil {
			return err
		}
	}
	count = uint64(len(tx.outs))
	err = util.WriteVarInt(writer, count)
	if err != nil {
		return err
	}
	for _, txOut := range tx.outs {
		err := txOut.Serialize(writer)
		if err != nil {
			return err
		}
	}
	return util.BinarySerializer.PutUint32(writer, binary.LittleEndian, tx.lockTime)
}

func (tx *Tx) Decode(reader io.Reader) error {
	version, err := util.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return err
	}
	count, err := util.ReadVarInt(reader)
	if err != nil {
		return err
	}
	if count > uint64(MaxTxInPerMessage) {
		err = errors.Errorf("too many input tx to fit into max message size [count %d , max %d]", count, MaxTxInPerMessage)
		return err
	}

	tx.version = int32(version)
	tx.ins = make([]*txin.TxIn, count)

	for i := uint64(0); i < count; i++ {
		txIn := new(txin.TxIn)
		txIn.PreviousOutPoint = new(outpoint.OutPoint)
		txIn.PreviousOutPoint.Hash = *new(util.Hash)
		err = txIn.Unserialize(reader)
		if err != nil {
			return err
		}
		tx.ins[i] = txIn
	}
	count, err = util.ReadVarInt(reader)
	if err != nil {
		return err
	}

	tx.outs = make([]*txout.TxOut, count)
	for i := uint64(0); i < count; i++ {
		// The pointer is set now in case a script buffer is borrowed
		// and needs to be returned to the pool on error.
		txOut := new(txout.TxOut)
		err = txOut.Unserialize(reader)
		if err != nil {
			return err
		}
		tx.outs[i] = txOut
	}

	tx.lockTime, err = util.BinarySerializer.Uint32(reader, binary.LittleEndian)
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
		if c, err := in.GetScriptSig().GetSigOpCount(false); err == nil {
			n += c
		}
	}
	for _, out := range tx.outs {
		if c, err := out.GetScriptPubKey().GetSigOpCount(false); err == nil {
			n += c
		}
	}
	return n
}

func (tx *Tx) GetLockTime() uint32 {
	return tx.lockTime
}

func (tx *Tx) GetVersion() int32 {
	return tx.version
}

func (tx *Tx) CheckCoinbaseTransaction() error {
	if !tx.IsCoinBase() {
		return errcode.New(errcode.TxErrNotCoinBase)
	}
	//if !tx.CheckTransactionCommon(false) {
	//	return false
	//}
	/*
		if tx.ins[0].script.Size() < 2 || tx.ins[0].Script.Size() > 100 {
			return state.Dos(100, false, RejectInvalid, "bad-cb-length", false, "")
		}*/
	return nil
}

func (tx *Tx) CheckTransactionCommon(checkDupInput bool) error {
	//check inputs and outputs
	if len(tx.ins) == 0 {
		//state.Dos(10, false, RejectInvalid, "bad-txns-vin-empty", false, "")
		return errcode.New(errcode.TxErrRejectInvalid)
	}
	if len(tx.outs) == 0 {
		//state.Dos(10, false, RejectInvalid, "bad-txns-vout-empty", false, "")
		return errcode.New(errcode.TxErrRejectInvalid)
	}

	if tx.EncodeSize() > consensus.MaxTxSize {
		return errcode.New(errcode.TxErrRejectInvalid)
	}

	// check outputs money
	totalOut := int64(0)
	for _, out := range tx.outs {
		err := out.CheckValue()
		if err != nil {
			return err
		}
		totalOut += out.GetValue()
		if !amount.MoneyRange(totalOut) {
			//state.Dos(100, false, RejectInvalid, "bad-txns-txouttotal-toolarge", false, "")
			return errcode.New(errcode.TxErrRejectInvalid)
		}
	}

	// check sigopcount
	if tx.GetSigOpCountWithoutP2SH() > MaxTxSigOpsCounts {
		return errcode.New(errcode.TxErrRejectInvalid)
	}

	// check dup input
	if checkDupInput {
		outPointSet := make(map[*outpoint.OutPoint]struct{})
		for _, in := range tx.ins {
			if in.PreviousOutPoint.IsNull() {
				return errcode.New(errcode.TxErrRejectInvalid)
			}
			if _, ok := outPointSet[in.PreviousOutPoint]; !ok {
				outPointSet[in.PreviousOutPoint] = struct{}{}
			} else {
				return errcode.New(errcode.TxErrRejectInvalid)
			}
		}
	}

	return nil
}

func (tx *Tx) CheckStandard() error {
	// check version
	if tx.version > MaxStandardVersion || tx.version < 1 {
		return errcode.New(errcode.TxErrBadVersion)
	}

	// check size
	if tx.EncodeSize() > MaxStandardTxSize {
		return errcode.New(errcode.TxErrOverSize)
	}

	// check inputs script
	for _, in := range tx.ins {
		err := in.CheckStandard()
		if err != nil {
			return err
		}
	}

	// check output scriptpubkey and inputs scriptsig
	nDataOut := 0
	for _, out := range tx.outs {
		pubKeyType, err := out.CheckStandard()
		if err != nil {
			return err
		}
		if pubKeyType == script.ScriptNullData {
			nDataOut++
			// only one OP_RETURN txout is permitted
			if nDataOut > 1 {
				return errcode.New(errcode.ScriptErrMultiOpReturn)
			}
		}
		if pubKeyType == script.ScriptMultiSig && !conf.Cfg.Script.IsBareMultiSigStd {
			return errcode.New(errcode.ScriptErrBareMultiSig)
		}
		if out.IsDust(util.NewFeeRate(conf.Cfg.TxOut.DustRelayFee)) {
			return errcode.New(errcode.ScriptErrDustOut)
		}
	}

	if tx.GetSigOpCountWithoutP2SH() > int(MaxStandardTxSigOps) {
		return errcode.New(errcode.TxErrRejectInvalid)
	}
	return nil
}

func (tx *Tx) IsCommitment(data []byte) bool {
	for _, e := range tx.outs {
		if e.IsCommitment(data) {
			return true
		}
	}
	return false
}

//func (tx *Tx) returnScriptBuffers() {
//	for _, txIn := range tx.ins {
//		if txIn == nil || txIn.scriptSig == nil {
//			continue
//		}
//		scriptPool.Return(txIn.scriptSig.bytes)
//	}
//	for _, txOut := range tx.outs {
//		if txOut == nil || txOut.scriptPubKey == nil {
//			continue
//		}
//		scriptPool.Return(txOut.scriptPubKey.bytes)
//	}
//}

func (tx *Tx) GetValueOut() int64 {
	var valueOut int64
	for _, out := range tx.outs {
		valueOut += out.GetValue()
		if !amount.MoneyRange(out.GetValue()) || amount.MoneyRange(valueOut) {
			panic("value out of range")
		}
	}
	return valueOut
}

//func (tx *Tx) Copy() *Tx {
//	newTx := Tx{
//		Version:  tx.Version,
//		LockTime: tx.LockTime,
//		ins:      make([]*TxIn, 0, len(tx.ins)),
//		outs:     make([]*TxOut, 0, len(tx.outs)),
//	}
//	newTx.GetHash() = Tx.GetHash()
//
//	for _, txOut := range tx.outs {
//		scriptLen := len(txOut.Script.bytes)
//		newOutScript := make([]byte, scriptLen)
//		copy(newOutScript, txOut.GetScriptPubKey().GetByteCodes()[:scriptLen])
//
//		newTxOut := TxOut{
//			value:  txOut.value,
//			scriptPubKey: NewScriptRaw(newOutScript),
//		}
//		newTx.outs = append(newTx.outs, &newTxOut)
//	}
//	for _, txIn := range tx.ins {
//		var hashBytes [32]byte
//		copy(hashBytes[:], txIn.PreviousOutPoint.Hash[:])
//		preHash := new(util.Hash)
//		preHash.SetBytes(hashBytes[:])
//		newOutPoint := OutPoint{Hash: *preHash, Index: txIn.PreviousOutPoint.Index}
//		scriptLen := txIn.Script.Size()
//		newScript := make([]byte, scriptLen)
//		copy(newScript[:], txIn.Script.GetByteCodes()[:scriptLen])
//		newTxTmp := TxIn{
//			Sequence:         txIn.Sequence,
//			PreviousOutPoint: newOutPoint,
//			Script:           NewScriptRaw(newScript),
//		}
//		newTx.ins = append(newTx.ins, &newTxTmp)
//	}
//	return &newTx
//
//}

//func (tx *Tx) Equal(dstTx *Tx) bool {
//	originBuf := bytes.NewBuffer(nil)
//	tx.Serialize(originBuf)
//
//	dstBuf := bytes.NewBuffer(nil)
//	dstTx.Serialize(dstBuf)
//
//	return bytes.Equal(originBuf.Bytes(), dstBuf.Bytes())
//}

func (tx *Tx) ComputePriority(priorityInputs float64, txSize int) float64 {
	txModifiedSize := tx.CalculateModifiedSize()
	if txModifiedSize == 0 {
		return 0
	}
	return priorityInputs / float64(txModifiedSize)
}

func (tx *Tx) CalculateModifiedSize() uint {
	// In order to avoid disincentivizing cleaning up the UTXO set we don't
	// count the constant overhead for each txin and up to 110 bytes of
	// scriptSig (which is enough to cover a compressed pubkey p2sh redemption)
	// for priority. Providing any more cleanup incentive than making additional
	// inputs free would risk encouraging people to create junk outputs to
	// redeem later.
	txSize := tx.EncodeSize()
	/*for _, in := range tx.ins {

		InscriptModifiedSize := math.Min(110, float64(len(in.Script.bytes)))
		offset := 41 + int(InScriptModifiedSize)
		if txSize > offset {
			txSize -= offset
		}
	}*/

	return txSize
}

func (tx *Tx) IsFinal(Height int, time int64) bool {
	if tx.lockTime == 0 {
		return true
	}

	lockTimeLimit := int64(0)

	if tx.lockTime < script.LockTimeThreshold {
		lockTimeLimit = int64(Height)
	} else {
		lockTimeLimit = time
	}

	if int64(tx.lockTime) < lockTimeLimit {
		return true
	}

	for _, txin := range tx.ins {
		if txin.Sequence != script.SequenceFinal {
			return false
		}
	}

	return true
}

func (tx *Tx) String() string {
	str := ""
	//str = fmt.Sprintf(" hash :%s version : %d  lockTime: %d , ins:%d outs:%d \n", Tx.GetHash().ToString(), tx.Version, tx.LockTime, len(tx.ins), len(tx.outs))
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

func (tx *Tx) GetHash() util.Hash {
	// cache hash
	if !tx.hash.IsNull() {
		return tx.hash
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.EncodeSize()))
	_ = tx.Encode(buf)
	hash := util.DoubleSha256Hash(buf.Bytes())
	tx.hash = hash

	return tx.hash
}

func (tx *Tx) GetIns() []*txin.TxIn {
	return tx.ins
}

func (tx *Tx) GetOuts() []*txout.TxOut {
	return tx.outs
}

func NewTx(locktime uint32, version int32) *Tx {
	tx := &Tx{lockTime: locktime, version: version}
	tx.ins = make([]*txin.TxIn, 0)
	tx.outs = make([]*txout.TxOut, 0)
	return tx
}

func NewEmptyTx() *Tx {
	return &Tx{}
}

/*
// PrecomputedTransactionData Precompute sighash midstate to avoid quadratic hashing
type PrecomputedTransactionData struct {
	HashPrevout  *util.Hash
	HashSequence *util.Hash
	HashOutputs  *util.Hash
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
