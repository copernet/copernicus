package tx

import (
	"bytes"
	lscript "github.com/copernet/copernicus/logic/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"

	"strconv"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/chainparams"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/pkg/errors"
)

var ScriptVerifyChan chan struct {
	txHash       util.Hash
	scriptSig    *script.Script
	scriptPubKey *script.Script
	err          error
}

// CheckRegularTransaction transaction service will use this func to check transaction before accepting to mempool
func CheckRegularTransaction(transaction *tx.Tx) error {
	err := transaction.CheckRegularTransaction()
	if err != nil {
		return err
	}

	// check standard
	if chainparams.ActiveNetParams.RequireStandard {
		err := transaction.CheckStandard()
		if err != nil {
			return err
		}
	}

	// check common locktime, sequence final can disable it
	err = ContextualCheckTransactionForCurrentBlock(transaction, int(tx.StandardLockTimeVerifyFlags))
	if err != nil {
		return errcode.New(errcode.TxErrRejectNonstandard)
	}

	// is mempool already have it? conflict tx with mempool
	gPool := mempool.GetInstance()
	if gPool.FindTx(transaction.GetHash()) != nil {
		log.Debug("tx already known in mempool")
		return errcode.New(errcode.TxErrRejectAlreadyKnown)
	}

	// check preout already spent
	ins := transaction.GetIns()
	for _, e := range ins {
		if gPool.HasSpentOut(e.PreviousOutPoint) {
			log.Debug("tx ins alread spent out in mempool")
			return errcode.New(errcode.TxErrRejectConflict)
		}
	}

	// check outpoint alread exist
	if areOutputsAlreadExist(transaction) {
		log.Debug("tx already known in utxo")
		return errcode.New(errcode.TxErrRejectAlreadyKnown)
	}

	// check inputs are avaliable
	tempCoinsMap := utxo.NewEmptyCoinsMap()
	if !areInputsAvailable(transaction, tempCoinsMap) {
		return errcode.New(errcode.TxErrNoPreviousOut)
	}

	// CLTV(CheckLockTimeVerify)
	// Only accept BIP68 sequence locked transactions that can be mined
	// in the next block; we don't want our mempool filled up with
	// transactions that can't be mined yet. Must keep pool.cs for this
	// unless we change CheckSequenceLocks to take a CoinsViewCache
	// instead of create its own.
	lp := CalculateLockPoints(transaction, uint32(tx.StandardLockTimeVerifyFlags))
	if lp == nil {
		log.Debug("cann't calculate out lockpoints")
		return errcode.New(errcode.TxErrRejectNonstandard)
	}
	// Only accept BIP68 sequence locked transactions that can be mined
	// in the next block; we don't want our mempool filled up with
	// transactions that can't be mined yet. Must keep pool.cs for this
	// unless we change CheckSequenceLocks to take a CoinsViewCache
	// instead of create its own.
	if !CheckSequenceLocks(lp.Height, lp.Time) {
		log.Debug("tx sequence lock check faild")
		return errcode.New(errcode.TxErrRejectNonstandard)
	}

	//check standard inputs
	if chainparams.ActiveNetParams.RequireStandard {
		err = checkInputsStandard(transaction, tempCoinsMap)
		if err != nil {
			return err
		}
	}

	//check inputs
	var scriptVerifyFlags = uint32(script.StandardScriptVerifyFlags)
	if !chainparams.ActiveNetParams.RequireStandard {
		configVerifyFlags, err := strconv.Atoi(conf.Cfg.Script.PromiscuousMempoolFlags)
		if err != nil {
			panic("config PromiscuousMempoolFlags err")
		}
		scriptVerifyFlags = uint32(configVerifyFlags) | script.ScriptEnableSigHashForkID
	}

	var extraFlags uint32 = script.ScriptVerifyNone
	tip := chain.GetInstance().Tip()
	//if chainparams.IsMagneticAnomalyEnable(tip.GetMedianTimePast()) {
	//	extraFlags |= script.ScriptEnableCheckDataSig
	//}
	if chainparams.IsMonolithEnabled(tip.GetMedianTimePast()) {
		extraFlags |= script.ScriptEnableMonolithOpcodes
	}

	if chainparams.IsReplayProtectionEnabled(tip.GetMedianTimePast()) {
		extraFlags |= script.ScriptEnableReplayProtection
	}
	scriptVerifyFlags |= extraFlags

	err = checkInputs(transaction, tempCoinsMap, scriptVerifyFlags)
	if err != nil {
		return err
	}
	var currentBlockScriptVerifyFlags = chain.GetInstance().GetBlockScriptFlags(tip)
	err = checkInputs(transaction, tempCoinsMap, currentBlockScriptVerifyFlags)
	if err != nil {
		if ((^scriptVerifyFlags) & currentBlockScriptVerifyFlags) == 0 {
			return errcode.New(errcode.ScriptCheckInputsBug)
		}
		err = checkInputs(transaction, tempCoinsMap, uint32(script.MandatoryScriptVerifyFlags)|extraFlags)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckBlockTransactions block service use these 3 func to check transactions or to apply transaction while connecting block to active chain
func CheckBlockTransactions(txs []*tx.Tx, maxBlockSigOps uint64) error {
	txsLen := len(txs)
	if txsLen == 0 {
		log.Debug("block has no transcations")
		return errcode.New(errcode.TxErrRejectInvalid)
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
			return errcode.New(errcode.TxErrRejectInvalid)
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

func ContextureCheckBlockTransactions(txs []*tx.Tx, blockHeight int32, blockLockTime int64) error {
	txsLen := len(txs)
	if txsLen == 0 {
		log.Debug("no transactions err")
		return errcode.New(errcode.TxErrRejectInvalid)
	}
	err := contextureCheckBlockCoinBaseTransaction(txs[0], blockHeight)
	if err != nil {
		return err
	}

	for _, transaction := range txs[1:] {
		err = ContextualCheckTransaction(transaction, blockHeight, blockLockTime)
		if err != nil {
			return err
		}
	}
	return nil
}

//func ApplyGeniusBlockTransactions(txs []*tx.Tx) (coinMap *utxo.CoinsMap, bundo *undo.BlockUndo, err error) {
//	coinMap = utxo.NewEmptyCoinsMap()
//	bundo = undo.NewBlockUndo(0)
//	txUndoList := make([]*undo.TxUndo, 0, len(txs)-1)
//	for _, transaction := range txs {
//		if transaction.IsCoinBase() {
//			UpdateTxCoins(transaction, coinMap, nil, 0)
//			continue
//		}
//		txundo := undo.NewTxUndo()
//		UpdateTxCoins(transaction, coinMap, txundo, 0)
//		txUndoList = append(txUndoList, txundo)
//	}
//
//	bundo.SetTxUndo(txUndoList)
//	return
//}

func ApplyBlockTransactions(txs []*tx.Tx, bip30Enable bool, scriptCheckFlags uint32, needCheckScript bool,
	blockSubSidy amount.Amount, blockHeight int32, blockMaxSigOpsCount uint64) (coinMap *utxo.CoinsMap, bundo *undo.BlockUndo, err error) {
	// make view
	coinsMap := utxo.NewEmptyCoinsMap()
	utxo := utxo.GetUtxoCacheInstance()
	sigOpsCount := uint64(0)
	var fees amount.Amount
	bundo = undo.NewBlockUndo(0)
	txUndoList := make([]*undo.TxUndo, 0, len(txs)-1)
	//updateCoins
	for i, transaction := range txs {
		//check duplicate out
		if bip30Enable {
			outs := transaction.GetOuts()
			for i := range outs {
				coin := utxo.GetCoin(outpoint.NewOutPoint(transaction.GetHash(), uint32(i)))
				if coin != nil {
					log.Debug("can't find coin before apply transaction")
					return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
				}
			}
		}
		var valueIn amount.Amount
		if !transaction.IsCoinBase() {
			ins := transaction.GetIns()
			for _, in := range ins {
				coin := coinsMap.FetchCoin(in.PreviousOutPoint)
				if coin == nil || coin.IsSpent() {
					log.Debug("can't find coin or has been spent out before apply transaction")
					return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
				}
				valueIn += coin.GetAmount()
			}
			coinHeight, coinTime := CaculateSequenceLocks(transaction, coinsMap, scriptCheckFlags)
			if !CheckSequenceLocks(coinHeight, coinTime) {
				log.Debug("block contains a non-bip68-final transaction")
				return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
			}
		}
		//check sigops
		sigsCount := GetTransactionSigOpCount(transaction, scriptCheckFlags, coinsMap)
		if sigsCount > tx.MaxTxSigOpsCounts {
			log.Debug("transaction has too many sigops")
			return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
		}
		sigOpsCount += uint64(sigsCount)
		if sigOpsCount > blockMaxSigOpsCount {
			log.Debug("block has too many sigops at %d transaction", i)
			return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
		}
		if transaction.IsCoinBase() {
			UpdateTxCoins(transaction, coinsMap, nil, blockHeight)
			continue
		}

		fees += valueIn - transaction.GetValueOut()
		if needCheckScript {
			//check inputs
			err := checkInputs(transaction, coinsMap, scriptCheckFlags)
			if err != nil {
				return nil, nil, err
			}
		}

		//update temp coinsMap
		txundo := undo.NewTxUndo()
		UpdateTxCoins(transaction, coinsMap, txundo, blockHeight)
		txUndoList = append(txUndoList, txundo)
	}
	bundo.SetTxUndo(txUndoList)
	//check blockReward
	if txs[0].GetValueOut() > fees+blockSubSidy {
		log.Debug("coinbase pays too much")
		return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
	}
	return coinsMap, bundo, nil
}

// check coinbase with height
func contextureCheckBlockCoinBaseTransaction(tx *tx.Tx, blockHeight int32) error {
	// Enforce rule that the coinbase starts with serialized block height
	if blockHeight > chainparams.ActiveNetParams.BIP34Height {
		heightNumb := script.NewScriptNum(int64(blockHeight))
		coinBaseScriptSig := tx.GetIns()[0].GetScriptSig()
		//heightData := make([][]byte, 0)
		//heightData = append(heightData, heightNumb.Serialize())
		heightScript := script.NewEmptyScript()
		//heightScript.PushData(heightData)
		heightScript.PushScriptNum(heightNumb)
		if coinBaseScriptSig.Size() < heightScript.Size() {
			log.Debug("coinbase err, not start with blockheight")
			return errcode.New(errcode.TxErrRejectInvalid)
		}
		scriptData := coinBaseScriptSig.GetData()[:heightScript.Size()]
		if !bytes.Equal(scriptData, heightScript.GetData()) {
			log.Debug("coinbase err, not start with blockheight")
			return errcode.New(errcode.TxErrRejectInvalid)
		}
	}
	return nil
}

func ContextualCheckTransaction(transaction *tx.Tx, nBlockHeight int32, nLockTimeCutoff int64) error {
	if !transaction.IsFinal(nBlockHeight, nLockTimeCutoff) {
		log.Debug("transaction is not final")
		return errcode.New(errcode.TxErrRejectInvalid)
	}

	//if chainparams.IsUAHFEnabled(nBlockHeight) && nBlockHeight <= chainparams.ActiveNetParams.AntiReplayOpReturnSunsetHeight {
	//	if transaction.IsCommitment(chainparams.ActiveNetParams.AntiReplayOpReturnCommitment) {
	//		log.Debug("transaction is commitment")
	//		return errcode.New(errcode.TxErrRejectInvalid)
	//	}
	//}

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

func areInputsAvailable(transaction *tx.Tx, coinMap *utxo.CoinsMap) bool {
	gMempool := mempool.GetInstance()
	utxo := utxo.GetUtxoCacheInstance()
	ins := transaction.GetIns()
	for _, e := range ins {
		coin := utxo.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			coin = gMempool.GetCoin(e.PreviousOutPoint)
		}
		if coin == nil {
			log.Debug("inpute can't find coin")
			return false
		}
		if coin.IsSpent() {
			log.Debug("inpute coin is already spent out")
			return false
		}
		coinMap.AddCoin(e.PreviousOutPoint, coin, coin.IsCoinBase())
	}

	return true
}

func GetTransactionSigOpCount(transaction *tx.Tx, flags uint32, coinMap *utxo.CoinsMap) int {
	sigOpsCount := 0
	if flags&script.ScriptVerifyP2SH == script.ScriptVerifyP2SH {
		sigOpsCount = GetSigOpCountWithP2SH(transaction, coinMap)
	} else {
		sigOpsCount = transaction.GetSigOpCountWithoutP2SH()
	}

	return sigOpsCount
}

// GetSigOpCountWithP2SH starting BIP16(Apr 1 2012), we should check p2sh
func GetSigOpCountWithP2SH(transaction *tx.Tx, coinMap *utxo.CoinsMap) int {
	n := transaction.GetSigOpCountWithoutP2SH()
	if transaction.IsCoinBase() {
		return n
	}

	ins := transaction.GetIns()
	for _, e := range ins {
		coin := coinMap.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			panic("can't find coin in temp coinsmap")
		}
		scriptPubKey := coin.GetScriptPubKey()
		if scriptPubKey.IsPayToScriptHash() {
			sigsCount := scriptPubKey.GetSigOpCount(true)
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

	return ContextualCheckTransaction(transaction, nBlockHeight, nLockTimeCutoff)
}

func checkInputsStandard(transaction *tx.Tx, coinsMap *utxo.CoinsMap) error {
	ins := transaction.GetIns()
	for i, e := range ins {
		coin := coinsMap.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			log.Debug("bug! tx input cann't find coin in temp coinsmap")
			panic("bug! tx input cann't find coin in temp coinsmap")
		}
		txOut := coin.GetTxOut()
		pubKeyType, err := txOut.GetPubKeyType()
		if err != nil {
			log.Debug("checkInputsStandard GetPubkeyType err: %v", err)
			return errcode.New(errcode.TxErrRejectNonstandard)
		}
		if pubKeyType == script.ScriptHash {
			scriptSig := e.GetScriptSig()
			err = lscript.EvalScript(util.NewStack(), scriptSig, transaction, i, amount.Amount(0), script.ScriptVerifyNone,
				lscript.NewScriptEmptyChecker())
			if err != nil {
				log.Debug("checkInputsStandard EvalScript err: %v", err)
				return errcode.New(errcode.TxErrRejectNonstandard)
			}
			subScript := script.NewScriptRaw(scriptSig.ParsedOpCodes[len(scriptSig.ParsedOpCodes)-1].Data)
			opCount := subScript.GetSigOpCount(true)
			if uint(opCount) > tx.MaxP2SHSigOps {
				log.Debug("transaction has too many sigops")
				return errcode.New(errcode.TxErrRejectNonstandard)
			}
		}
	}

	return nil
}

func checkInputs(tx *tx.Tx, tempCoinMap *utxo.CoinsMap, flags uint32) error {
	//check inputs money range
	bestBlockHash, _ := utxo.GetUtxoCacheInstance().GetBestBlock()
	spendHeight := chain.GetInstance().GetSpendHeight(&bestBlockHash)
	if spendHeight == -1 {
		return errors.New("indexMap can`t find block")
	}

	err := CheckInputsMoney(tx, tempCoinMap, spendHeight)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	ins := tx.GetIns()
	for i, in := range ins {
		coin := tempCoinMap.GetCoin(in.PreviousOutPoint)
		if coin == nil {
			panic("can't find coin in temp coinsmap")
		}
		scriptPubKey := coin.GetScriptPubKey()
		scriptSig := in.GetScriptSig()
		err := lscript.VerifyScript(tx, scriptSig, scriptPubKey, i, coin.GetAmount(), flags, lscript.NewScriptRealChecker())
		if err != nil {
			if ((flags & uint32(script.StandardNotMandatoryVerifyFlags)) ==
				uint32(script.StandardNotMandatoryVerifyFlags)) || (flags&script.ScriptEnableMonolithOpcodes == 0) {
				err = lscript.VerifyScript(tx, scriptSig, scriptPubKey, i, coin.GetAmount(),
					uint32(uint64(flags)&uint64(^script.StandardNotMandatoryVerifyFlags)|
						script.ScriptEnableMonolithOpcodes), lscript.NewScriptRealChecker())
			}
			if err == nil {
				log.Debug("verifyScript err, but without StandardNotMandatoryVerifyFlags success")
				return errcode.New(errcode.TxErrRejectNonstandard)
			}
			log.Debug("verifyScript err, coin:%v, preout hash: %s, preout index: %d", *coin,
				in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
			return errcode.New(errcode.TxErrRejectInvalid)
		}
	}
	return nil
}

//func checkLockTime(lockTime int64, txLockTime int64, sequence uint32) bool {
//	// There are two kinds of nLockTime: lock-by-blockheight and
//	// lock-by-blocktime, distinguished by whether nLockTime <
//	// LOCKTIME_THRESHOLD.
//	//
//	// We want to compare apples to apples, so fail the script unless the type
//	// of nLockTime being tested is the same as the nLockTime in the
//	// transaction.
//	if !((txLockTime < script.LockTimeThreshold && lockTime < script.LockTimeThreshold) ||
//		(txLockTime >= script.LockTimeThreshold && lockTime >= script.LockTimeThreshold)) {
//		return false
//	}
//
//	// Now that we know we're comparing apples-to-apples, the comparison is a
//	// simple numeric one.
//	if lockTime > txLockTime {
//		return false
//	}
//	// Finally the nLockTime feature can be disabled and thus
//	// checkLockTimeVerify bypassed if every txIN has been finalized by setting
//	// nSequence to maxInt. The transaction would be allowed into the
//	// blockChain, making the opCode ineffective.
//	//
//	// Testing if this vin is not final is sufficient to prevent this condition.
//	// Alternatively we could test all inputs, but testing just this input
//	// minimizes the data required to prove correct checkLockTimeVerify
//	// execution.
//	if script.SequenceFinal == sequence {
//		return false
//	}
//	return true
//}
//
//func checkSequence(sequence int64, txToSequence int64, txVersion uint32) bool {
//	// Fail if the transaction's version number is not set high enough to
//	// trigger BIP 68 rules.
//	if txVersion < 2 {
//		return false
//	}
//	// Sequence numbers with their most significant bit set are not consensus
//	// constrained. Testing that the transaction's sequence number do not have
//	// this bit set prevents using this property to get around a
//	// checkSequenceVerify check.
//	if txToSequence&script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
//		return false
//	}
//	// Mask off any bits that do not have consensus-enforced meaning before
//	// doing the integer comparisons
//	nLockTimeMask := script.SequenceLockTimeTypeFlag | script.SequenceLockTimeMask
//	txToSequenceMasked := txToSequence & int64(nLockTimeMask)
//	nSequenceMasked := sequence & int64(nLockTimeMask)
//
//	// There are two kinds of nSequence: lock-by-blockHeight and
//	// lock-by-blockTime, distinguished by whether nSequenceMasked <
//	// CTxIn::SEQUENCE_LOCKTIME_TYPE_FLAG.
//	//
//	// We want to compare apples to apples, so fail the script unless the type
//	// of nSequenceMasked being tested is the same as the nSequenceMasked in the
//	// transaction.
//	if !((txToSequenceMasked < script.SequenceLockTimeTypeFlag && nSequenceMasked < script.SequenceLockTimeTypeFlag) ||
//		(txToSequenceMasked >= script.SequenceLockTimeTypeFlag && nSequenceMasked >= script.SequenceLockTimeTypeFlag)) {
//		return false
//	}
//	if nSequenceMasked > txToSequenceMasked {
//		return false
//	}
//	return true
//}

//CalculateLockPoints caculate lockpoint(all ins' max time or height at which it can be spent) of transaction
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

func CaculateSequenceLocks(transaction *tx.Tx, coinsMap *utxo.CoinsMap, flags uint32) (height int32, time int64) {
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

// caculate lockpoint(all ins' max time or height at which it can be spent) of transaction
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
				return errcode.New(errcode.TxErrRejectInvalid)
			}
		}
		txOut := coin.GetTxOut()
		if !amount.MoneyRange(txOut.GetValue()) {
			log.Debug("CheckInputsMoney coin money range err")
			return errcode.New(errcode.TxErrRejectInvalid)
		}
		nValue += txOut.GetValue()
		if !amount.MoneyRange(nValue) {
			log.Debug("CheckInputsMoney total coin money range err")
			return errcode.New(errcode.TxErrRejectInvalid)
		}
	}
	if nValue < transaction.GetValueOut() {
		log.Debug("CheckInputsMoney coins money little than out's")
		return errcode.New(errcode.TxErrRejectInvalid)
	}

	txFee := nValue - transaction.GetValueOut()
	if !amount.MoneyRange(txFee) {
		log.Debug("CheckInputsMoney fee err")
		return errcode.New(errcode.TxErrRejectInvalid)
	}
	return nil
}

//
//func CheckSig(transaction *tx.Tx, signature []byte, pubKey []byte, scriptCode *script.Script,
//	nIn int, money amount.Amount, flags uint32) (bool, error) {
//	if len(signature) == 0 || len(pubKey) == 0 {
//		return false, nil
//	}
//	hashType := signature[len(signature)-1]
//	txSigHash, err := tx.SignatureHash(transaction, scriptCode, uint32(hashType), nIn, money, flags)
//	if err != nil {
//		return false, err
//	}
//	signature = signature[:len(signature)-1]
//	txHash := transaction.GetHash()
//	//log.Debug("CheckSig: txid: %s, txSigHash: %s, signature: %s, pubkey: %s", txHash.String(),
//	//	txSigHash.String(), hex.EncodeToString(signature), hex.EncodeToString(pubKey))
//	fOk := tx.CheckSig(txSigHash, signature, pubKey)
//	log.Debug("CheckSig: txid: %s, txSigHash: %s, signature: %s, pubkey: %s, result: %v", txHash.String(),
//		txSigHash.String(), hex.EncodeToString(signature), hex.EncodeToString(pubKey), fOk)
//	//if !fOk {
//	//	panic("CheckSig failed")
//	//}
//	return fOk, err
//}

// SignRawTransaction txs, preouts, private key, hash type
func SignRawTransaction(transaction *tx.Tx, redeemScripts map[string]string, keys map[string]*crypto.PrivateKey,
	hashType uint32) (err error) {
	coinMap := utxo.NewEmptyCoinsMap()
	ins := transaction.GetIns()
	for i, in := range ins {
		coin := coinMap.FetchCoin(in.PreviousOutPoint)
		if coin == nil || coin.IsSpent() {
			log.Debug("TxErrNoPreviousOut")
			return errcode.New(errcode.TxErrNoPreviousOut)
		}
		prevPubKey := coin.GetScriptPubKey()
		var scriptSig *script.Script
		var sigData [][]byte
		var scriptType int
		if hashType&(^(uint32(crypto.SigHashAnyoneCanpay)|crypto.SigHashForkID)) != crypto.SigHashSingle ||
			i < transaction.GetOutsCount() {
			sigData, scriptType, err = transaction.SignStep(redeemScripts, keys, hashType, prevPubKey,
				i, coin.GetAmount())
			if err != nil {
				return err
			}
			// get signatures and redeemscript
			if scriptType == script.ScriptHash {
				redeemScriptPubKey := script.NewScriptRaw(sigData[0])
				var redeemScriptType int
				sigData, redeemScriptType, err = transaction.SignStep(redeemScripts, keys, hashType,
					redeemScriptPubKey, i, coin.GetAmount())
				if err != nil {
					return err
				}
				if redeemScriptType == script.ScriptHash {
					log.Debug("TxErrSignRawTransaction")
					return errcode.New(errcode.TxErrSignRawTransaction)
				}
				sigData = append(sigData, redeemScriptPubKey.GetData())
			}
		}
		scriptSig = script.NewEmptyScript()
		scriptSig.PushMultData(sigData)
		err = lscript.VerifyScript(transaction, scriptSig, prevPubKey, i, coin.GetAmount(),
			uint32(script.StandardScriptVerifyFlags), lscript.NewScriptRealChecker())
		if err != nil {
			return err
		}
		scriptSig, err = combineSignature(transaction, prevPubKey, scriptSig, transaction.GetIns()[i].GetScriptSig(),
			i, coin.GetAmount(), uint32(script.StandardScriptVerifyFlags), lscript.NewScriptRealChecker())
		if err != nil {
			return err
		}
		err = transaction.UpdateInScript(i, scriptSig)
		if err != nil {
			return err
		}
		err = lscript.VerifyScript(transaction, scriptSig, prevPubKey, i, coin.GetAmount(),
			uint32(script.StandardScriptVerifyFlags), lscript.NewScriptRealChecker())
		if err != nil {
			return err
		}
	}
	return
}

func combineSignature(transaction *tx.Tx, prevPubKey *script.Script, scriptSig *script.Script,
	txOldScriptSig *script.Script, nIn int, money amount.Amount, flags uint32,
	scriptChecker lscript.Checker) (*script.Script, error) {
	pubKeyType, pubKeys, err := prevPubKey.CheckScriptPubKeyStandard()
	if err != nil {
		return nil, err
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
		sigData := make([][]byte, 0, len(scriptSig.ParsedOpCodes)-1)
		okSigs := make(map[string][]byte, len(scriptSig.ParsedOpCodes)-1)
		var parsedOpCodes = scriptSig.ParsedOpCodes[:]
		parsedOpCodes = append(parsedOpCodes, txOldScriptSig.ParsedOpCodes...)
		for _, opCode := range parsedOpCodes {
			for _, pubKey := range pubKeys[1 : len(pubKeys)-2] {
				if okSigs[string(pubKey)] != nil {
					continue
				}
				ok, err := scriptChecker.CheckSig(transaction, opCode.Data, pubKey, prevPubKey, nIn, money, flags)
				if err != nil {
					return nil, err
				}
				if ok {
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
		for sigN < sigsRequired {
			data := make([]byte, 0, 1)
			data = append(data, byte(opcodes.OP_0))
			sigData = append(sigData, data)
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
		scriptResult, err := combineSignature(transaction, redeemScript, scriptSig,
			txOldScriptSig, nIn, money, flags, scriptChecker)
		if err != nil {
			return nil, err
		}
		scriptResult.PushSingleData(redeemScript.GetData())
		return scriptResult, nil
	}
	log.Debug("TxErrPubKeyType")
	return nil, errcode.New(errcode.TxErrPubKeyType)
}
