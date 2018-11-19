package ltx

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/model/blockindex"
	"strconv"
	"strings"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lscript"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

type ScriptVerifyJob struct {
	Tx                     *tx.Tx
	ScriptSig              *script.Script
	ScriptPubKey           *script.Script
	IputNum                int
	Value                  amount.Amount
	Flags                  uint32
	ScriptChecker          lscript.Checker
	ScriptVerifyResultChan chan ScriptVerifyResult
}

type ScriptVerifyResult struct {
	TxHash       util.Hash
	ScriptSig    *script.Script
	ScriptPubKey *script.Script
	InputNum     int
	Err          error
}

type SignError struct {
	TxIn   *txin.TxIn
	ErrMsg string
}

func verifyResult(j ScriptVerifyJob, err error) ScriptVerifyResult {
	return ScriptVerifyResult{j.Tx.GetHash(), j.ScriptSig, j.ScriptPubKey, j.IputNum, err}
}

const (
	MaxScriptVerifyJobNum = 50000
)

var (
	scriptVerifyJobChan         chan ScriptVerifyJob
	blockScriptVerifyResultChan chan ScriptVerifyResult
	txScriptVerifyResultChan    chan ScriptVerifyResult
)

func ScriptVerifyInit() {
	scriptVerifyJobChan = make(chan ScriptVerifyJob, MaxScriptVerifyJobNum)
	blockScriptVerifyResultChan = make(chan ScriptVerifyResult, MaxScriptVerifyJobNum)
	txScriptVerifyResultChan = make(chan ScriptVerifyResult, MaxScriptVerifyJobNum)
	for i := 0; i < conf.Cfg.Script.Par; i++ {
		go checkScript()
	}
}

func CheckTxBeforeAcceptToMemPool(txn *tx.Tx) (*mempool.TxEntry, error) {
	if err := txn.CheckRegularTransaction(); err != nil {
		return nil, err
	}

	if model.ActiveNetParams.RequireStandard {
		ok, reason := txn.IsStandard()
		if !ok {
			log.Debug("non standard tx: %s, reason: %s", txn.GetHash(), reason)
			return nil, errcode.NewError(errcode.RejectNonstandard, reason)
		}
	}

	// check common locktime, sequence final can disable it
	err := ContextualCheckTransactionForCurrentBlock(txn, int(tx.StandardLockTimeVerifyFlags))
	if err != nil {
		return nil, err
	}

	// is mempool already have it? conflict tx with mempool
	gPool := mempool.GetInstance()
	gPool.RLock()
	defer gPool.RUnlock()
	if gPool.FindTx(txn.GetHash()) != nil {
		log.Debug("tx already known in mempool, hash: %s", txn.GetHash())
		return nil, errcode.NewError(errcode.RejectAlreadyKnown, "txn-already-in-mempool")
	}

	for _, e := range txn.GetIns() {
		if gPool.HasSpentOut(e.PreviousOutPoint) {
			log.Debug("tx ins alread spent out in mempool")
			return nil, errcode.NewError(errcode.RejectConflict, "txn-mempool-conflict")
		}
	}

	// check outpoint alread exist
	if areOutputsAlreadExist(txn) {
		log.Debug("tx's outpoint already known in utxo, tx hash: %s", txn.GetHash())
		return nil, errcode.NewError(errcode.RejectAlreadyKnown, "txn-already-known")
	}

	// are inputs are exists and available?
	inputCoins, missingInput, spendCoinbase := inputCoinsOf(txn)
	if missingInput {
		return nil, errcode.New(errcode.TxErrNoPreviousOut)
	}

	// CLTV(CheckLockTimeVerify)
	// Only accept BIP68 sequence locked transactions that can be mined
	// in the next block; we don't want our mempool filled up with
	// transactions that can't be mined yet. Must keep pool.cs for this
	// unless we change CheckSequenceLocks to take a CoinsViewCache
	// instead of create its own.
	lp := CalculateLockPoints(txn, uint32(tx.StandardLockTimeVerifyFlags))
	if lp == nil {
		log.Debug("cann't calculate out lockpoints")
		return nil, errcode.New(errcode.RejectNonstandard)
	}
	// Only accept BIP68 sequence locked transactions that can be mined
	// in the next block; we don't want our mempool filled up with
	// transactions that can't be mined yet. Must keep pool.cs for this
	// unless we change CheckSequenceLocks to take a CoinsViewCache
	// instead of create its own.
	if !CheckSequenceLocks(lp.Height, lp.Time) {
		log.Debug("tx sequence lock check faild")
		return nil, errcode.NewError(errcode.RejectNonstandard, "non-BIP68-final")
	}

	//check standard inputs
	if model.ActiveNetParams.RequireStandard {
		if !AreInputsStandard(txn, inputCoins) {
			return nil, errcode.NewError(errcode.RejectNonstandard, "bad-txns-nonstandard-inputs")
		}
	}

	// Check that the transaction doesn't have an excessive number of
	// sigops, making it impossible to mine. Since the coinbase transaction
	// itself can contain sigops MAX_STANDARD_TX_SIGOPS is less than
	// MAX_BLOCK_SIGOPS_PER_MB; we still consider this an invalid rather
	// than merely non-standard transaction.
	sigOpsCount := GetTransactionSigOpCount(txn, uint32(script.StandardScriptVerifyFlags), inputCoins)
	if uint(sigOpsCount) > tx.MaxStandardTxSigOps {
		return nil, errcode.NewError(errcode.RejectNonstandard, "bad-txns-too-many-sigops")
	}

	txFee, err := checkFee(txn, inputCoins)
	if err != nil {
		return nil, err
	}

	//TODO: Require that free transactions have sufficient priority to be mined in the next block
	//TODO: Continuously rate-limit free (really, very-low-fee) transactions.
	//TODO: check absurdly-high-fee (nFees > nAbsurdFee)

	var extraFlags uint32 = script.ScriptVerifyNone
	tip := chain.GetInstance().Tip()

	if model.IsReplayProtectionEnabled(tip.GetMedianTimePast()) {
		extraFlags |= script.ScriptEnableReplayProtection
	}

	if model.IsMagneticAnomalyEnabled(tip.GetMedianTimePast()) {
		extraFlags |= script.ScriptEnableCheckDataSig
	}

	//check inputs
	var scriptVerifyFlags = uint32(script.StandardScriptVerifyFlags)
	if !model.ActiveNetParams.RequireStandard {
		scriptVerifyFlags = promiscuousMempoolFlags() | script.ScriptEnableSigHashForkID
	}
	scriptVerifyFlags |= extraFlags

	// Check against previous transactions. This is done last to help
	// prevent CPU exhaustion denial-of-service attacks.
	err = checkInputs(txn, inputCoins, scriptVerifyFlags, txScriptVerifyResultChan)
	if err != nil {
		return nil, err
	}

	// Check again against the current block tip's script verification flags
	// to cache our script execution flags. This is, of course, useless if
	// the next block has different script flags from the previous one, but
	// because the cache tracks script flags for us it will auto-invalidate
	// and we'll just have a few blocks of extra misses on soft-fork
	// activation.
	//
	// This is also useful in case of bugs in the standard flags that cause
	// transactions to pass as valid when they're actually invalid. For
	// instance the STRICTENC flag was incorrectly allowing certain CHECKSIG
	// NOT scripts to pass, even though they were invalid.
	//
	// There is a similar check in CreateNewBlock() to prevent creating
	// invalid blocks (using TestBlockValidity), however allowing such
	// transactions into the mempool can be exploited as a DoS attack.
	var currentBlockScriptVerifyFlags = chain.GetInstance().GetBlockScriptFlags(tip)
	err = checkInputs(txn, inputCoins, currentBlockScriptVerifyFlags, txScriptVerifyResultChan)
	if err != nil {
		if ((^scriptVerifyFlags) & currentBlockScriptVerifyFlags) == 0 {
			return nil, errcode.New(errcode.ScriptCheckInputsBug)
		}
		err = checkInputs(txn, inputCoins, uint32(script.MandatoryScriptVerifyFlags)|extraFlags, txScriptVerifyResultChan)
		if err != nil {
			return nil, err
		}

		log.Debug("Warning: -promiscuousmempool flags set to not include currently enforced soft forks, " +
			"this may break mining or otherwise cause instability!\n")
	}

	txEntry := mempool.NewTxentry(txn, txFee, util.GetTime(),
		chain.GetInstance().Height(), *lp, sigOpsCount, spendCoinbase)

	return txEntry, nil
}

