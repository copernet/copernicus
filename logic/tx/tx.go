package tx

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"

	"bytes"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/opcodes"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/model/undo"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/util/amount"
	"strconv"
)

// transaction service will use this func to check transaction before accepting to mempool
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
		return err
	}

	// is mempool already have it? conflict tx with mempool
	if mempool.GetInstance().FindTx(transaction.GetHash()) != nil {
		log.Debug("tx already known in mempool")
		return errcode.New(errcode.TxErrRejectAlreadyKnown)
	}

	// check preout already spent
	ins := transaction.GetIns()
	for _, e := range ins {
		if mempool.GetInstance().HasSpentOut(e.PreviousOutPoint) {
			log.Debug("tx ins alread spent out in mempool")
			return errcode.New(errcode.TxErrRejectConflict)
		}
	}

	// check outpoint alread exist
	exist := areOutputsAlreadExist(transaction)
	if exist {
		log.Debug("tx already known in utxo")
		return errcode.New(errcode.TxErrRejectAlreadyKnown)
	}

	// check inputs are avaliable
	tempCoinsMap := utxo.NewEmptyCoinsMap()
	err = areInputsAvailable(transaction, tempCoinsMap)
	if err != nil {
		return err
	}

	// CLTV(CheckLockTimeVerify)
	lp := CalculateSequenceLocks(transaction, uint32(tx.StandardLockTimeVerifyFlags))
	// Only accept BIP68 sequence locked transactions that can be mined
	// in the next block; we don't want our mempool filled up with
	// transactions that can't be mined yet. Must keep pool.cs for this
	// unless we change CheckSequenceLocks to take a CoinsViewCache
	// instead of create its own.
	if !CheckSequenceLocks(lp) {
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
	var scriptVerifyFlags uint32 = uint32(script.StandardScriptVerifyFlags)
	if !chainparams.ActiveNetParams.RequireStandard {
		configVerifyFlags, err := strconv.Atoi(conf.Cfg.Script.PromiscuousMempoolFlags)
		if err != nil {
			panic("config PromiscuousMempoolFlags err")
		}
		scriptVerifyFlags = uint32(configVerifyFlags)
	}
	err = checkInputs(transaction, tempCoinsMap, scriptVerifyFlags)
	if err != nil {
		return err
	}
	tip := chain.GetInstance().Tip()
	var currentBlockScriptVerifyFlags uint32 = chain.GetInstance().GetBlockScriptFlags(tip)
	err = checkInputs(transaction, tempCoinsMap, currentBlockScriptVerifyFlags)
	if err != nil {
		if ((^scriptVerifyFlags) & currentBlockScriptVerifyFlags) == 0 {
			return errcode.New(errcode.ScriptCheckInputsBug)
		}
		err = checkInputs(transaction, tempCoinsMap, uint32(script.MandatoryScriptVerifyFlags))
		if err != nil {
			return err
		}
	}

	return nil
}

// block service use these 3 func to check transactions or to apply transaction while connecting block to active chain
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
	outPointSet := make(map[*outpoint.OutPoint]bool)
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

