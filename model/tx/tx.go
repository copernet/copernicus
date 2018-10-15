package tx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

const (
	TxOrphan = iota
	TxInvalid
	CoinAmount = 100000000
)

const (
	RequireStandard = 1
	DefaultVersion  = 0x01
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

	DefaultBlockPriorityPercentage int64 = 5

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

	// DefaultIncrementalRelayFee is default for -incrementalrelayfee, which sets the minimum feerate increase
	// for mempool limiting or BIP 125 replacement
	DefaultIncrementalRelayFee int64 = 1000

	// DefaultBytesPerSigop is default for -bytespersigop
	DefaultBytesPerSigop uint = 20

	// MaxStandardP2WSHStackItems is the maximum number of witness stack items in a standard P2WSH script
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

func (tx *Tx) AddTxIn(txIn *txin.TxIn) {
	tx.ins = append(tx.ins, txIn)
}

func (tx *Tx) AddTxOut(txOut *txout.TxOut) {
	tx.outs = append(tx.outs, txOut)
}

func (tx *Tx) GetTxOut(index int) (out *txout.TxOut) {
	if index < 0 || index > len(tx.outs) {
		log.Warn("GetTxOut index %d over large")
		return nil
	}

	return tx.outs[index]
}

func (tx *Tx) GetTxIn(index int) (out *txin.TxIn) {
	if index < 0 || index > len(tx.ins) {
		log.Warn("GetTxOut index %d over large")
		return nil
	}

	return tx.ins[index]
}

func (tx *Tx) GetAllPreviousOut() (outs []outpoint.OutPoint) {
	outs = make([]outpoint.OutPoint, 0, len(tx.ins))
	for _, txin := range tx.ins {
		outs = append(outs, *txin.PreviousOutPoint)
	}
	return
}

func (tx *Tx) PrevoutHashs() (outs []util.Hash) {
	outs = make([]util.Hash, 0, len(tx.ins))
	for _, txin := range tx.ins {
		outs = append(outs, txin.PreviousOutPoint.Hash)
	}
	return
}

func (tx *Tx) AnyInputTxIn(container *map[util.Hash]struct{}) bool {
	if container != nil {
		for _, txin := range tx.ins {
			prevout := txin.PreviousOutPoint.Hash
			if _, exists := (*container)[prevout]; exists {
				return true
			}
		}
	}

	return false
}

func (tx *Tx) GetOutsCount() int {
	return len(tx.outs)
}
func (tx *Tx) GetInsCount() int {
	return len(tx.ins)
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

func (tx *Tx) SerializeSize() uint32 {
	return tx.EncodeSize()
}

func (tx *Tx) Serialize(writer io.Writer) error {
	return tx.Encode(writer)
}

func (tx *Tx) Unserialize(reader io.Reader) error {
	return tx.Decode(reader)
}

func (tx *Tx) EncodeSize() uint32 {
	// Version 4 bytes + LockTime 4 bytes + Serialized varint size for the
	// number of transaction inputs and outputs.
	n := 8 + util.VarIntSerializeSize(uint64(len(tx.ins))) + util.VarIntSerializeSize(uint64(len(tx.outs)))

	for _, txIn := range tx.ins {
		n += txIn.EncodeSize()
	}
	for _, txOut := range tx.outs {
		n += txOut.EncodeSize()
	}

	return n
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
		err := txIn.Encode(writer)
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
		err := txOut.Encode(writer)
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
		log.Error("too many input txs to fit into max message size [count %d , max %d]", count, MaxTxInPerMessage)
		return err
	}

	tx.version = int32(version)
	//log.Debug("tx version %d", tx.version)
	tx.ins = make([]*txin.TxIn, count)
	for i := uint64(0); i < count; i++ {
		txIn := new(txin.TxIn)
		txIn.PreviousOutPoint = new(outpoint.OutPoint)
		err = txIn.Decode(reader)
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
		err = txOut.Decode(reader)
		if err != nil {
			return err
		}
		tx.outs[i] = txOut
	}

	tx.lockTime, err = util.BinarySerializer.Uint32(reader, binary.LittleEndian)
	return err
}

func (tx *Tx) IsCoinBase() bool {
	if len(tx.ins) != 1 {
		return false
	}

	return tx.ins[0].PreviousOutPoint.IsNull()
}

func (tx *Tx) GetSigOpCountWithoutP2SH() int {
	n := 0

	for _, in := range tx.ins {
		n += in.GetScriptSig().GetSigOpCount(false)
	}
	for _, out := range tx.outs {
		n += out.GetScriptPubKey().GetSigOpCount(false)
	}
	return n
}

func (tx *Tx) GetLockTime() uint32 {
	return tx.lockTime
}

func (tx *Tx) GetVersion() int32 {
	return tx.version
}

func (tx *Tx) CheckRegularTransaction() error {
	if tx.IsCoinBase() {
		log.Debug("tx should not be coinbase, hash: %s", tx.hash)
		return errcode.NewError(errcode.RejectInvalid, "bad-tx-coinbase")
	}

	err := tx.checkTransactionCommon(true)
	if err != nil {
		return err
	}

	for _, in := range tx.ins {
		if in.PreviousOutPoint.IsNull() {
			log.Debug("tx input prevout null")
			return errcode.New(errcode.RejectInvalid)
		}
	}

	return nil
}

func (tx *Tx) CheckCoinbaseTransaction() error {
	if !tx.IsCoinBase() {
		log.Warn("CheckCoinBaseTransaction: TxErrNotCoinBase")
		return errcode.New(errcode.RejectInvalid)
	}
	err := tx.checkTransactionCommon(false)
	if err != nil {
		return err
	}

	// coinbase in script check
	if tx.ins[0].GetScriptSig().Size() < 2 || tx.ins[0].GetScriptSig().Size() > 100 {
		log.Debug("coinbash input hash err script size")
		return errcode.New(errcode.RejectInvalid)
	}

	return nil
}

func (tx *Tx) checkTransactionCommon(checkDupInput bool) error {
	//check inputs and outputs
	if len(tx.ins) == 0 {
		log.Warn("bad tx: %s, empty ins", tx.hash)
		return errcode.NewError(errcode.RejectInvalid, "bad-txns-vin-empty")
	}
	if len(tx.outs) == 0 {
		log.Warn("bad tx: %s, empty out", tx.hash)
		return errcode.NewError(errcode.RejectInvalid, "bad-txns-vout-empty")
	}

	if tx.EncodeSize() > consensus.MaxTxSize {
		log.Warn("tx is oversize, tx:%v, tx size:%d, MaxTxSize:%d", tx, tx.EncodeSize(), consensus.MaxTxSize)
		return errcode.NewError(errcode.RejectInvalid, "bad-txns-oversize")
	}

	// check outputs money
	totalOut := amount.Amount(0)
	for _, out := range tx.outs {
		err := out.CheckValue()
		if err != nil {
			return err
		}

		totalOut += out.GetValue()
		if !amount.MoneyRange(totalOut) {
			log.Debug("bad tx: %s totalOut value :%d", tx.hash, totalOut)
			return errcode.NewError(errcode.RejectInvalid, "bad-txns-txouttotal-toolarge");)
		}
	}

	// check sigopcount
	if tx.GetSigOpCountWithoutP2SH() > MaxTxSigOpsCounts {
		log.Debug("bad tx sigops :%d", tx.GetSigOpCountWithoutP2SH())
		return errcode.New(errcode.RejectInvalid)
	}

	// check dup input
	if checkDupInput {
		outPointSet := make(map[outpoint.OutPoint]bool)
		err := tx.CheckDuplicateIns(&outPointSet)
		if err != nil {
			return err
		}
	}

	return nil
}
func (tx *Tx) CheckDuplicateIns(outpoints *map[outpoint.OutPoint]bool) error {
	for _, in := range tx.ins {
		if _, ok := (*outpoints)[*(in.PreviousOutPoint)]; !ok {
			(*outpoints)[*(in.PreviousOutPoint)] = true
		} else {
			log.Debug("bad tx, duplicate inputs")
			return errcode.New(errcode.RejectInvalid)
		}
	}
	return nil
}
func (tx *Tx) CheckStandard() error {
	// check version
	if tx.version > MaxStandardVersion || tx.version < 1 {
		log.Debug("TxErrBadVersion")
		return errcode.New(errcode.RejectNonstandard)
	}

	// check size
	if tx.EncodeSize() > uint32(MaxStandardTxSize) {
		log.Debug("TxErrBadStandardSize")
		return errcode.New(errcode.RejectNonstandard)
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
		} else if pubKeyType == script.ScriptMultiSig && !conf.Cfg.Script.IsBareMultiSigStd {
			log.Debug("TxErrBareMultiSig")
			return errcode.New(errcode.RejectNonstandard)
		} else if out.IsDust(util.NewFeeRate(conf.Cfg.TxOut.DustRelayFee)) {
			log.Debug("TxErrDustOut")
			return errcode.New(errcode.RejectNonstandard)
		}
	}

	// only one OP_RETURN txout is permitted
	if nDataOut > 1 {
		log.Debug("TxErrMultiOpReturn")
		return errcode.New(errcode.RejectNonstandard)
	}

	if tx.GetSigOpCountWithoutP2SH() > int(MaxStandardTxSigOps) {
		log.Debug("TxBadSigOps")
		return errcode.New(errcode.RejectNonstandard)
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

func (tx *Tx) GetValueOut() amount.Amount {
	var valueOut amount.Amount
	for _, out := range tx.outs {
		valueOut += out.GetValue()
		if !amount.MoneyRange(out.GetValue()) || !amount.MoneyRange(valueOut) {
			panic("value out of range")
		}
	}
	return valueOut
}

func (tx *Tx) SignStep(redeemScripts map[string]string, keys map[string]*crypto.PrivateKey,
	hashType uint32, scriptPubKey *script.Script, nIn int, value amount.Amount) (sigData [][]byte, pubKeyType int, err error) {
	pubKeyType, pubKeys, err := scriptPubKey.CheckScriptPubKeyStandard()
	if err != nil {
		log.Debug("SignStep CheckScriptPubKeyStandard err")
		return nil, pubKeyType, errcode.New(errcode.RejectInvalid)
	}
	if pubKeyType == script.ScriptNonStandard || pubKeyType == script.ScriptNullData {
		log.Debug("SignStep CheckScriptPubKeyStandard err")
		return nil, pubKeyType, errcode.New(errcode.RejectInvalid)
	}
	if pubKeyType == script.ScriptMultiSig {
		sigData = make([][]byte, 0, len(pubKeys)-2)
	} else {
		sigData = make([][]byte, 0, 1)
	}

	// return signatureData|hashType
	if pubKeyType == script.ScriptPubkey {
		pubKeyHashString := string(util.Hash160(pubKeys[0]))
		privateKey := keys[pubKeyHashString]
		signature, err := tx.signOne(scriptPubKey, privateKey, hashType, nIn, value)
		if err != nil {
			return nil, pubKeyType, err
		}
		sigBytes := signature.Serialize()
		sigBytes = append(sigBytes, byte(hashType))
		sigData = append(sigData, sigBytes)
		return sigData, pubKeyType, nil
	}
	// return signatureData|hashType + pubKeyHashString
	if pubKeyType == script.ScriptPubkeyHash {
		pubKeyHashString := string(pubKeys[0])
		privateKey := keys[pubKeyHashString]
		signature, err := tx.signOne(scriptPubKey, privateKey, hashType, nIn, value)
		if err != nil {
			return nil, pubKeyType, err
		}
		sigBytes := signature.Serialize()
		sigBytes = append(sigBytes, byte(hashType))
		sigData = append(sigData, sigBytes)
		pkBytes := privateKey.PubKey().ToBytes()
		sigData = append(sigData, pkBytes)
		return sigData, pubKeyType, nil
	}
	// signature1|hashType signature2|hashType...signatureM|hashType
	if pubKeyType == script.ScriptMultiSig {
		emptyBytes := []byte{}
		sigData = append(sigData, emptyBytes)
		nSigned := 0
		nRequired := int(pubKeys[0][0])
		for _, v := range pubKeys[1 : len(pubKeys)-1] {
			if nSigned < nRequired {
				pubKyHash := string(util.Hash160(v))
				privateKey := keys[pubKyHash]
				signature, err := tx.signOne(scriptPubKey, privateKey, hashType, nIn, value)
				if err != nil {
					continue
				}
				sigData = append(sigData, append(signature.Serialize(), byte(hashType)))
				nSigned++
			}
		}

		if nSigned != nRequired {
			log.Debug("SignStep signed not equal requiredSigs")
			return nil, pubKeyType, errcode.New(errcode.TxErrSignRawTransaction)
		}
		return sigData, pubKeyType, nil
	}
	// return redeemscript, outside will SignStep again use redeemScript
	if pubKeyType == script.ScriptHash {
		scriptHashString := string(pubKeys[0])
		redeemScriptString := redeemScripts[scriptHashString]
		if len(redeemScriptString) == 0 {
			log.Debug("SignStep redeemScript len err")
			return nil, pubKeyType, errcode.New(errcode.TxErrSignRawTransaction)
		}
		sigData = append(sigData, []byte(redeemScriptString))
		return sigData, pubKeyType, nil
	}
	log.Debug("SignStep err")
	return nil, pubKeyType, errcode.New(errcode.TxErrSignRawTransaction)
}

func (tx *Tx) signOne(scriptPubKey *script.Script, privateKey *crypto.PrivateKey, hashType uint32,
	nIn int, value amount.Amount) (signature *crypto.Signature, err error) {

	hash, err := SignatureHash(tx, scriptPubKey, hashType, nIn, value, script.ScriptEnableSigHashForkID)
	if err != nil {
		return nil, err
	}
	signature, err = privateKey.Sign(hash[:])
	return
}

func (tx *Tx) UpdateInScript(i int, scriptSig *script.Script) error {
	if i >= len(tx.ins) || i < 0 {
		log.Debug("TxErrInvalidIndexOfIn")
		return errcode.New(errcode.TxErrInvalidIndexOfIn)
	}
	tx.ins[i].SetScriptSig(scriptSig)

	if !tx.hash.IsNull() {
		tx.hash = tx.calHash()
	}

	return nil
}

func (tx *Tx) ComputePriority(priorityInputs float64, txSize int) float64 {
	txModifiedSize := tx.CalculateModifiedSize()
	if txModifiedSize == 0 {
		return 0
	}
	return priorityInputs / float64(txModifiedSize)
}

func (tx *Tx) CalculateModifiedSize() uint32 {
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

// IsFinal proceeds as follows
// 1. tx.locktime > 0 and tx.locktime < Threshold, use height to check(tx.locktime > current height)
// 2. tx.locktime > Threshold, use time to check(tx.locktime > current blocktime)
// 3. sequence can disable it
func (tx *Tx) IsFinal(Height int32, time int64) bool {
	if tx.lockTime == 0 {
		return true
	}

	var lockTimeLimit int64
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

	tx.hash = tx.calHash()
	return tx.hash
}

func (tx *Tx) calHash() util.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, tx.EncodeSize()))
	err := tx.Encode(buf)
	if err != nil {
		panic("tx encode failed: " + err.Error())
	}
	return util.DoubleSha256Hash(buf.Bytes())
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

func NewGenesisCoinbaseTx() *Tx {
	tx := NewTx(0, DefaultVersion)
	scriptSigNum := script.NewScriptNum(4)
	scriptSigString := "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
	//scriptSigData := make([][]byte, 0)
	//scriptSigData = append(scriptSigData, []byte(scriptSigString))

	scriptPubKeyBytes, _ := hex.DecodeString("04678afdb0fe5548271967f1a67130b7105cd6a828e03909" +
		"a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112" +
		"de5c384df7ba0b8d578a4c702b6bf11d5f")
	scriptPubKey := script.NewEmptyScript()
	scriptPubKey.PushSingleData(scriptPubKeyBytes)
	scriptPubKey.PushOpCode(opcodes.OP_CHECKSIG)

	scriptSig := script.NewEmptyScript()
	scriptSig.PushInt64(486604799)
	scriptSig.PushScriptNum(scriptSigNum)
	scriptSig.PushSingleData([]byte(scriptSigString))

	txIn := txin.NewTxIn(nil, scriptSig, math.MaxUint32)
	txOut := txout.NewTxOut(50*100000000, scriptPubKey)
	tx.AddTxIn(txIn)
	tx.AddTxOut(txOut)

	return tx
}