func checkFee(txn *tx.Tx, inputCoins *utxo.CoinsMap) (int64, error) {
	inputValue := inputCoins.GetValueIn(txn)
	txFee := inputValue - txn.GetValueOut()

	txsize := int64(txn.EncodeSize())
	minfeeRate := mempool.GetInstance().GetMinFee(conf.Cfg.Mempool.MaxPoolSize)
	rejectFee := minfeeRate.GetFee(int(txsize))

	if int64(txFee) < rejectFee {
		reason := fmt.Sprintf("mempool min fee not met %d < %d", txFee, rejectFee)
		log.Debug("reject tx:%s, for %s", txn.GetHash(), reason)
		return 0, errcode.NewError(errcode.RejectInsufficientFee, reason)
	}

	return int64(txFee), nil
}

func promiscuousMempoolFlags() uint32 {
	flag, err := strconv.Atoi(conf.Cfg.Script.PromiscuousMempoolFlags)
	if err != nil {
		return uint32(script.StandardScriptVerifyFlags)
	}
	return uint32(flag)
}

// CheckBlockTransactions block service use these 3 func to check transactions or to apply transaction while connecting block to active chain
func CheckBlockTransactions(txs []*tx.Tx, maxBlockSigOps uint64) error {
	txsLen := len(txs)
	if txsLen == 0 {
		log.Debug("block has no transactions")
		return errcode.NewError(errcode.RejectInvalid, "bad-cb-missing")
	}
	err := txs[0].CheckCoinbaseTransaction()
	if err != nil {
		return err
	}
	sigOps := txs[0].GetSigOpCountWithoutP2SH()
	outPointSet := make(map[outpoint.OutPoint]bool)
	for i, transaction := range txs[1:] {
		sigOps += txs[i+1].GetSigOpCountWithoutP2SH()
		if uint64(sigOps) > maxBlockSigOps {
			log.Debug("block has too many sigOps:%d", sigOps)
			return errcode.NewError(errcode.RejectInvalid, "bad-blk-sigops")
		}
		err := transaction.CheckRegularTransaction()
		if err != nil {
			return err
		}

		// check dup input
		err = transaction.CheckDuplicateIns(&outPointSet)
		if err != nil {
			return err
		}

	}
	return nil
}