func CheckBlockContextureTransactions(txs []*tx.Tx, blockHeight int32, blockLockTime int64) error {
	txsLen := len(txs)
	if txsLen == 0 {
		return errcode.New(errcode.TxErrRejectInvalid)
	}
	err := checkBlockContextureCoinBaseTransaction(txs[0], blockHeight)
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

func ApplyBlockTransactions(txs []*tx.Tx, bip30Enable bool, scriptCheckFlags uint32, needCheckScript bool,
	blockSubSidy amount.Amount, blockHeight int32) (coinMap *utxo.CoinsMap, bundo *undo.BlockUndo, err error) {
	// make view
	coinsMap := utxo.NewEmptyCoinsMap()
	utxo := utxo.GetUtxoCacheInstance()
	sigOpsCount := 0
	var fees amount.Amount = 0
	bundo = undo.NewBlockUndo(0)
	txUndoList := make([]*undo.TxUndo, 0, len(txs)-1)
	//updateCoins
	for _, transaction := range txs {
		//check duplicate out
		if bip30Enable {
			outs := transaction.GetOuts()
			for i, _ := range outs {
				coin := utxo.GetCoin(outpoint.NewOutPoint(transaction.GetHash(), uint32(i)))
				if coin != nil {
					return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
				}
			}
		}
		var valueIn amount.Amount = 0
		if !transaction.IsCoinBase() {
			ins := transaction.GetIns()
			for _, in := range ins {
				coin := coinsMap.FetchCoin(in.PreviousOutPoint)
				if coin == nil || coin.IsSpent() {
					return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
				}
				valueIn += coin.GetAmount()
			}
			lp := CalculateSequenceLocks(transaction, scriptCheckFlags)
			if !CheckSequenceLocks(lp) {
				return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
			}
		}
		//check sigops
		sigsCount, err := GetTransactionSigOpCount(transaction, scriptCheckFlags, coinMap)
		if err != nil {
			return nil, nil, err
		}
		sigOpsCount += sigsCount
		if sigOpsCount > tx.MaxTxSigOpsCounts {
			return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
		}
		if transaction.IsCoinBase() {
			UpdateCoins(transaction, coinsMap, nil, blockHeight)
			continue
		}
		if !transaction.IsCoinBase() {
			fees += valueIn - transaction.GetValueOut()
			if needCheckScript {
				//check inputs
				err := checkInputs(transaction, coinsMap, scriptCheckFlags)
				if err != nil {
					return nil, nil, err
				}
			}
		}
		//update temp coinsMap
		txundo := undo.NewTxUndo()
		UpdateCoins(transaction, coinsMap, txundo, blockHeight)
		txUndoList = append(txUndoList, txundo)
	}
	bundo.SetTxUndo(txUndoList)
	//check blockReward
	if fees+blockSubSidy > txs[0].GetValueOut() {
		return nil, nil, errcode.New(errcode.TxErrRejectInvalid)
	}
	return coinsMap, bundo, nil
}

// check coinbase with height
func checkBlockContextureCoinBaseTransaction(tx *tx.Tx, blockHeight int32) error {
	// Enforce rule that the coinbase starts with serialized block height
	if blockHeight > chainparams.ActiveNetParams.BIP34Height {
		heightNumb := script.NewScriptNum(int64(blockHeight))
		coinBaseScriptSig := tx.GetIns()[0].GetScriptSig()
		heightData := heightNumb.Serialize()
		if coinBaseScriptSig.Size() < len(heightData) {
			return errcode.New(errcode.TxErrRejectInvalid)
		}
		scriptData := coinBaseScriptSig.GetData()[:len(heightData)-1]
		if !bytes.Equal(scriptData, heightData) {
			return errcode.New(errcode.TxErrRejectInvalid)
		}
	}
	return nil
}

func ContextualCheckTransaction(transaction *tx.Tx, nBlockHeight int32, nLockTimeCutoff int64) error {
	if !transaction.IsFinal(nBlockHeight, nLockTimeCutoff) {
		return errcode.New(errcode.TxErrNotFinal)
	}

	if chainparams.IsUAHFEnabled(nBlockHeight) && nBlockHeight <= chainparams.ActiveNetParams.AntiReplayOpReturnSunsetHeight {
		if transaction.IsCommitment(chainparams.ActiveNetParams.AntiReplayOpReturnCommitment) {
			return errcode.New(errcode.TxErrTxCommitment)
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
	for i, _ := range outs {
		coin := utxo.GetCoin(outpoint.NewOutPoint(transaction.GetHash(), uint32(i)))
		if coin != nil {
			return true
		}
	}
	return false
}

func areInputsAvailable(transaction *tx.Tx, coinMap *utxo.CoinsMap) error {
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
			return errcode.New(errcode.TxErrNoPreviousOut)
		}
		if coin.IsSpent() {
			log.Debug("inpute coin is already spent out")
			return errcode.New(errcode.TxErrInputsNotAvailable)
		}
		coinMap.AddCoin(e.PreviousOutPoint, coin)
	}

	return nil
}

func GetTransactionSigOpCount(transaction *tx.Tx, flags uint32, coinMap *utxo.CoinsMap) (int, error) {
	sigOpsCount := transaction.GetSigOpCountWithoutP2SH()
	if transaction.IsCoinBase() || flags&script.ScriptVerifyP2SH == script.ScriptVerifyP2SH {
		return sigOpsCount, nil
	}
	sigOps, err := GetSigOpCountWithP2SH(transaction, coinMap)
	if err != nil {
		return 0, err
	}
	return sigOpsCount + sigOps, nil
}

// starting BIP16(Apr 1 2012), we should check p2sh
func GetSigOpCountWithP2SH(transaction *tx.Tx, coinMap *utxo.CoinsMap) (int, error) {
	n := transaction.GetSigOpCountWithoutP2SH()
	if transaction.IsCoinBase() {
		return n, nil
	}

	ins := transaction.GetIns()
	for _, e := range ins {
		coin := coinMap.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			return 0, errcode.New(errcode.TxErrNoPreviousOut)
		}
		scriptPubKey := coin.GetScriptPubKey()
		if !scriptPubKey.IsPayToScriptHash() {
			sigsCount := scriptPubKey.GetSigOpCount(true)
			n += sigsCount
		} else {
			sigsCount := e.GetScriptSig().GetP2SHSigOpCount()
			n += sigsCount
		}
	}

	return n, nil
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
	var nLockTimeCutoff int64 = 0
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
	for _, e := range ins {
		coin := coinsMap.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			log.Debug("bug! tx input cann't find coin in temp coinsmap")
			errcode.New(errcode.TxErrRejectNonstandard)
		}
		txOut := coin.GetTxOut()
		pubKeyType, err := txOut.GetPubKeyType()
		if err != nil {
			return err
		}
		if pubKeyType == script.ScriptHash {
			scriptSig := e.GetScriptSig()
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
	bestBlockHash := utxo.GetUtxoCacheInstance().GetBestBlock()
	spendHeight := chain.GetInstance().GetSpendHeight(&bestBlockHash)

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
		err := verifyScript(tx, scriptSig, scriptPubKey, i, coin.GetAmount(), flags)
		if err != nil {
			if (flags & uint32(script.StandardNotMandatoryVerifyFlags)) == uint32(script.StandardNotMandatoryVerifyFlags) {
				err = verifyScript(tx, scriptSig, scriptPubKey, i, coin.GetAmount(), flags)
			}
			if err == nil {
				return errcode.New(errcode.TxErrRejectNonstandard)
			}
			return errcode.New(errcode.TxErrRejectInvalid)
		}
	}
	return nil
}

func verifyScript(transaction *tx.Tx, scriptSig *script.Script, scriptPubKey *script.Script,
	nIn int, value amount.Amount, flags uint32) error {
	if flags&script.ScriptEnableSigHashForkId == script.ScriptEnableSigHashForkId {
		flags |= script.ScriptVerifyStrictEnc
	}
	if flags&script.ScriptVerifySigPushOnly == script.ScriptVerifySigPushOnly && !scriptSig.IsPushOnly() {
		return errcode.New(errcode.ScriptErrSigPushOnly)
	}
	stack := util.NewStack()
	err := evalScript(stack, scriptSig, transaction, nIn, value, flags)
	if err != nil {
		return err
	}
	stackCopy := stack.Copy()
	err = evalScript(stack, scriptPubKey, transaction, nIn, value, flags)
	if err != nil {
		return err
	}
	if stack.Empty() {
		return errcode.New(errcode.ScriptErrEvalFalse)
	}
	if stack.Top(-1).(bool) == false {
		return errcode.New(errcode.ScriptErrEvalFalse)
	}
	if flags&script.ScriptVerifyP2SH == script.ScriptVerifyP2SH && scriptPubKey.IsPayToScriptHash() {
		if !scriptSig.IsPushOnly() {
			return errcode.New(errcode.ScriptErrSigPushOnly)
		}
		util.Swap(stack, stackCopy)
		topBytes := stack.Top(-1)
		stack.Pop()
		scriptPubKey2 := script.NewScriptRaw(topBytes.([]byte))
		err = evalScript(stack, scriptPubKey2, transaction, nIn, value, flags)
		if err != nil {
			return err
		}
		if stack.Empty() {
			return errcode.New(errcode.ScriptErrEvalFalse)
		}
		if !stack.Top(-1).(bool) {
			return errcode.New(errcode.ScriptErrEvalFalse)
		}
	}

	// The CLEANSTACK check is only performed after potential P2SH evaluation,
	// as the non-P2SH evaluation of a P2SH script will obviously not result in
	// a clean stack (the P2SH inputs remain). The same holds for witness
	// evaluation.
	if (flags & script.ScriptVerifyCleanStack) != 0 {
		// Disallow CLEANSTACK without P2SH, as otherwise a switch
		// CLEANSTACK->P2SH+CLEANSTACK would be possible, which is not a
		// softfork (and P2SH should be one).
		if flags&script.ScriptVerifyP2SH != 0 {
			panic("")
		}
		if stack.Size() != 1 {
			return errcode.New(errcode.ScriptErrCleanStack)
		}
	}
	return nil
}

