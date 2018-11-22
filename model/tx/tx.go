package tx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
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
	if index < 0 || index >= len(tx.outs) {
		log.Warn("GetTxOut index %d over large")
		return nil
	}

	return tx.outs[index]
}

func (tx *Tx) GetTxIn(index int) (out *txin.TxIn) {
	if index < 0 || index >= len(tx.ins) {
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

func (tx *Tx) AnyInputTxIn(container map[util.Hash]struct{}) bool {
	if container != nil {
		for _, txin := range tx.ins {
			prevout := txin.PreviousOutPoint.Hash
			if _, exists := container[prevout]; exists {
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

func (tx *Tx) GetSigOpCountWithoutP2SH(flags uint32) int {
	n := 0
	for _, in := range tx.ins {
		n += in.GetScriptSig().GetSigOpCount(flags, false)
	}
	for _, out := range tx.outs {
		n += out.GetScriptPubKey().GetSigOpCount(flags, false)
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
			return errcode.NewError(errcode.RejectInvalid, "bad-txns-prevout-null")
		}
	}

	return nil
}

func (tx *Tx) CheckCoinbaseTransaction() error {
	if !tx.IsCoinBase() {
		log.Warn("CheckCoinBaseTransaction: TxErrNotCoinBase")
		return errcode.NewError(errcode.RejectInvalid, "bad-cb-missing")
	}
	err := tx.checkTransactionCommon(false)
	if err != nil {
		return err
	}

	// coinbase in script check
	if tx.ins[0].GetScriptSig().Size() < 2 || tx.ins[0].GetScriptSig().Size() > 100 {
		log.Debug("coinbash input hash err script size")
		return errcode.NewError(errcode.RejectInvalid, "bad-cb-length")
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
			return errcode.NewError(errcode.RejectInvalid, "bad-txns-txouttotal-toolarge")
		}
	}

	// check sigopcount
	sigOpCount := tx.GetSigOpCountWithoutP2SH(script.ScriptEnableCheckDataSig)
	if sigOpCount > MaxTxSigOpsCounts {
		log.Debug("bad tx: %s bad-txn-sigops :%d", tx.hash, sigOpCount)
		return errcode.NewError(errcode.RejectInvalid, "bad-txn-sigops")
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
		if _, exists := (*outpoints)[*(in.PreviousOutPoint)]; !exists {
			(*outpoints)[*(in.PreviousOutPoint)] = true
		} else {
			log.Error("bad tx: %s, duplicate inputs:[%s:%d]",
				tx.hash, in.PreviousOutPoint.Hash, in.PreviousOutPoint.Index)
			return errcode.NewError(errcode.RejectInvalid, "bad-txns-inputs-duplicate")
		}
	}
	return nil
}

func (tx *Tx) IsStandard() (bool, string) {
	// check version
	if tx.version > MaxStandardVersion || tx.version < 1 {
		return false, "version"
	}

	// check size
	if tx.EncodeSize() > uint32(MaxStandardTxSize) {
		return false, "tx-size"
	}

	// check inputs script
	for _, in := range tx.ins {
		ok, reason := in.CheckStandard()
		if !ok {
			return false, reason
		}
	}

	// check output scriptpubkey and inputs scriptsig
	nDataOut := 0
	for _, out := range tx.outs {
		pubKeyType, isStandard := out.IsStandard()
		if !isStandard {
			return false, "scriptpubkey"
		}

		if pubKeyType == script.ScriptNullData {
			nDataOut++
		} else if pubKeyType == script.ScriptMultiSig && !conf.Cfg.Script.IsBareMultiSigStd {
			return false, "bare-multisig"
		} else if out.IsDust(util.NewFeeRate(conf.Cfg.TxOut.DustRelayFee)) {
			return false, "dust"
		}
	}

	// only one OP_RETURN txout is permitted
	if nDataOut > 1 {
		return false, "multi-op-return"
	}

	return true, ""
}

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

func (tx *Tx) SignStep(nIn int, keyStore *crypto.KeyStore, redeemScript *script.Script, hashType uint32,
	scriptPubKey *script.Script, value amount.Amount) (sigData [][]byte, err error) {
	pubKeyType, pubKeys, isStandard := scriptPubKey.IsStandardScriptPubKey()
	if !isStandard || pubKeyType == script.ScriptNonStandard || pubKeyType == script.ScriptNullData {
		log.Debug("SignStep IsStandardScriptPubKey err")
		return nil, errcode.New(errcode.RejectNonstandard)
	}
	scriptPubKeySign := scriptPubKey

	isScriptHash := false
	if pubKeyType == script.ScriptHash {
		if redeemScript == nil {
			return nil, errors.New("redeem script not found")
		}
		isScriptHash = true
		scriptPubKeySign = redeemScript
		pubKeyType, pubKeys, isStandard = scriptPubKeySign.IsStandardScriptPubKey()
		if !isStandard || pubKeyType == script.ScriptNonStandard || pubKeyType == script.ScriptNullData {
			return nil, errcode.New(errcode.RejectNonstandard)
		}
	}

	sigData = make([][]byte, 0)
	if pubKeyType == script.ScriptPubkey {
		keyPair := keyStore.GetKeyPairByPubKey(pubKeys[0])
		if keyPair == nil {
			return nil, errors.New("private key not found")
		}
		signature, err := tx.signOne(scriptPubKeySign, keyPair.GetPrivateKey(), hashType, nIn, value)
		if err != nil {
			return nil, err
		}
		sigBytes := append(signature.Serialize(), byte(hashType))
		// <signature>
		sigData = append(sigData, sigBytes)

	} else if pubKeyType == script.ScriptPubkeyHash {
		keyPair := keyStore.GetKeyPair(pubKeys[0])
		if keyPair == nil {
			return nil, errors.New("private key not found")
		}
		signature, err := tx.signOne(scriptPubKeySign, keyPair.GetPrivateKey(), hashType, nIn, value)
		if err != nil {
			return nil, err
		}
		sigBytes := append(signature.Serialize(), byte(hashType))
		pkBytes := keyPair.GetPublicKey().ToBytes()
		// <signature> <pubkey>
		sigData = append(sigData, sigBytes, pkBytes)

	} else if pubKeyType == script.ScriptMultiSig {
		signed := 0
		required := int(pubKeys[0][0])
		// <OP_0> <signature0> ... <signatureM>
		sigData = append(sigData, []byte{})
		for _, pubKey := range pubKeys[1:] {
			keyPair := keyStore.GetKeyPairByPubKey(pubKey)
			if keyPair == nil {
				log.Info("Private key not found:%s", hex.EncodeToString(pubKey))
				continue
			}
			signature, err := tx.signOne(scriptPubKeySign, keyPair.GetPrivateKey(), hashType, nIn, value)
			if err != nil {
				log.Info("getSignatureData error:%s", err.Error())
				continue
			}
			sigBytes := append(signature.Serialize(), byte(hashType))
			sigData = append(sigData, sigBytes)
			signed++
			if signed == required {
				break
			}
		}
		if signed != required {
			errMsg := fmt.Sprintf("ScriptMultiSig signed(%d) does not match required(%d)", signed, required)
			return nil, errors.New(errMsg)
		}
	} else {
		errMsg := fmt.Sprintf("unexpected script type(%d)", pubKeyType)
		return nil, errors.New(errMsg)
	}

	if isScriptHash {
		// <signature> <redeemscript>
		sigData = append(sigData, redeemScript.GetData())
	}

	return sigData, nil
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

// IsFinal proceeds as follows
// 1. tx.locktime > 0 and tx.locktime < Threshold, use height to check final (tx.locktime < current height)
// 2. tx.locktime > Threshold, use time to check final (tx.locktime < current blocktime)
// 3. sequence can disable it(sequence == sequencefinal)
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

func (tx *Tx) InsertTxOut(pos int, txOut *txout.TxOut) {
	if pos > len(tx.outs) {
		tx.outs = append(tx.outs, txOut)
		return
	}
	rear := append([]*txout.TxOut{}, tx.outs[pos:]...)
	tx.outs = append(tx.outs[:pos], txOut)
	tx.outs = append(tx.outs, rear...)
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

	scriptPubKeyBytes, _ := hex.DecodeString("04678afdb0fe5548271967f1a67130b7105cd6a828e03909" +
		"a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112" +
		"de5c384df7ba0b8d578a4c702b6bf11d5f")
	scriptPubKey := script.NewEmptyScript()
	err := scriptPubKey.PushSingleData(scriptPubKeyBytes)
	if err != nil {
		log.Error("push single data error:%v", err)
		return nil
	}
	err = scriptPubKey.PushOpCode(opcodes.OP_CHECKSIG)
	if err != nil {
		log.Error("push single data error:%v", err)
		return nil
	}

	scriptSig := script.NewEmptyScript()
	err = scriptSig.PushInt64(486604799)
	if err != nil {
		log.Error("push int64 error:%v", err)
		return nil
	}
	err = scriptSig.PushScriptNum(scriptSigNum)
	if err != nil {
		log.Error("push script num error:%v", err)
		return nil
	}
	err = scriptSig.PushSingleData([]byte(scriptSigString))
	if err != nil {
		log.Error("push single data error:%v", err)
		return nil
	}

	txIn := txin.NewTxIn(nil, scriptSig, math.MaxUint32)
	txOut := txout.NewTxOut(50*100000000, scriptPubKey)
	tx.AddTxIn(txIn)
	tx.AddTxOut(txOut)

	return tx
}