func ContextureCheckBlockTransactions(txs []*tx.Tx, blockHeight int32, blockLockTime, mediaTimePast int64) error {
	txsLen := len(txs)
	if txsLen == 0 {
		log.Debug("no transactions err")
		return errcode.New(errcode.RejectInvalid)
	}
	err := contextureCheckBlockCoinBaseTransaction(txs[0], blockHeight)
	if err != nil {
		return err
	}

	var prevTx *tx.Tx
	for _, transaction := range txs {
		if model.IsMagneticAnomalyEnabled(mediaTimePast) {
			if prevTx != nil {
				transactionHash := transaction.GetHash()
				prevTxHash := prevTx.GetHash()
				if pow.HashToBig(&transactionHash).Cmp(pow.HashToBig(&prevTxHash)) < 0 {
					return fmt.Errorf(
						"bad-ordering: transaction order is invalid(%s <%s) in block(height %d)",
						transaction.GetHash().String(), prevTx.GetHash().String(), blockHeight)
				}
			}
			if prevTx != nil || !transaction.IsCoinBase() {
				prevTx = transaction
			}
		}

		err = ContextualCheckTransaction(transaction, blockHeight, blockLockTime, mediaTimePast)
		if err != nil {
			return err
		}
	}
	return nil
}

func ApplyBlockTransactions(txs []*tx.Tx, bip30Enable bool, scriptCheckFlags uint32,
	needCheckScript bool, blockSubSidy amount.Amount, blockHeight int32, blockMaxSigOpsCount uint64,
	lockTimeFlags uint32, pindex *blockindex.BlockIndex) (coinMap *utxo.CoinsMap, bundo *undo.BlockUndo, err error) {
	// make view
	coinsMap := utxo.NewEmptyCoinsMap()
	utxo := utxo.GetUtxoCacheInstance()
	sigOpsCount := uint64(0)
	var fees amount.Amount
	bundo = undo.NewBlockUndo(0)
	txUndoList := make([]*undo.TxUndo, 0, len(txs)-1)

	isMagneticAnomalyEnabled := model.IsMagneticAnomalyEnabled(pindex.GetMedianTimePast())
	//updateCoins
	for i, transaction := range txs {
		// check BIP30: do not allow overwriting unspent old transactions
		if bip30Enable {
			outs := transaction.GetOuts()
			for i := range outs {
				if utxo.HaveCoin(outpoint.NewOutPoint(transaction.GetHash(), uint32(i))) {
					log.Debug("tried to overwrite transaction")
					return nil, nil, errcode.NewError(errcode.RejectInvalid, "bad-txns-BIP30")
				}
			}
		}

		var valueIn amount.Amount
		if !transaction.IsCoinBase() {
			ins := transaction.GetIns()
			for _, in := range ins {
				coin := coinsMap.FetchCoin(in.PreviousOutPoint)
				if coin == nil || coin.IsSpent() {
					log.Debug("can't find coin or has been spent out before apply transaction: %+v", in.PreviousOutPoint)
					return nil, nil, errcode.NewError(errcode.RejectInvalid, "bad-txns-inputs-missingorspent")
				}
				valueIn += coin.GetAmount()
			}

			coinHeight, coinTime := CalculateSequenceLocks(transaction, coinsMap, lockTimeFlags)
			if !CheckSequenceLocks(coinHeight, coinTime) {
				log.Debug("block contains a non-bip68-final transaction")
				return nil, nil, errcode.NewError(errcode.RejectInvalid, "bad-txns-nonfinal")
			}
		}
		//check sigops
		sigsCount := GetTransactionSigOpCount(transaction, scriptCheckFlags, coinsMap)
		if sigsCount > tx.MaxTxSigOpsCounts {
			log.Debug("transaction has too many sigops")
			return nil, nil, errcode.NewError(errcode.RejectInvalid, "bad-txn-sigops")
		}
		sigOpsCount += uint64(sigsCount)
		if sigOpsCount > blockMaxSigOpsCount {
			log.Debug("block has too many sigops at %d transaction", i)
			return nil, nil, errcode.NewError(errcode.RejectInvalid, "bad-blk-sigops")
		}
		if transaction.IsCoinBase() || isMagneticAnomalyEnabled {
			UpdateTxCoins(transaction, coinsMap, nil, blockHeight)
			continue
		}

		fees += valueIn - transaction.GetValueOut()
		if needCheckScript {
			//check inputs
			err := checkInputs(transaction, coinsMap, scriptCheckFlags, blockScriptVerifyResultChan)
			if err != nil {
				if strings.Contains(err.Error(), "script-verify") {
					return nil, nil, errcode.NewError(errcode.RejectInvalid, "blk-bad-inputs")
				}
				return nil, nil, err
			}
		}

		//update temp coinsMap
		txundo := undo.NewTxUndo()
		if !isMagneticAnomalyEnabled {
			UpdateTxCoins(transaction, coinsMap, txundo, blockHeight)
		}
		txUndoList = append(txUndoList, txundo)
	}
	bundo.SetTxUndo(txUndoList)
	//check blockReward
	if txs[0].GetValueOut() > fees+blockSubSidy {
		log.Debug("coinbase pays too much")
		return nil, nil, errcode.NewError(errcode.RejectInvalid, "bad-cb-amount")
	}
	return coinsMap, bundo, nil
}