func evalScript(stack *util.Stack, s *script.Script, transaction *tx.Tx, nIn int,
	money amount.Amount, flags uint32) error {
	nOpCount := 0

	bnZero := script.ScriptNum{0}
	bnOne := script.ScriptNum{1}
	bnFalse := script.ScriptNum{0}
	bnTrue := script.ScriptNum{1}

	beginCodeHash := 0
	var fRequireMinimal bool
	if flags&script.ScriptVerifyMinmalData == script.ScriptVerifyMinmalData {
		fRequireMinimal = true
	} else {
		fRequireMinimal = false
	}

	fExec := false
	stackExec := util.NewStack()
	stackAlt := util.NewStack()
	for i, e := range s.ParsedOpCodes {
		if stackExec.CountBool(false) == 0 {
			fExec = true
		}

		if len(e.Data) > script.MaxScriptElementSize {
			return errcode.New(errcode.ScriptErrPushSize)
		}
		nOpCount++
		// Note how OP_RESERVED does not count towards the opCode limit.
		if e.OpValue > opcodes.OP_16 && nOpCount > script.MaxOpsPerScript {
			return errcode.New(errcode.ScriptErrOpCount)
		}

		if e.OpValue == opcodes.OP_CAT || e.OpValue == opcodes.OP_SUBSTR || e.OpValue == opcodes.OP_LEFT ||
			e.OpValue == opcodes.OP_RIGHT || e.OpValue == opcodes.OP_INVERT || e.OpValue == opcodes.OP_AND ||
			e.OpValue == opcodes.OP_OR || e.OpValue == opcodes.OP_XOR || e.OpValue == opcodes.OP_2MUL ||
			e.OpValue == opcodes.OP_2DIV || e.OpValue == opcodes.OP_MUL || e.OpValue == opcodes.OP_DIV ||
			e.OpValue == opcodes.OP_MOD || e.OpValue == opcodes.OP_LSHIFT ||
			e.OpValue == opcodes.OP_RSHIFT {
			// Disabled opcodes.
			return errcode.New(errcode.ScriptErrDisabledOpCode)
		}

		if fExec && 0 <= e.OpValue && e.OpValue <= opcodes.OP_PUSHDATA4 {
			if fRequireMinimal && !e.CheckMinimalDataPush() {
				return errcode.New(errcode.ScriptErrMinimalData)
			}
			stack.Push(e.Data)
		} else if fExec || (opcodes.OP_IF <= e.OpValue && e.OpValue <= opcodes.OP_ENDIF) {
			switch e.OpValue {
			// Push value
			case opcodes.OP_1NEGATE:
			case opcodes.OP_1:
			case opcodes.OP_2:
			case opcodes.OP_3:
			case opcodes.OP_4:
			case opcodes.OP_5:
			case opcodes.OP_6:
			case opcodes.OP_7:
			case opcodes.OP_8:
			case opcodes.OP_9:
			case opcodes.OP_10:
			case opcodes.OP_11:
			case opcodes.OP_12:
			case opcodes.OP_13:
			case opcodes.OP_14:
			case opcodes.OP_15:
			case opcodes.OP_16:
				{
					// ( -- value)
					bn := script.NewScriptNum(int64(e.OpValue) - int64(opcodes.OP_1-1))
					stack.Push(bn.Serialize())
					// The result of these opcodes should always be the
					// minimal way to push the data they push, so no need
					// for a CheckMinimalPush here.
					break
				}
				//
				// Control
				//
			case opcodes.OP_NOP:
				break
			case opcodes.OP_CHECKLOCKTIMEVERIFY:
				{
					if flags&script.ScriptVerifyCheckLockTimeVerify == script.ScriptVerifyCheckLockTimeVerify {
						// not enabled; treat as a NOP2
						if flags&script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
							return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)
						}
						break
					}
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					// Note that elsewhere numeric opcodes are limited to
					// operands in the range -2**31+1 to 2**31-1, however it
					// is legal for opcodes to produce results exceeding
					// that range. This limitation is implemented by
					// CScriptNum's default 4-byte limit.
					//
					// If we kept to that limit we'd have a year 2038
					// problem, even though the nLockTime field in
					// transactions themselves is uint32 which only becomes
					// meaningless after the year 2106.
					//
					// Thus as a special case we tell CScriptNum to accept
					// up to 5-byte bignums, which are good until 2**39-1,
					// well beyond the 2**32-1 limit of the nLockTime field
					// itself.
					topBytes := stack.Top(-1)
					if topBytes == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//nLocktime, err := script.GetScriptNum(topBytes.([]byte), fRequireMinimal, 5)
					nLocktime, err := script.GetScriptNum(topBytes.([]byte), true, 5)
					if err != nil {
						return err
					}
					// In the rare event that the argument may be < 0 due to
					// some arithmetic being done first, you can always use
					// 0 MAX CHECKLOCKTIMEVERIFY.
					if nLocktime.Value < 0 {
						return errcode.New(errcode.ScriptErrNegativeLockTime)
					}
					// Actually compare the specified lock time with the
					// transaction.
					if !checkLockTime(nLocktime.Value, int64(transaction.GetLockTime()), transaction.GetIns()[nIn].Sequence) {
						return errcode.New(errcode.ScriptErrUnsatisfiedLockTime)
					}
					break
				}
			case opcodes.OP_CHECKSEQUENCEVERIFY:
				{
					if flags&script.ScriptVerifyCheckSequenceVerify == script.ScriptVerifyCheckSequenceVerify {
						// not enabled; treat as a NOP3
						if flags&script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
							return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)

						}
						break
					}
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					// nSequence, like nLockTime, is a 32-bit unsigned
					// integer field. See the comment in checkLockTimeVerify
					// regarding 5-byte numeric operands.
					topBytes := stack.Top(-1)
					if topBytes == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					nSequence, err := script.GetScriptNum(topBytes.([]byte), fRequireMinimal, 5)
					//nSequence, err := script.GetScriptNum(topBytes.([]byte), true, 5)
					if err != nil {
						return err
					}
					// In the rare event that the argument may be < 0 due to
					// some arithmetic being done first, you can always use
					// 0 MAX checkSequenceVerify.
					if nSequence.Value < 0 {
						return errcode.New(errcode.ScriptErrNegativeLockTime)
					}

					// To provide for future soft-fork extensibility, if the
					// operand has the disabled lock-time flag set,
					// checkSequenceVerify behaves as a NOP.
					if nSequence.Value&script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
						break
					}
					if !checkSequence(nSequence.Value, int64(transaction.GetIns()[nIn].Sequence), uint32(transaction.GetVersion())) {
						return errcode.New(errcode.ScriptErrUnsatisfiedLockTime)
					}
					break
				}
			case opcodes.OP_NOP1:
				fallthrough
			case opcodes.OP_NOP4:
				fallthrough
			case opcodes.OP_NOP5:
				fallthrough
			case opcodes.OP_NOP6:
				fallthrough
			case opcodes.OP_NOP7:
				fallthrough
			case opcodes.OP_NOP8:
				fallthrough
			case opcodes.OP_NOP9:
				fallthrough
			case opcodes.OP_NOP10:
				{
					if flags&script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
						return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)
					}
					break
				}
			case opcodes.OP_IF:
				fallthrough
			case opcodes.OP_NOTIF:
				{
					// <expression> if [statements] [else [statements]]
					// endif
					fValue := false
					if fExec {
						if stack.Size() < 1 {
							return errcode.New(errcode.ScriptErrUnbalancedConditional)
						}
						vch := stack.Top(-1)
						if vch == nil {
							return errcode.New(errcode.ScriptErrUnbalancedConditional)
						}
						vchBytes := vch.([]byte)
						if flags&script.ScriptVerifyMinimalIf == script.ScriptVerifyMinimalIf {
							if len(vchBytes) > 1 {
								return errcode.New(errcode.ScriptErrMinimalIf)
							}
							if len(vchBytes) == 1 && vchBytes[0] != 1 {
								return errcode.New(errcode.ScriptErrMinimalIf)
							}
						}
						fValue = script.BytesToBool(vchBytes)
						if e.OpValue == opcodes.OP_NOTIF {
							fValue = !fValue
						}
						stack.Pop()
					}
					stackExec.Push(fValue)
					break
				}
			case opcodes.OP_ELSE:
				{
					if stackExec.Empty() {
						return errcode.New(errcode.ScriptErrUnbalancedConditional)
					}
					vfBack := !stackExec.Top(-1).(bool)
					if stackExec.SetTop(-1, vfBack) == false {
						return errcode.New(errcode.ScriptErrUnbalancedConditional)
					}
					break
				}
			case opcodes.OP_ENDIF:
				{
					if stackExec.Empty() {
						return errcode.New(errcode.ScriptErrUnbalancedConditional)
					}
					stackExec.Pop()
					break
				}
			case opcodes.OP_VERIFY:
				{
					// (true -- ) or
					// (false -- false) and return
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchBytes := vch.([]byte)
					fValue := script.BytesToBool(vchBytes)
					if fValue {
						stack.Pop()
					} else {
						return errcode.New(errcode.ScriptErrVerify)
					}
					break
				}
			case opcodes.OP_RETURN:
				{
					return errcode.New(errcode.ScriptErrOpReturn)
				}
				//
				// Stack ops
				//
			case opcodes.OP_TOALTSTACK:
				{
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stackAlt.Push(vch)
					stack.Pop()
					break
				}

			case opcodes.OP_FROMALTSTACK:
				{
					if stackAlt.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
					}
					stack.Push(stackAlt.Top(-1))
					stackAlt.Pop()
					break
				}
			case opcodes.OP_2DROP:
				{
					// (x1 x2 -- )
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Pop()
					stack.Pop()
					break
				}
			case opcodes.OP_2DUP:
				{
					// (x1 x2 -- x1 x2 x1 x2)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
					}
					vch1 := stack.Top(-2)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
					}
					vch2 := stack.Top(-1)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
					}
					stack.Push(vch1)
					stack.Push(vch2)
					break
				}
			case opcodes.OP_3DUP:
				{
					// (x1 x2 x3 -- x1 x2 x3 x1 x2 x3)
					if stack.Size() < 3 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-3)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-2)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch3 := stack.Top(-1)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch1)
					stack.Push(vch2)
					stack.Push(vch3)
					break
				}
			case opcodes.OP_2OVER:
				{
					// (x1 x2 x3 x4 -- x1 x2 x3 x4 x1 x2)
					if stack.Size() < 4 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-4)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-3)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch1)
					stack.Push(vch2)
					break
				}
			case opcodes.OP_2ROT:
				{
					// (x1 x2 x3 x4 x5 x6 -- x3 x4 x5 x6 x1 x2)
					if stack.Size() < 6 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-6)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-5)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Erase(stack.Size()-6, stack.Size()-4)
					stack.Push(vch1)
					stack.Push(vch2)
					break
				}
			case opcodes.OP_2SWAP:
				{
					// (x1 x2 x3 x4 -- x3 x4 x1 x2)
					if stack.Size() < 4 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Swap(stack.Size()-4, stack.Size()-2)
					stack.Swap(stack.Size()-3, stack.Size()-1)
					break
				}
			case opcodes.OP_IFDUP:
				{
					// (x - 0 | x x)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchBytes := vch.([]byte)
					if script.BytesToBool(vchBytes) {
						stack.Push(vch)
					}
					break
				}
			case opcodes.OP_DEPTH:
				{
					// -- stacksize
					bn := script.NewScriptNum(int64(stack.Size()))
					stack.Push(bn.Serialize())
					break
				}
			case opcodes.OP_DROP:
				{
					// (x -- )
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Pop()
					break
				}
			case opcodes.OP_DUP:
				{
					// (x -- x x)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch)
					break
				}
			case opcodes.OP_NIP:
				{
					// (x1 x2 -- x2)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.RemoveAt(stack.Size() - 2)
					break
				}
			case opcodes.OP_OVER:
				{
					// (x1 x2 -- x1 x2 x1)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-2)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch)
					break
				}
			case opcodes.OP_PICK:
				fallthrough
			case opcodes.OP_ROLL:
				{
					// (xn ... x2 x1 x0 n - xn ... x2 x1 x0 xn)
					// (xn ... x2 x1 x0 n - ... x2 x1 x0 xn)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					scriptNum, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					//scriptNum, err := script.GetScriptNum(vch.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err

					}
					n := scriptNum.ToInt32()
					stack.Pop()
					if n < 0 || n >= int32(stack.Size()) {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchn := stack.Top(int(-n - 1))
					if vchn == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					if e.OpValue == opcodes.OP_ROLL {
						stack.RemoveAt(stack.Size() - int(n) - 1)
					}
					stack.Push(vchn)
					break
				}
			case opcodes.OP_ROT:
				{
					// (x1 x2 x3 -- x2 x3 x1)
					//  x2 x1 x3  after first swap
					//  x2 x3 x1  after second swap
					if stack.Size() < 3 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Swap(stack.Size()-3, stack.Size()-2)
					stack.Swap(stack.Size()-2, stack.Size()-1)
					break
				}
			case opcodes.OP_SWAP:
				{
					// (x1 x2 -- x2 x1)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Swap(stack.Size()-2, stack.Size()-1)
					break
				}
			case opcodes.OP_TUCK:
				{
					// (x1 x2 -- x2 x1 x2)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					if !stack.Insert(stack.Size()-2, vch) {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					break
				}
			case opcodes.OP_SIZE:
				{
					// (in -- in size)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					size := len(vch.([]byte))
					bn := script.NewScriptNum(int64(size))
					stack.Push(bn.Serialize())
					break
				}
				//
				// Bitwise logic
				//
			case opcodes.OP_EQUAL:
				fallthrough
			case opcodes.OP_EQUALVERIFY:
				// case opcodes.OP_NOTEQUAL: // use opcodes.OP_NUMNOTEQUAL
				{
					// (x1 x2 - bool)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-2)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-1)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					fEqual := bytes.Equal(vch1.([]byte), vch2.([]byte))
					//fEqual := reflect.DeepEqual(vch1, vch2)
					// opcodes.OP_NOTEQUAL is disabled because it would be too
					// easy to say something like n != 1 and have some
					// wiseguy pass in 1 with extra zero bytes after it
					// (numerically, 0x01 == 0x0001 == 0x000001)
					// if (opcode == opcodes.OP_NOTEQUAL)
					//    fEqual = !fEqual;
					stack.Pop()
					stack.Pop()
					if fEqual {
						stack.Push(bnTrue.Value)
					} else {
						stack.Push(bnFalse.Value)
					}
					if e.OpValue == opcodes.OP_EQUALVERIFY {
						if fEqual {
							stack.Pop()
						} else {
							return errcode.New(errcode.ScriptErrEqualVerify)
						}
					}
					break

				}
				//Numeric
			case opcodes.OP_1ADD:
				fallthrough
			case opcodes.OP_1SUB:
				fallthrough
			case opcodes.OP_NEGATE:
				fallthrough
			case opcodes.OP_ABS:
				fallthrough
			case opcodes.OP_NOT:
				fallthrough
			case opcodes.OP_0NOTEQUAL:
				{
					// (in -- out)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					bn, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					//bn, err := script.GetScriptNum(vch.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					switch e.OpValue {
					case opcodes.OP_1ADD:
						bn.Value += bnOne.Value
						break
					case opcodes.OP_1SUB:
						bn.Value -= bnOne.Value
						break
					case opcodes.OP_NEGATE:
						bn.Value = -bn.Value
						break
					case opcodes.OP_ABS:
						if bn.Value < 0 {
							bn.Value = -bn.Value
						}
						break
					case opcodes.OP_NOT:
						if bn.Value == bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_0NOTEQUAL:
						if bn.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					default:
						return errcode.New(errcode.ScriptErrInvalidOpCode)
					}
					stack.Pop()
					stack.Push(bn.Serialize())
					break
				}
			case opcodes.OP_ADD:
				fallthrough
			case opcodes.OP_SUB:
				fallthrough
			case opcodes.OP_BOOLAND:
				fallthrough
			case opcodes.OP_BOOLOR:
				fallthrough
			case opcodes.OP_NUMEQUAL:
				fallthrough
			case opcodes.OP_NUMEQUALVERIFY:
				fallthrough
			case opcodes.OP_NUMNOTEQUAL:
				fallthrough
			case opcodes.OP_LESSTHAN:
				fallthrough
			case opcodes.OP_GREATERTHAN:
				fallthrough
			case opcodes.OP_LESSTHANOREQUAL:
				fallthrough
			case opcodes.OP_GREATERTHANOREQUAL:
				fallthrough
			case opcodes.OP_MIN:
				fallthrough
			case opcodes.OP_MAX:
				{
					// (x1 x2 -- out)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-2)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-1)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					bn1, err := script.GetScriptNum(vch1.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					//bn1, err := script.GetScriptNum(vch1.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					bn2, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					//bn2, err := script.GetScriptNum(vch2.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					bn := script.NewScriptNum(0)
					switch e.OpValue {
					case opcodes.OP_ADD:
						bn.Value = bn1.Value + bn2.Value
						break
					case opcodes.OP_SUB:
						bn.Value = bn1.Value - bn2.Value
						break
					case opcodes.OP_BOOLAND:
						if bn1.Value != bnZero.Value && bn2.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_BOOLOR:
						if bn1.Value != bnZero.Value || bn2.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_NUMEQUAL:
						if bn1.Value == bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_NUMEQUALVERIFY:
						if bn1.Value == bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_NUMNOTEQUAL:
						if bn1.Value != bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_LESSTHAN:
						if bn1.Value < bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_GREATERTHAN:
						if bn1.Value > bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_LESSTHANOREQUAL:
						if bn1.Value <= bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_GREATERTHANOREQUAL:
						if bn1.Value >= bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_MIN:
						if bn1.Value < bn2.Value {
							bn = bn1
						} else {
							bn = bn2
						}
						break
					case opcodes.OP_MAX:
						if bn1.Value > bn2.Value {
							bn = bn1
						} else {
							bn = bn2
						}
						break
					default:
						return errcode.New(errcode.ScriptErrInvalidOpCode)
					}
					stack.Pop()
					stack.Pop()
					stack.Push(bn.Serialize())

					if e.OpValue == opcodes.OP_NUMEQUALVERIFY {
						vch := stack.Top(-1)
						if vch == nil {
							return errcode.New(errcode.ScriptErrInvalidStackOperation)
						}
						if script.BytesToBool(vch.([]byte)) {
							stack.Pop()
						} else {
							return errcode.New(errcode.ScriptErrNumEqualVerify)
						}
					}
					break
				}

			case opcodes.OP_WITHIN:
				{
					// (x min max -- out)
					if stack.Size() < 3 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-3)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-2)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch3 := stack.Top(-1)
					if vch3 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					bn1, err := script.GetScriptNum(vch1.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					bn2, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					bn3, err := script.GetScriptNum(vch3.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					var fValue int = 0
					if bn2.Value <= bn1.Value && bn1.Value < bn3.Value {
						fValue = 1
					} else {
						fValue = 0
					}
					stack.Pop()
					stack.Pop()
					stack.Pop()
					stack.Push(fValue)
					break
				}
				// Crypto
			case opcodes.OP_RIPEMD160:
				fallthrough
			case opcodes.OP_SHA1:
				fallthrough
			case opcodes.OP_SHA256:
				fallthrough
			case opcodes.OP_HASH160:
				fallthrough
			case opcodes.OP_HASH256:
				{
					// (in -- GetHash)
					var vchHash []byte
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					switch e.OpValue {
					case opcodes.OP_RIPEMD160:
						vchHash = util.Ripemd160(vch.([]byte))
					case opcodes.OP_SHA1:
						result := util.Sha1(vch.([]byte))
						vchHash = append(vchHash, result[:]...)
					case opcodes.OP_SHA256:
						vchHash = util.Sha256Bytes(vch.([]byte))
					case opcodes.OP_HASH160:
						vchHash = util.Hash160(vch.([]byte))
					case opcodes.OP_HASH256:
						vchHash = util.Sha256Bytes(vch.([]byte))
					}
					stack.Pop()
					stack.Push(vchHash)
					break
				}
			case opcodes.OP_CODESEPARATOR:
				{
					// Hash starts after the code separator
					beginCodeHash = i
					break
				}
			case opcodes.OP_CHECKSIG:
				fallthrough
			case opcodes.OP_CHECKSIGVERIFY:
				{
					// (sig pubkey -- bool)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchSig := stack.Top(-2)
					if vchSig == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchPubkey := stack.Top(-1)
					if vchPubkey == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					vchSigBytes := vchSig.([]byte)
					err := script.CheckSignatureEncoding(vchSigBytes, flags)
					if err != nil {
						return err
					}
					err = script.CheckPubKeyEncoding(vchPubkey.([]byte), flags)
					if err != nil {
						return err
					}
					// signature is DER format, the second byte + 2 indicates the len of signature
					// 0x30 if the first byte that indicates the beginning of signature
					vchSigBytes = vchSigBytes[:vchSigBytes[1]+2]
					// Subset of script starting at the most recent codeSeparator
					scriptCode := script.NewScriptOps(s.ParsedOpCodes[beginCodeHash:])

					// Remove the signature since there is no way for a signature
					// to sign itself.
					scriptCode = scriptCode.RemoveOpcodeByData(vchSigBytes)

					fSuccess, err := CheckSig(transaction, vchSigBytes, vchPubkey.([]byte), scriptCode, nIn, money, flags)
					if err != nil {
						return err
					}

					if !fSuccess &&
						(flags&script.ScriptVerifyNullFail == script.ScriptVerifyNullFail) &&
						len(vchSig.([]byte)) > 0 {
						return errcode.New(errcode.ScriptErrSigNullFail)

					}

					stack.Pop()
					stack.Pop()
					if fSuccess {
						stack.Push(true)
					} else {
						stack.Push(false)
					}
					if e.OpValue == opcodes.OP_CHECKSIGVERIFY {
						if fSuccess {
							stack.Pop()
						} else {
							return errcode.New(errcode.ScriptErrCheckSigVerify)
						}
					}
				}
			case opcodes.OP_CHECKMULTISIG:
				fallthrough
			case opcodes.OP_CHECKMULTISIGVERIFY:
				{
					// ([sig ...] num_of_signatures [pubkey ...]
					// num_of_pubkeys -- bool)
					i := 1
					if stack.Size() < i {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					vch := stack.Top(-i)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					// ScriptSig1 ScriptSig2...ScriptSigM M PubKey1 PubKey2...PubKey N
					pubKeysNum, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					if err != nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					pubKeysCount := pubKeysNum.ToInt32()
					if pubKeysCount < 0 || pubKeysCount > script.MaxOpsPerScript {
						return errcode.New(errcode.ScriptErrOpCount)
					}
					// skip N
					i++
					// PubKey start position
					iPubKey := i
					// iKey2 is the position of last non-signature item in
					// the stack. Top stack item = 1. With
					// ScriptVerifyNullFail, this is used for cleanup if
					// operation fails.
					iKey2 := pubKeysCount + 2
					i += int(pubKeysCount)
					if stack.Size() < i {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					sigsVch := stack.Top(-i)
					if err != nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					nSigsNum, err := script.GetScriptNum(sigsVch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					if err != nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					nSigsCount := nSigsNum.ToInt32()
					if nSigsCount < 0 || nSigsCount > pubKeysCount {
						return errcode.New(errcode.ScriptErrSigCount)
					}
					i++
					/// Sig start position
					iSig := i
					i += int(nSigsCount)
					if stack.Size() < i {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					// Subset of script starting at the most recent codeSeparator
					scriptCode := script.NewScriptOps(s.ParsedOpCodes[beginCodeHash:])

					// Drop the signature in pre-segwit scripts but not segwit scripts
					for k := 0; k < int(nSigsCount); k++ {
						vchSig := stack.Top(-iSig - k)
						if vchSig == nil {
							return errcode.New(errcode.ScriptErrInvalidStackOperation)
						}
						scriptCode = scriptCode.RemoveOpcodeByData(vchSig.([]byte))
					}
					fSuccess := true
					for fSuccess && nSigsCount > 0 {
						vchSig := stack.Top(-iSig)
						if vchSig == nil {
							return errcode.New(errcode.ScriptErrInvalidStackOperation)
						}
						vchPubkey := stack.Top(-iPubKey)
						if vchPubkey == nil {
							return errcode.New(errcode.ScriptErrInvalidStackOperation)
						}
						// Note how this makes the exact order of
						// pubkey/signature evaluation distinguishable by
						// CHECKMULTISIG NOT if the STRICTENC flag is set.
						// See the script_(in)valid tests for details.
						err := script.CheckSignatureEncoding(vchSig.([]byte), flags)
						if err != nil {
							return err
						}
						err = script.CheckPubKeyEncoding(vchPubkey.([]byte), flags)
						if err != nil {
							return err
						}
						fOk, err := CheckSig(transaction, vchSig.([]byte), vchPubkey.([]byte), scriptCode, nIn, money, flags)
						if err != nil {
							return err
						}
						if fOk {
							iSig++
							nSigsCount--
						}
						iPubKey++
						pubKeysCount--
						// If there are more signatures left than keys left,
						// then too many signatures have failed. Exit early,
						// without checking any further signatures.
						if nSigsCount > pubKeysCount {
							fSuccess = false
						}
					}
					// Clean up stack of actual arguments
					for i > 1 {
						// If the operation failed, we require that all
						// signatures must be empty vector
						if !fSuccess && (flags&script.ScriptVerifyNullFail == script.ScriptVerifyNullFail) &&
							iKey2 == 0 && len(stack.Top(-1).([]byte)) > 0 {
							return errcode.New(errcode.ScriptErrSigNullFail)

						}
						if iKey2 > 0 {
							iKey2--
						}
						stack.Pop()
						i--
					}
					// A bug causes CHECKMULTISIG to consume one extra
					// argument whose contents were not checked in any way.
					//
					// Unfortunately this is a potential source of
					// mutability, so optionally verify it is exactly equal
					// to zero prior to removing it from the stack.
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					if flags&script.ScriptVerifyNullDummy == script.ScriptVerifyNullDummy &&
						len(stack.Top(-1).([]byte)) > 0 {
						return errcode.New(errcode.ScriptErrSigNullDummy)

					}
					stack.Pop()
					if fSuccess {
						stack.Push(true)
					} else {
						stack.Push(false)
					}
					if e.OpValue == opcodes.OP_CHECKMULTISIGVERIFY {
						if fSuccess {
							stack.Pop()
						} else {
							return errcode.New(errcode.ScriptErrCheckMultiSigVerify)
						}
					}
					break
				}
			default:
				return errcode.New(errcode.ScriptErrBadOpCode)
			}

			if stack.Size()+stackAlt.Size() > 1000 {
				return errcode.New(errcode.ScriptErrStackSize)
			}
		}
	}

	if !stackExec.Empty() {
		return errcode.New(errcode.ScriptErrUnbalancedConditional)
	}

	return nil
}

func checkLockTime(lockTime int64, txLockTime int64, sequence uint32) bool {
	// There are two kinds of nLockTime: lock-by-blockheight and
	// lock-by-blocktime, distinguished by whether nLockTime <
	// LOCKTIME_THRESHOLD.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nLockTime being tested is the same as the nLockTime in the
	// transaction.
	if !((txLockTime < script.LockTimeThreshold && lockTime < script.LockTimeThreshold) ||
		(txLockTime >= script.LockTimeThreshold && lockTime >= script.LockTimeThreshold)) {
		return false
	}

	// Now that we know we're comparing apples-to-apples, the comparison is a
	// simple numeric one.
	if lockTime > txLockTime {
		return false
	}
	// Finally the nLockTime feature can be disabled and thus
	// checkLockTimeVerify bypassed if every txIN has been finalized by setting
	// nSequence to maxInt. The transaction would be allowed into the
	// blockChain, making the opCode ineffective.
	//
	// Testing if this vin is not final is sufficient to prevent this condition.
	// Alternatively we could test all inputs, but testing just this input
	// minimizes the data required to prove correct checkLockTimeVerify
	// execution.
	if script.SequenceFinal == sequence {
		return false
	}
	return true
}

func checkSequence(sequence int64, txToSequence int64, txVersion uint32) bool {
	// Fail if the transaction's version number is not set high enough to
	// trigger BIP 68 rules.
	if txVersion < 2 {
		return false
	}
	// Sequence numbers with their most significant bit set are not consensus
	// constrained. Testing that the transaction's sequence number do not have
	// this bit set prevents using this property to get around a
	// checkSequenceVerify check.
	if txToSequence&script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
		return false
	}
	// Mask off any bits that do not have consensus-enforced meaning before
	// doing the integer comparisons
	nLockTimeMask := script.SequenceLockTimeTypeFlag | script.SequenceLockTimeMask
	txToSequenceMasked := txToSequence & int64(nLockTimeMask)
	nSequenceMasked := sequence & int64(nLockTimeMask)

	// There are two kinds of nSequence: lock-by-blockHeight and
	// lock-by-blockTime, distinguished by whether nSequenceMasked <
	// CTxIn::SEQUENCE_LOCKTIME_TYPE_FLAG.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nSequenceMasked being tested is the same as the nSequenceMasked in the
	// transaction.
	if !((txToSequenceMasked < script.SequenceLockTimeTypeFlag && nSequenceMasked < script.SequenceLockTimeTypeFlag) ||
		(txToSequenceMasked >= script.SequenceLockTimeTypeFlag && nSequenceMasked >= script.SequenceLockTimeTypeFlag)) {
		return false
	}
	if nSequenceMasked > txToSequenceMasked {
		return false
	}
	return true
}

