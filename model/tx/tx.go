package tx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"github.com/btcboost/copernicus/model/txin"
	"github.com/btcboost/copernicus/model/txout"
	"github.com/pkg/errors"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/errcode"
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

	FreeListMaxItems          = 12500
	MaxMessagePayload         = 32 * 1024 * 1024
	MinTxInPayload            = 9 + util.Hash256Size
	MaxTxInPerMessage         = (MaxMessagePayload / MinTxInPayload) + 1
	TxVersion                 = 1
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
	DefaultBytesPerSigop uint= 20

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
	Hash     util.Hash // Cached transaction hash	todo defined a pointer will be the optimization
	lockTime uint32
	version  int32
	ins      []*txin.TxIn
	outs     []*txout.TxOut
	//ValState int
}

//var scriptPool ScriptFreeList = make(chan []byte, FreeListMaxItems)
func (tx *Tx) AddTxIn(txIn *txin.TxIn) {
	tx.ins = append(tx.ins, txIn)
}

func (tx *Tx) AddTxOut(txOut *txout.TxOut) {
	tx.outs = append(tx.outs, txOut)
}

func (tx *Tx) GetTxOut(index int) (out *txout.TxOut){
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

func (tx *Tx) SerializeSize() uint {
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

func (tx *Tx) Serialize(writer io.Writer) error {
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

func (tx *Tx)Unserialize(reader io.Reader) error {
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
		return errcode.New(errcode.TxErrEmptyInputs)
	}
	if len(tx.outs) == 0 {
		//state.Dos(10, false, RejectInvalid, "bad-txns-vout-empty", false, "")
		return nil
	}

	if tx.SerializeSize() > consensus.MaxTxSize {
		//state.Dos(100, false, RejectInvalid, "bad-txns-oversize", false, "")
		return nil
	}

	// check outputs money
	totalOut := int64(0)
	for _, out := range tx.outs {
		err := out.CheckValue()
		if err != nil {
			return err
		}
		totalOut += out.GetValue()
		if totalOut < 0 || totalOut > MaxMoney {
			//state.Dos(100, false, RejectInvalid, "bad-txns-txouttotal-toolarge", false, "")
			return errcode.New(errcode.TxErrTotalMoneyTooLarge)
		}
	}

	// check sigopcount
	if tx.GetSigOpCountWithoutP2SH() > MaxTxSigOpsCounts {
		return errcode.New(errcode.TxErrTooManySigOps)
	}

	// check dup input
	if checkDupInput {
		outPointSet := make(map[*outpoint.OutPoint]struct{})
		for _, in := range tx.ins {
			if _, ok := outPointSet[in.PreviousOutPoint]; !ok {
				outPointSet[in.PreviousOutPoint] = struct{}{}
			} else {
				return errcode.New(errcode.TxErrDupIns)
			}
		}
	}

	return nil
}

func (tx *Tx) checkStandard(allowLargeOpReturn bool) bool {
	// check version
	/*
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
		if pubKeyType == ScriptMultiSig && !IsBareMultiSigStd {
			state.Dos(100, false, RejectInvalid, "bare-multisig", false, "")
			return false
		}
		if pubKeyType == ScriptNullData {
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
	*/
	return true
}

func (tx *Tx) ContextualCheckTransaction(flag int) bool {
	/*flags := 0
	if flag > 0 {
		flags = flag
	}

	nBlockHeight := ActiveChain.Height() + 1

	var nLockTimeCutoff int64 = 0

	if flags & consensus.LocktimeMedianTimePast {
		nLockTimeCutoff = ActiveChain.Tip()->GetMedianTimePast()
	} else {
		nLockTimeCutoff = util2.GetAdjustedTime()
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
	}*/

	return true
}

func (tx *Tx) isOutputAlreadyExist() bool {
	/*
	for i, e := range tx.outs {
		outPoint := NewOutPoint(tx.GetID(), i)
		if GMempool.GetCoin(outPoint) {
			return false
		}
		if GUtxo.GetCoin(outPoint) {
			return false
		}
	}*/

	return true
}

func (tx *Tx) areInputsAvailable() bool {
	/*
	for e := range tx.ins {
		outPoint := e.PreviousOutPoint
		if !GMempool.GetCoin(outPoint) {
			return false
		}
		if !GUtxo.GetCoin(outPoint) {
			return false
		}
	}
	*/
	return true
}

func (tx *Tx) caculateLockPoint(flags uint) (lp *LockPoints) {
	/*
	lp = NewLockPoints()
	maxHeight int = 0
	maxTime int64 = 0
	for _, e := range tx.ins {
		if e.Sequence & SequenceLockTimeDisableFlag != 0 {
			continue
		}
		coin := mempool.GetCoin(e.PreviousOutPoint)
		if !coin {
			coin = utxo.GetCoin(e.PreviousOutPoint)
		}
		if !coin {
			lp = nil
			return
		}
		coinTime int64 = 0
		coinHeight := coin.GetHeight()
		if coinHeight == MEMPOOL_HEIGHT {
			coinHeight = ActiveChain.GetHeight() + 1
		}
		if e.Sequence & SequenceLockTimeTypeFlag != 0 {
			if coinHeight - 1 > 0 {
				coinTime = ActiveChain.Tip().GetAncesstor(coinHeight - 1).GetMedianTimePast()
			} else {
				coinTime = ActiveChain.Tip().GetAncesstor(0).GetMedianTimePast()
			}
			coinTime = ((e.Sequence & SequenceLockTimeMask) << wire.SequenceLockTimeGranularity) - 1
			if maxTime < coinTime {
				maxTime = coinTime
			}
		} else {
			if maxHeight < coinHeight {
				maxHeight = coinHeight
			}
		}
	}
	lp.MaxInputBlock = ActiveChain.GetAncestor(maxHeight)

	if tx.Version >= 2 && flags & LocktimeVerifySequence != 0 {
		lp.Height = maxHeight
		lp.Time = maxTime
		return
	}

	lp.Height = -1
	lp.Time = -1
	*/
	return
}
/*
func (tx *Tx) checkSequenceLocks(lp *LockPoints) bool {
	BlockTime := lp.MaxInputBlock.GetMedianTimePast()
	if lp.Height >= lp.Height || lp.Time >= BlockTime {
		return false
	}

	return true
}
*/
func (tx *Tx) areInputsStandard() bool {
	/*
	for _, e := range tx.ins {
		coin := utxo.GetCoin(e.PreviousOutPoint)
		if !coin {
			coin = mempool.GetCoin(e.PreviousOutPoint)
		}
		txOut := coin.txOut
		succeed, pubKeyType := txOut.CheckScript()
		if !succeed {
			return false
		}
		if pubKeyType == ScriptHash {
			subScript := NewScriptRaw(e.scriptSig.ParsedOpCodes[len(e.scriptSig.ParsedOpCodes) - 1].data)
			if subScript.GetSigOpCount(true) > MaxP2SHSigOps {
				return false
			}
		}
	}
*/
	return true
}

func (tx *Tx) CheckInputsMoney() bool {
	/*
	nValue := 0
	for _, e := range tx.ins {
		coin := mempool.GetCoin(e.PreviousOutPoint)
		if !coin {
			coin = utxo.GetCoin()
		}
		if coin.txout.value < 0 || coin.txout.value > MaxMoney {
			return false
		}
		nValue += coin.txout.value
		if nValue < 0 || nValue > MaxMoney {
			return false
		}
	}
	if nValue < tx.GetValueOut() {
		return false
	}
	*/
	return true
}

/*
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
*/
func (tx *Tx) GetValueOut() int64 {
	var valueOut int64
	for _, out := range tx.outs {
		valueOut += out.GetValue()
		if out.GetValue() < 0  || out.GetValue() > MaxMoney ||
			valueOut < 0 || valueOut> MaxMoney{
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
		copy(newOutScript, txOut.GetScriptPubKey().GetByteCodes()[:scriptLen])

		newTxOut := TxOut{
			value:  txOut.value,
			scriptPubKey: NewScriptRaw(newOutScript),
		}
		newTx.outs = append(newTx.outs, &newTxOut)
	}
	for _, txIn := range tx.ins {
		var hashBytes [32]byte
		copy(hashBytes[:], txIn.PreviousOutPoint.Hash[:])
		preHash := new(util.Hash)
		preHash.SetBytes(hashBytes[:])
		newOutPoint := OutPoint{Hash: *preHash, Index: txIn.PreviousOutPoint.Index}
		scriptLen := txIn.Script.Size()
		newScript := make([]byte, scriptLen)
		copy(newScript[:], txIn.Script.GetByteCodes()[:scriptLen])
		newTxTmp := TxIn{
			Sequence:         txIn.Sequence,
			PreviousOutPoint: newOutPoint,
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

func (tx *Tx) CalculateModifiedSize() uint {
	// In order to avoid disincentivizing cleaning up the UTXO set we don't
	// count the constant overhead for each txin and up to 110 bytes of
	// scriptSig (which is enough to cover a compressed pubkey p2sh redemption)
	// for priority. Providing any more cleanup incentive than making additional
	// inputs free would risk encouraging people to create junk outputs to
	// redeem later.
	txSize := tx.SerializeSize()
	/*for _, in := range tx.ins {

		InscriptModifiedSize := math.Min(110, float64(len(in.Script.bytes)))
		offset := 41 + int(InScriptModifiedSize)
		if txSize > offset {
			txSize -= offset
		}
	}*/

	return txSize
}

func (tx *Tx) isFinal(Height int, time int64) bool {
	if tx.lockTime == 0 {
		return true
	}

	lockTimeLimit := int64(0)
	/*
	if tx.LockTime < LockTimeThreshold {
		lockTimeLimit = int64(Height)
	} else {
		lockTimeLimit = time
	}
	*/
	if int64(tx.lockTime) < lockTimeLimit {
		return true
	}
	/*
	for _, txin := range tx.ins {
		if txin.Sequence != SequenceFinal {
			return false
		}
	}
	*/
	return true
}

func (tx *Tx) String() string {
	str := ""
	//str = fmt.Sprintf(" hash :%s version : %d  lockTime: %d , ins:%d outs:%d \n", tx.Hash.ToString(), tx.Version, tx.LockTime, len(tx.ins), len(tx.outs))
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

func (tx *Tx) TxHash() util.Hash {
	// cache hash
	if !tx.Hash.IsNull() {
		return tx.Hash
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	_ = tx.Serialize(buf)
	hash := util.DoubleSha256Hash(buf.Bytes())
	tx.Hash = hash
	return hash
}

// CheckSequenceLocks Check if transaction will be BIP 68 final in the next block to be created.
//
// Simulates calling SequenceLocks() with data from the tip of the current
// active chain. Optionally stores in LockPoints the resulting height and time
// calculated and the hash of the block needed for calculation or skips the
// calculation and uses the LockPoints passed in for evaluation. The LockPoints
// should not be considered valid if CheckSequenceLocks returns false.
//
// See core/core.h for flag definitions.
func (tx *Tx)CheckSequenceLocks(flags int, lp *LockPoints, useExistingLockPoints bool) bool {

	//TODO:AssertLockHeld(cs_main) and AssertLockHeld(mempool.cs) not finish
	/*
	tip := GChainActive.Tip()
	var index *BlockIndex
	index.Prev = tip
	// CheckSequenceLocks() uses chainActive.Height()+1 to evaluate height based
	// locks because when SequenceLocks() is called within ConnectBlock(), the
	// height of the block *being* evaluated is what is used. Thus if we want to
	// know if a transaction can be part of the *next* block, we need to use one
	// more than chainActive.Height()
	index.Height = tip.Height + 1
	lockPair := make(map[int]int64)

	if useExistingLockPoints {
		if lp == nil {
			panic("the mempool lockPoints is nil")
		}
		lockPair[lp.Height] = lp.Time
	} else {
		// pcoinsTip contains the UTXO set for chainActive.Tip()
		//viewMempool := mempool.CoinsViewMemPool{
		//	Base:  GCoinsTip,
		//	Mpool: GMemPool,
		//}
		var prevheights []int
		for txinIndex := 0; txinIndex < len(tx.Ins); txinIndex++ {
			//txin := tx.Ins[txinIndex]
			var coin *utxo.Coin
			//if !viewMempool.GetCoin(txin.PreviousOutPoint, coin) {
			//	logs.Error("Missing input")
			//	return false
			//}
			if coin.GetHeight() == consensus.MEMPOOL_HEIGHT {
				// Assume all mempool transaction confirm in the next block
				prevheights[txinIndex] = tip.Height + 1
			} else {
				prevheights[txinIndex] = int(coin.GetHeight())
			}
		}

		lockPair = tx.CalculateSequenceLocks(flags, prevheights, index)
		if lp != nil {
			lockPair[lp.Height] = lp.Time
			// Also store the hash of the block with the highest height of all
			// the blocks which have sequence locked prevouts. This hash needs
			// to still be on the chain for these LockPoint calculations to be
			// valid.
			// Note: It is impossible to correctly calculate a maxInputBlock if
			// any of the sequence locked inputs depend on unconfirmed txs,
			// except in the special case where the relative lock time/height is
			// 0, which is equivalent to no sequence lock. Since we assume input
			// height of tip+1 for mempool txs and test the resulting lockPair
			// from CalculateSequenceLocks against tip+1. We know
			// EvaluateSequenceLocks will fail if there was a non-zero sequence
			// lock on a mempool input, so we can use the return value of
			// CheckSequenceLocks to indicate the LockPoints validity
			maxInputHeight := 0
			for height := range prevheights {
				// Can ignore mempool inputs since we'll fail if they had non-zero locks
				if height != tip.Height+1 {
					maxInputHeight = int(math.Max(float64(maxInputHeight), float64(height)))
				}
			}
			lp.MaxInputBlock = tip.GetAncestor(maxInputHeight)
		}
	}
	return EvaluateSequenceLocks(index, lockPair)
	*/

	return true
}

func(tx *Tx)GetIns() []*txin.TxIn {
	return tx.ins
}

func(tx *Tx)GetOuts() []*txout.TxOut {
	return tx.outs
}

func NewTx(locktime uint32, version int32) *Tx {
	return &Tx{lockTime: locktime, version: version}
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