// check coinbase with height
func contextureCheckBlockCoinBaseTransaction(tx *tx.Tx, blockHeight int32) error {
	// Enforce rule that the coinbase starts with serialized block height
	if blockHeight > model.ActiveNetParams.BIP34Height {
		heightNumb := script.NewScriptNum(int64(blockHeight))
		coinBaseScriptSig := tx.GetIns()[0].GetScriptSig()
		//heightData := make([][]byte, 0)
		//heightData = append(heightData, heightNumb.Serialize())
		heightScript := script.NewEmptyScript()
		//heightScript.PushData(heightData)
		heightScript.PushScriptNum(heightNumb)
		if coinBaseScriptSig.Size() < heightScript.Size() {
			log.Debug("coinbase err, not start with blockheight")
			return errcode.NewError(errcode.RejectInvalid, "bad-cb-height")
		}
		scriptData := coinBaseScriptSig.GetData()[:heightScript.Size()]
		if !bytes.Equal(scriptData, heightScript.GetData()) {
			log.Debug("coinbase err, not start with blockheight")
			return errcode.NewError(errcode.RejectInvalid, "bad-cb-height")
		}
	}
	return nil
}

func ContextualCheckTransaction(txn *tx.Tx, nBlockHeight int32, nLockTimeCutoff, mediaTimePast int64) error {
	if !txn.IsFinal(nBlockHeight, nLockTimeCutoff) {
		log.Debug("txn is not final, hash: %s", txn.GetHash())
		return errcode.NewError(errcode.RejectInvalid, "bad-txns-nonfinal")
	}

	//if chainparams.IsUAHFEnabled(nBlockHeight) && nBlockHeight <= chainparams.ActiveNetParams.AntiReplayOpReturnSunsetHeight {
	//	if txn.IsCommitment(chainparams.ActiveNetParams.AntiReplayOpReturnCommitment) {
	//		log.Debug("txn is commitment")
	//		return errcode.New(errcode.RejectInvalid)
	//	}
	//}

	if model.IsMagneticAnomalyEnabled(mediaTimePast) {
		txnsize := txn.SerializeSize()
		if txnsize < consensus.MinTxSize {
			return fmt.Errorf(
				"bad-txn-undersize: tx(%d) should be equal to or greater than %d",
				txnsize, consensus.MinTxSize)
		}
	}
	return nil
}

//
//func GetBlockSubsidy(nHeight int32) uint32 {
//	var halvings uint32 = uint32(nHeight) / uint32(chainparams.ActiveNetParams.SubsidyHalvingInterval)
//	// Force block reward to zero when right shift is undefined.
//	if halvings >= 64 {
//		return 0
//	}
//	nSubsidy := 50 * util.COIN
//	// Subsidy is cut in half every 210,000 blocks which will occur
//	// approximately every 4 years.
//	return uint32(nSubsidy) >> halvings
//}

func areOutputsAlreadExist(transaction *tx.Tx) (exist bool) {
	utxo := utxo.GetUtxoCacheInstance()
	outs := transaction.GetOuts()
	for i := range outs {
		if utxo.HaveCoin(outpoint.NewOutPoint(transaction.GetHash(), uint32(i))) {
			return true
		}
	}
	return false
}

func inputCoinsOf(txn *tx.Tx) (coinMap *utxo.CoinsMap, missingInput bool, spendCoinbase bool) {
	coinMap = utxo.NewEmptyCoinsMap()

	for _, txin := range txn.GetIns() {
		prevout := txin.PreviousOutPoint

		coin := utxo.GetUtxoCacheInstance().GetCoin(prevout)
		if coin == nil {
			coin = mempool.GetInstance().GetCoin(prevout)
		}

		if coin == nil || coin.IsSpent() {
			return coinMap, true, spendCoinbase
		}

		if coin.IsCoinBase() {
			spendCoinbase = true
		}

		coinMap.AddCoin(prevout, coin, coin.IsCoinBase())
	}

	return coinMap, false, spendCoinbase
}

func GetTransactionSigOpCount(txn *tx.Tx, flags uint32, coinMap *utxo.CoinsMap) int {
	var sigOpsCount int
	if flags&script.ScriptVerifyP2SH == script.ScriptVerifyP2SH {
		sigOpsCount = GetSigOpCountWithP2SH(txn, coinMap)
	} else {
		sigOpsCount = txn.GetSigOpCountWithoutP2SH()
	}

	return sigOpsCount
}

// GetSigOpCountWithP2SH starting BIP16(Apr 1 2012), we should check p2sh
func GetSigOpCountWithP2SH(txn *tx.Tx, coinMap *utxo.CoinsMap) int {
	n := txn.GetSigOpCountWithoutP2SH()
	if txn.IsCoinBase() {
		return n
	}

	for _, txin := range txn.GetIns() {
		coin := coinMap.GetCoin(txin.PreviousOutPoint)
		if coin == nil {
			panic("can't find coin in temp coinsmap")
		}

		scriptPubKey := coin.GetScriptPubKey()
		if scriptPubKey.IsPayToScriptHash() {
			sigsCount := scriptPubKey.GetP2SHSigOpCount(txin.GetScriptSig())
			n += sigsCount
		}
	}

	return n
}