// caculate lockpoint(all ins' max time or height at which it can be spent) of transaction
func CalculateSequenceLocks(transaction *tx.Tx, flags uint32) (lp *mempool.LockPoints) {
	lp = mempool.NewLockPoints()
	var maxHeight int32 = -1
	var maxTime int64 = -1
	var maxInputHeight int32 = 0
	activeChain := chain.GetInstance()
	utxo := utxo.GetUtxoCacheInstance()
	ins := transaction.GetIns()
	for _, e := range ins {
		coin := utxo.GetCoin(e.PreviousOutPoint)
		if coin == nil {
			coin = mempool.GetInstance().GetCoin(e.PreviousOutPoint)
		}
		if coin == nil {
			return nil
		}
		var coinHeight int32
		var coinTime int64
		if coin.IsMempoolCoin() {
			coinHeight = activeChain.Height() + 1
		} else {
			coinHeight = coin.GetHeight()
		}
		if e.Sequence&script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
			continue
		}
		// Can ignore mempool inputs since we'll fail if they had
		// non-zero locks
		if coinHeight != activeChain.Height()+1 && maxHeight < coinHeight {
			maxInputHeight = coinHeight
		}
		if e.Sequence&script.SequenceLockTimeTypeFlag == script.SequenceLockTimeTypeFlag {
			coinTime = activeChain.GetAncestor(coinHeight - 1).GetMedianTimePast()
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
	lp.MaxInputBlock = activeChain.GetAncestor(maxInputHeight)
	if transaction.GetVersion() < 2 || flags&consensus.LocktimeVerifySequence != consensus.LocktimeVerifySequence {
		lp.Time = -1
		lp.Height = -1
	} else {
		lp.Height = maxHeight
		lp.Time = maxTime
	}

	return
}

func CheckSequenceLocks(lp *mempool.LockPoints) bool {
	activeChain := chain.GetInstance()
	blockTime := activeChain.Tip().GetMedianTimePast()
	if lp.Height >= activeChain.Height()+1 || lp.Time >= blockTime {
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
			return errcode.New(errcode.TxErrInputsNotAvailable)
		}
		if coin.IsCoinBase() {
			if spendHeight-coin.GetHeight() < consensus.CoinbaseMaturity {
				return errcode.New(errcode.TxErrRejectInvalid)
			}
		}
		txOut := coin.GetTxOut()
		if !amount.MoneyRange(txOut.GetValue()) {
			return errcode.New(errcode.TxErrRejectInvalid)
		}
		nValue += txOut.GetValue()
		if amount.MoneyRange(nValue) {
			return errcode.New(errcode.TxErrRejectInvalid)
		}
	}
	if nValue < transaction.GetValueOut() {
		return errcode.New(errcode.TxErrRejectInvalid)
	}

	txFee := nValue - transaction.GetValueOut()
	if !amount.MoneyRange(txFee) {
		return errcode.New(errcode.TxErrRejectInvalid)
	}
	return nil
}

func CheckSig(transaction *tx.Tx, signature []byte, pubKey []byte, scriptCode *script.Script,
	nIn int, money amount.Amount, flags uint32) (bool, error) {
	hashType := signature[len(signature)-1]
	txHash, err := tx.SignatureHash(transaction, scriptCode, uint32(hashType), nIn, money, flags)
	if err != nil {
		return false, err
	}
	fOk := tx.CheckSig(txHash, signature, pubKey)
	return fOk, err
}

// txs, preouts, private key, hash type
func SignRawTransaction(transaction *tx.Tx, redeemScripts map[string]string, keys map[string]*crypto.PrivateKey,
	hashType uint32) (err error) {
	coinMap := utxo.NewEmptyCoinsMap()
	ins := transaction.GetIns()
	for i, in := range ins {
		coin := coinMap.FetchCoin(in.PreviousOutPoint)
		if coin == nil || coin.IsSpent() {
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
				sigData, redeemScriptType, err := transaction.SignStep(redeemScripts, keys, hashType,
					redeemScriptPubKey, i, coin.GetAmount())
				if err != nil {
					return err
				}
				if redeemScriptType == script.ScriptHash {
					return errcode.New(errcode.TxErrSignRawTransaction)
				}
				sigData = append(sigData, redeemScriptPubKey.GetData())
			}
		}
		scriptSig = script.NewEmptyScript()
		scriptSig.PushData(sigData)
		err = verifyScript(transaction, scriptSig, prevPubKey, i, coin.GetAmount(), uint32(script.StandardScriptVerifyFlags))
		if err != nil {
			return err
		}
		scriptSig, err = combineSignature(transaction, prevPubKey, scriptSig, transaction.GetIns()[i].GetScriptSig(),
			i, coin.GetAmount(), uint32(script.StandardScriptVerifyFlags))
		if err != nil {
			return err
		}
		err = transaction.UpdateInScript(i, scriptSig)
		if err != nil {
			return err
		}
		err = verifyScript(transaction, scriptSig, prevPubKey, i, coin.GetAmount(), uint32(script.StandardScriptVerifyFlags))
		if err != nil {
			return err
		}
	}
	return
}