func ContextualCheckTransactionForCurrentBlock(transaction *tx.Tx, flags int) error {

	// By convention a negative value for flags indicates that the current
	// network-enforced consensus rules should be used. In a future soft-fork
	// scenario that would mean checking which rules would be enforced for the
	// next block and setting the appropriate flags. At the present time no
	// soft-forks are scheduled, so no flags are set.
	if flags < 0 {
		flags = 0
	}

	activeChain := chain.GetInstance()

	// BIP113 will require that time-locked transactions have nLockTime set to
	// less than the median time of the previous block they're contained in.
	// When the next block is created its previous block will be the current
	// chain tip, so we use that to calculate the median time passed to
	// ContextualCheckTransaction() if LOCKTIME_MEDIAN_TIME_PAST is set.
	var nLockTimeCutoff int64
	if flags&consensus.LocktimeMedianTimePast == consensus.LocktimeMedianTimePast {
		nLockTimeCutoff = activeChain.Tip().GetMedianTimePast()
	} else {
		nLockTimeCutoff = util.GetAdjustedTime()
	}

	nBlockHeight := activeChain.Height() + 1

	return ContextualCheckTransaction(transaction, nBlockHeight,
		nLockTimeCutoff, activeChain.Tip().GetMedianTimePast())
}

func AreInputsStandard(transaction *tx.Tx, coinsMap *utxo.CoinsMap) bool {
	ins := transaction.GetIns()
	for i, e := range ins {
		coin := coinsMap.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			log.Debug("bug! tx input cann't find coin in temp coinsmap")
			panic("bug! tx input cann't find coin in temp coinsmap")
		}
		txOut := coin.GetTxOut()
		pubKeyType, isStandard := txOut.GetPubKeyType()
		if !isStandard {
			log.Debug("AreInputsStandard GetPubkeyType err: not StandardScriptPubKey")
			return false
		}
		if pubKeyType == script.ScriptHash {
			scriptSig := e.GetScriptSig()
			stack := util.NewStack()
			err := lscript.EvalScript(stack, scriptSig, transaction, i, amount.Amount(0), script.ScriptVerifyNone,
				lscript.NewScriptEmptyChecker())
			if err != nil {
				log.Debug("AreInputsStandard EvalScript err: %v", err)
				return false
			}

			if stack.Empty() {
				log.Trace("AreInputsStandard: empty stack after EvalScript")
				return false
			}

			subScript := script.NewScriptRaw(scriptSig.ParsedOpCodes[len(scriptSig.ParsedOpCodes)-1].Data)
			opCount := subScript.GetSigOpCount(true)
			if uint(opCount) > tx.MaxP2SHSigOps {
				log.Debug("transaction has too many sigops")
				return false
			}
		}
	}

	return true
}

func checkInputs(tx *tx.Tx, tempCoinMap *utxo.CoinsMap, flags uint32,
	scriptVerifyResultChan chan ScriptVerifyResult) error {
	//check inputs money range
	bestBlockHash, _ := utxo.GetUtxoCacheInstance().GetBestBlock()
	spendHeight := chain.GetInstance().GetSpendHeight(&bestBlockHash)
	if spendHeight == -1 {
		log.Debug("indexMap can`t find bestblock")
		return errcode.New(errcode.RejectInvalid)
	}

	err := CheckInputsMoney(tx, tempCoinMap, spendHeight)
	if err != nil {
		return err
	}

	ins := tx.GetIns()
	insLen := len(ins)

	batches := insLen / MaxScriptVerifyJobNum
	reminder := insLen % MaxScriptVerifyJobNum
	if reminder > 0 {
		batches++
	}

	for batch := 0; batch < batches; batch++ {

		jobNum := MaxScriptVerifyJobNum
		if batch+1 == batches && reminder > 0 {
			jobNum = reminder
		}

		for j := 0; j < jobNum; j++ {
			index := batch*MaxScriptVerifyJobNum + j

			coin := tempCoinMap.GetCoin(ins[index].PreviousOutPoint)
			if coin == nil {
				panic("can't find coin in temp coinsmap")
			}
			scriptPubKey := coin.GetScriptPubKey()
			scriptSig := ins[index].GetScriptSig()
			log.Debug("Push Script verify job txid: %s, inex: %d", tx.GetHash().String(), index)
			scriptVerifyJobChan <- ScriptVerifyJob{tx, scriptSig, scriptPubKey, index,
				coin.GetAmount(), flags, lscript.NewScriptRealChecker(), scriptVerifyResultChan}
		}

		var err error

		//drain all result from channel
		for k := 0; k < jobNum; k++ {
			result := <-scriptVerifyResultChan
			if result.Err != nil {
				log.Debug("Read script verify err result: %v, tx hash: %s, index: %d, "+
					"len of scriptVerifyResultChan: %d", result.Err, result.TxHash.String(),
					result.InputNum, len(scriptVerifyResultChan))
				if err == nil {
					err = result.Err
				}
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func checkScript() {
	for {
		j := <-scriptVerifyJobChan

		err1 := lscript.VerifyScript(j.Tx, j.ScriptSig, j.ScriptPubKey, j.IputNum, j.Value, j.Flags, j.ScriptChecker)
		if err1 != nil {

			hasNonMandatoryFlags := (j.Flags & uint32(script.StandardNotMandatoryVerifyFlags)) != 0
			if hasNonMandatoryFlags {
				fallbackFlags := uint32(uint64(j.Flags) & uint64(^script.StandardNotMandatoryVerifyFlags))
				err2 := lscript.VerifyScript(j.Tx, j.ScriptSig, j.ScriptPubKey, j.IputNum, j.Value, fallbackFlags, j.ScriptChecker)
				if err2 == nil {
					j.ScriptVerifyResultChan <- verifyResult(j, errorNonMandatoryPass(j, err1))
					continue
				}
			}

			j.ScriptVerifyResultChan <- verifyResult(j, errorMandatoryFailed(j, err1))
			continue
		}

		j.ScriptVerifyResultChan <- verifyResult(j, nil)
	}
}

func errorMandatoryFailed(j ScriptVerifyJob, innerErr error) error {
	log.Debug("VerifyScript err, tx hash: %s, input: %d, scriptSig: %s, scriptPubKey: %s, err: %v",
		j.Tx.GetHash(), j.IputNum, hex.EncodeToString(j.ScriptSig.GetData()),
		hex.EncodeToString(j.ScriptPubKey.GetData()), innerErr)

	return errcode.MakeError(errcode.RejectInvalid, "mandatory-script-verify-flag-failed (%s)", innerErr)
}

func errorNonMandatoryPass(j ScriptVerifyJob, innerErr error) error {
	log.Debug("VerifyScript err, but without StandardNotMandatoryVerifyFlags success, tx hash: %s, "+
		"input: %d, scriptSig: %s, scriptPubKey: %s, err: %v", j.Tx.GetHash(),
		j.IputNum, hex.EncodeToString(j.ScriptSig.GetData()),
		hex.EncodeToString(j.ScriptPubKey.GetData()), innerErr)

	return errcode.MakeError(errcode.RejectNonstandard, "non-mandatory-script-verify-flag (%s)", innerErr)
}

//CalculateLockPoints calculate lockpoint(all ins' max time or height at which it can be spent) of transaction
func CalculateLockPoints(transaction *tx.Tx, flags uint32) (lp *mempool.LockPoints) {
	activeChain := chain.GetInstance()
	tipHeight := activeChain.Height()
	utxo := utxo.GetUtxoCacheInstance()

	var coinHeight int32
	ins := transaction.GetIns()
	preHeights := make([]int32, 0, len(ins))

	for _, e := range ins {
		coin := utxo.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			coin = mempool.GetInstance().GetCoin(e.PreviousOutPoint)
		}
		if coin == nil {
			return nil
		}
		if coin.IsMempoolCoin() {
			coinHeight = tipHeight + 1
		} else {
			coinHeight = coin.GetHeight()
		}
		preHeights = append(preHeights, coinHeight)
	}

	maxHeight, maxTime := calculateSequenceLockPair(transaction, preHeights, flags)
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
	lp = mempool.NewLockPoints()
	lp.Height = maxHeight
	lp.Time = maxTime

	var maxInputHeight int32
	for _, height := range preHeights {
		if height != tipHeight+1 {
			if maxInputHeight < height {
				maxInputHeight = height
			}
		}
	}
	lp.MaxInputBlock = activeChain.GetAncestor(maxInputHeight)

	return
}

func CalculateSequenceLocks(transaction *tx.Tx, coinsMap *utxo.CoinsMap, flags uint32) (height int32, time int64) {
	ins := transaction.GetIns()
	preHeights := make([]int32, 0, len(ins))
	var coinHeight int32
	for _, e := range ins {
		coin := coinsMap.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			panic("no coin")
		}
		coinHeight = coin.GetHeight()
		preHeights = append(preHeights, coinHeight)
	}
	return calculateSequenceLockPair(transaction, preHeights, flags)
}

// calculate lockpoint(all ins' max time or height at which it can be spent) of transaction
func calculateSequenceLockPair(transaction *tx.Tx, preHeight []int32, flags uint32) (height int32, time int64) {
	var maxHeight int32 = -1
	var maxTime int64 = -1

	// tx.nVersion is signed integer so requires cast to unsigned otherwise
	// we would be doing a signed comparison and half the range of nVersion
	// wouldn't support BIP 68.
	fEnforceBIP68 := false
	if transaction.GetVersion() >= 2 && flags&consensus.LocktimeVerifySequence == consensus.LocktimeVerifySequence {
		fEnforceBIP68 = true
	}

	// Do not enforce sequence numbers as a relative lock time
	// unless we have been instructed to
	if !fEnforceBIP68 {
		return maxHeight, maxTime
	}

	activeChain := chain.GetInstance()
	ins := transaction.GetIns()
	for i, e := range ins {
		// Sequence numbers with the most significant bit set are not
		// treated as relative lock-times, nor are they given any
		// consensus-enforced meaning at this point.
		if e.Sequence&script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
			preHeight[i] = 0
			continue
		}

		coinHeight := preHeight[i]
		if e.Sequence&script.SequenceLockTimeTypeFlag == script.SequenceLockTimeTypeFlag {
			coinTime := activeChain.GetAncestor(coinHeight - 1).GetMedianTimePast()
			// NOTE: Subtract 1 to maintain nLockTime semantics.
			// BIP 68 relative lock times have the semantics of calculating the
			// first block or time at which the transaction would be valid. When
			// calculating the effective block time or height for the entire
			// transaction, we switch to using the semantics of nLockTime which
			// is the last invalid block time or height. Thus we subtract 1 from
			// the calculated time or height.

			// Time-based relative lock-times are measured from the smallest
			// allowed timestamp of the block containing the txout being spent,
			// which is the median time past of the block prior.
			expireTime := coinTime + ((int64(e.Sequence) & script.SequenceLockTimeMask) <<
				script.SequenceLockTimeGranularity) - 1
			if maxTime < expireTime {
				maxTime = expireTime
			}
		} else {
			expireHeight := coinHeight + (int32(e.Sequence) & script.SequenceLockTimeMask) - 1
			if maxHeight < expireHeight {
				maxHeight = expireHeight
			}
		}
	}

	return maxHeight, maxTime
}

func CheckSequenceLocks(height int32, time int64) bool {
	activeChain := chain.GetInstance()
	blockTime := activeChain.Tip().GetMedianTimePast()
	if height >= activeChain.Height()+1 || time >= blockTime {
		return false
	}
	return true
}

func CheckInputsMoney(transaction *tx.Tx, coinsMap *utxo.CoinsMap, spendHeight int32) (err error) {
	nValue := amount.Amount(0)
	ins := transaction.GetIns()
	for _, e := range ins {
		coin := coinsMap.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			log.Debug("CheckInputsMoney can't find coin")
			panic("CheckInputsMoney can't find coin")
		}

		if coin.IsCoinBase() {
			if spendHeight-coin.GetHeight() < consensus.CoinbaseMaturity {
				log.Debug("CheckInputsMoney coinbase can't spend now")
				return errcode.NewError(errcode.RejectInvalid, "bad-txns-premature-spend-of-coinbase")
			}
		}

		coinOut := coin.GetTxOut()
		if !amount.MoneyRange(coinOut.GetValue()) {
			log.Debug("CheckInputsMoney coin money range err")
			return errcode.NewError(errcode.RejectInvalid, "bad-txns-inputvalues-outofrange")
		}

		nValue += coinOut.GetValue()
		if !amount.MoneyRange(nValue) {
			log.Debug("CheckInputsMoney total coin money range err")
			return errcode.NewError(errcode.RejectInvalid, "bad-txns-inputvalues-outofrange")
		}
	}

	if nValue < transaction.GetValueOut() {
		log.Debug("CheckInputsMoney coins money little than out's")
		return errcode.NewError(errcode.RejectInvalid, "bad-txns-in-belowout")
	}

	txFee := nValue - transaction.GetValueOut()
	if !amount.MoneyRange(txFee) {
		log.Debug("CheckInputsMoney fee err")
		return errcode.NewError(errcode.RejectInvalid, "bad-txns-fee-outofrange")
	}
	return nil
}