func combineSignature(transaction *tx.Tx, prevPubKey *script.Script, scriptSig *script.Script,
	txOldScriptSig *script.Script, nIn int, money amount.Amount, flags uint32) (*script.Script, error) {
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
		if scriptSig.Size() == 0 {
			return txOldScriptSig, nil
		}
		return scriptSig, nil
	}
	if pubKeyType == script.ScriptMultiSig {
		sigData := make([][]byte, len(scriptSig.ParsedOpCodes)-1)
		okSigs := make(map[string][]byte, len(scriptSig.ParsedOpCodes)-1)
		var parsedOpCodes = scriptSig.ParsedOpCodes[:]
		parsedOpCodes = append(parsedOpCodes, txOldScriptSig.ParsedOpCodes...)
		for _, opCode := range parsedOpCodes {
			for _, pubKey := range pubKeys[1 : len(pubKeys)-2] {
				if okSigs[string(pubKey)] != nil {
					continue
				}
				ok, err := CheckSig(transaction, opCode.Data, pubKey, prevPubKey, nIn, money, flags)
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
			data := make([]byte, 1)
			data = append(data, byte(opcodes.OP_0))
			sigData = append(sigData, data)
			sigN++
		}
		scriptResult := script.NewEmptyScript()
		scriptResult.PushData(sigData)
		return scriptResult, nil
	}
	if pubKeyType == script.ScriptHash {
		if scriptSig.Size() == 0 {
			return txOldScriptSig, nil
		}
		if txOldScriptSig.Size() == 0 {
			return scriptSig, nil
		}
		redeemScript := script.NewScriptRaw(scriptSig.ParsedOpCodes[len(scriptSig.ParsedOpCodes)-1].Data)
		scriptSig = scriptSig.RemoveOpCodeByIndex(len(scriptSig.ParsedOpCodes) - 1)
		txOldScriptSig = txOldScriptSig.RemoveOpCodeByIndex(len(scriptSig.ParsedOpCodes) - 1)
		scriptResult, err := combineSignature(transaction, redeemScript, scriptSig,
			txOldScriptSig, nIn, money, flags)
		if err != nil {
			return nil, err
		}
		return scriptResult, nil
	}
	return nil, errcode.New(errcode.TxErrPubKeyType)
}