func isCoinValid(coin *utxo.Coin, out *outpoint.OutPoint) bool {
	if coin == nil {
		return false
	}
	if coin.IsMempoolCoin() {
		return !mempool.GetInstance().HasSpentOut(out)
	}
	return !coin.IsSpent()
}

func SignRawTransaction(transactions []*tx.Tx, redeemScripts map[outpoint.OutPoint]*script.Script,
	keyStore *crypto.KeyStore, coinsMap *utxo.CoinsMap, hashType uint32) []*SignError {
	var err error
	var signErrors []*SignError

	mergedTx := transactions[0]
	hashSingle := int(hashType) & ^(crypto.SigHashAnyoneCanpay|crypto.SigHashForkID) == crypto.SigHashSingle

	for index, in := range mergedTx.GetIns() {
		coin := coinsMap.GetCoin(in.PreviousOutPoint)
		if !isCoinValid(coin, in.PreviousOutPoint) {
			signErrors = append(signErrors, &SignError{
				TxIn:   in,
				ErrMsg: "Input not found or already spent",
			})
			continue
		}

		scriptSig := script.NewEmptyScript()
		scriptPubKey := coin.GetScriptPubKey()
		value := coin.GetAmount()

		// Only sign SIGHASH_SINGLE if there's a corresponding output
		if !hashSingle || index < mergedTx.GetOutsCount() {
			redeemScript := redeemScripts[*in.PreviousOutPoint]
			// Sign what we can
			sigData, err := mergedTx.SignStep(index, keyStore, redeemScript, hashType, scriptPubKey, value)
			if err != nil {
				log.Info("SignStep error:%s", err.Error())
			} else {
				scriptSig.PushMultData(sigData)
				err = lscript.VerifyScript(mergedTx, scriptSig, scriptPubKey, index, value,
					uint32(script.StandardScriptVerifyFlags), lscript.NewScriptRealChecker())
				if err != nil {
					scriptSig = script.NewEmptyScript()
					log.Info("VerifyScript error:%s", err.Error())
				}
			}
		}

		// ... and merge in other signatures
		for _, transaction := range transactions {
			if len(transaction.GetIns()) > index {
				scriptSig, err = CombineSignature(transaction, scriptPubKey, scriptSig,
					transaction.GetIns()[index].GetScriptSig(), index, value,
					uint32(script.StandardScriptVerifyFlags), lscript.NewScriptRealChecker())
				if err != nil {
					log.Info("CombineSignature error:%s", err.Error())
				}
			}
		}

		err = mergedTx.UpdateInScript(index, scriptSig)
		if err != nil {
			log.Info("UpdateInScript error:%s", err.Error())
		}

		err = lscript.VerifyScript(mergedTx, scriptSig, scriptPubKey, index, value,
			uint32(script.StandardScriptVerifyFlags), lscript.NewScriptRealChecker())
		if err != nil {
			signErrors = append(signErrors, &SignError{
				TxIn:   in,
				ErrMsg: err.Error(),
			})
			continue
		}
	}
	return signErrors
}

func CombineSignature(transaction *tx.Tx, prevPubKey *script.Script, scriptSig *script.Script,
	txOldScriptSig *script.Script, nIn int, money amount.Amount, flags uint32,
	scriptChecker lscript.Checker) (*script.Script, error) {
	if scriptSig == nil {
		scriptSig = script.NewEmptyScript()
	}
	pubKeyType, pubKeys, isStandard := prevPubKey.IsStandardScriptPubKey()
	if !isStandard {
		// ScriptErrNonStandard returns error
		// Don't know anything about this, assume bigger one is correct
		if scriptSig.Size() >= txOldScriptSig.Size() {
			return scriptSig, nil
		}
		return txOldScriptSig, nil
	}
	if pubKeyType == script.ScriptNonStandard || pubKeyType == script.ScriptNullData {
		if scriptSig.Size() >= txOldScriptSig.Size() {
			return scriptSig, nil
		}
		return txOldScriptSig, nil
	}
	if pubKeyType == script.ScriptPubkey || pubKeyType == script.ScriptPubkeyHash {
		if len(scriptSig.ParsedOpCodes) == 0 {
			return txOldScriptSig, nil
		}
		if len(scriptSig.ParsedOpCodes[0].Data) == 0 {
			return txOldScriptSig, nil
		}
		return scriptSig, nil
	}
	if pubKeyType == script.ScriptMultiSig {
		sigData := make([][]byte, 0, len(scriptSig.ParsedOpCodes))
		sigData = append(sigData, []byte{})

		okSigs := make(map[string][]byte, len(scriptSig.ParsedOpCodes))

		// parseOpCodes is the variable of script, put the two script signature to a slice,
		// find both script's signature and check it, then combine them to a result slice
		var parsedOpCodes = scriptSig.ParsedOpCodes
		parsedOpCodes = append(parsedOpCodes, txOldScriptSig.ParsedOpCodes...)
		for _, opCode := range parsedOpCodes {
			for _, pubKey := range pubKeys[1 : len(pubKeys)-1] {
				if okSigs[string(pubKey)] != nil {
					continue
				}
				ok, err := scriptChecker.CheckSig(transaction, opCode.Data, pubKey, prevPubKey, nIn, money, flags)
				if err == nil && ok {
					okSigs[string(pubKey)] = opCode.Data
					break
				}
			}
		}
		sigN := 0
		sigsRequired := int(pubKeys[0][0])
		for _, pubKey := range pubKeys {
			if okSigs[string(pubKey)] != nil {
				sigData = append(sigData, okSigs[string(pubKey)])
				sigN++
				if sigN >= sigsRequired {
					break
				}
			}
		}

		// if the amount of signature is not the required, then put the OP_0
		for sigN < sigsRequired {
			sigData = append(sigData, []byte{})
			sigN++
		}
		scriptResult := script.NewEmptyScript()
		scriptResult.PushMultData(sigData)
		return scriptResult, nil
	}
	if pubKeyType == script.ScriptHash {
		if len(scriptSig.ParsedOpCodes) == 0 {
			return txOldScriptSig, nil
		}
		if len(scriptSig.ParsedOpCodes[0].Data) == 0 {
			return txOldScriptSig, nil
		}
		if len(txOldScriptSig.ParsedOpCodes) == 0 {
			return scriptSig, nil
		}
		if len(txOldScriptSig.ParsedOpCodes[0].Data) == 0 {
			return scriptSig, nil
		}
		redeemScript := script.NewScriptRaw(scriptSig.ParsedOpCodes[len(scriptSig.ParsedOpCodes)-1].Data)
		scriptSig = scriptSig.RemoveOpCodeByIndex(len(scriptSig.ParsedOpCodes) - 1)
		txOldScriptSig = txOldScriptSig.RemoveOpCodeByIndex(len(txOldScriptSig.ParsedOpCodes) - 1)
		scriptResult, err := CombineSignature(transaction, redeemScript, scriptSig,
			txOldScriptSig, nIn, money, flags, scriptChecker)
		scriptResult.PushSingleData(redeemScript.GetData())
		return scriptResult, err
	}
	log.Debug("TxErrPubKeyType")
	return script.NewEmptyScript(), errcode.New(errcode.TxErrPubKeyType)
}
