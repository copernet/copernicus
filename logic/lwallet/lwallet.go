// Package lwallet implements logic of the wallet
// It is not a complete wallet and only provides basic testing capabilities for rpc currently
package lwallet

import (
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/model/wallet"
	"github.com/copernet/copernicus/net/server"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/copernet/copernicus/util/cashaddr"
	"github.com/pkg/errors"
	"math"
)

// SpendZeroConfChange TODO: read from config
var SpendZeroConfChange = true

type TxnCoin struct {
	OutPoint *outpoint.OutPoint
	Coin     *utxo.Coin
	IsSafe   bool
}

func IsWalletEnable() bool {
	return wallet.GetInstance().IsEnable()
}

func GetNewAddress(account string, isLegacyAddr bool) (string, error) {
	pubKey, err := wallet.GetInstance().GenerateNewKey()
	if err != nil {
		return "", nil
	}
	pubKeyHash := pubKey.ToHash160()

	var address string
	if isLegacyAddr {
		legacyAddr, err := script.AddressFromHash160(pubKeyHash, script.AddressVerPubKey())
		if err != nil {
			return "", err
		}
		address = legacyAddr.String()
	} else {
		cashAddr, err := cashaddr.NewCashAddressPubKeyHash(pubKeyHash, chain.GetInstance().GetParams())
		if err != nil {
			return "", err
		}
		address = cashAddr.String()
	}

	wallet.GetInstance().SetAddressBook(pubKeyHash, account, "receive")

	return address, nil
}

func GetMiningAddress() (string, error) {
	pubKey, err := wallet.GetInstance().GetReservedKey()
	if err != nil {
		return "", nil
	}

	pubKeyHash := pubKey.ToHash160()
	cashAddr, err := cashaddr.NewCashAddressPubKeyHash(pubKeyHash, chain.GetInstance().GetParams())
	if err != nil {
		return "", err
	}
	return cashAddr.String(), nil
}

func GetKeyPair(pubKeyHash []byte) *crypto.KeyPair {
	return wallet.GetInstance().GetKeyPair(pubKeyHash)
}

func GetKeyPairs(pubKeyHashList [][]byte) []*crypto.KeyPair {
	return wallet.GetInstance().GetKeyPairs(pubKeyHashList)
}

func CheckFinalTx(txn *tx.Tx) bool {
	err := ltx.ContextualCheckTransactionForCurrentBlock(txn, int(tx.StandardLockTimeVerifyFlags))
	return err == nil
}

func AvailableCoins(onlySafe bool, includeZeroValue bool) []*TxnCoin {
	coins := make([]*TxnCoin, 0)
	walletTxns := wallet.GetInstance().GetWalletTxns()
	for _, walletTx := range walletTxns {
		txn := walletTx.Tx
		if !CheckFinalTx(txn) {
			continue
		}
		txHash := txn.GetHash()
		depth := walletTx.GetDepthInMainChain()

		if txn.IsCoinBase() && depth <= consensus.CoinbaseMaturity {
			continue
		}
		// We should not consider coins which aren't at least in our mempool.
		// It's possible for these to be conflicted via ancestors which we may
		// never be able to detect.
		if depth == 0 && !mempool.GetInstance().IsTransactionInPool(txn) {
			continue
		}

		isSafe := wallet.GetInstance().IsTrusted(walletTx)
		if onlySafe && !isSafe {
			continue
		}

		for index := 0; index < txn.GetOutsCount(); index++ {
			// check coin is unspent
			outPoint := outpoint.NewOutPoint(txHash, uint32(index))
			coin := wallet.GetInstance().GetUnspentCoin(outPoint)
			if coin == nil {
				continue
			}
			// check coin is mine
			if !wallet.IsUnlockable(coin.GetScriptPubKey()) {
				continue
			}
			// check zero value
			if !includeZeroValue && coin.GetAmount() == 0 {
				continue
			}
			coins = append(coins, &TxnCoin{
				OutPoint: outPoint,
				Coin:     coin,
				IsSafe:   isSafe,
			})
		}
	}
	return coins
}

func GetAccountName(keyHash []byte) string {
	return wallet.GetInstance().GetAccountName(keyHash)
}

func GetScript(scriptHash []byte) *script.Script {
	return wallet.GetInstance().GetScript(scriptHash)
}

func AddToWallet(txn *tx.Tx, blockhash util.Hash, extInfo map[string]string) {
	wallet.GetInstance().AddToWallet(txn, blockhash, extInfo)
}
func RemoveFromWallet(txn *tx.Tx) {
	wallet.GetInstance().RemoveFromWallet(txn)
}

func SetFeeRate(feePaid int64, byteSize int64) {
	wallet.GetInstance().SetFeeRate(feePaid, byteSize)
}

func CreateTransaction(recipients []*wallet.Recipient, changePosInOut *int, sign bool) (*tx.Tx,
	amount.Amount, error) {
	if len(recipients) == 0 {
		return nil, 0, errors.New("Transaction must have at least one recipient")
	}

	value := amount.Amount(0)
	changePosRequest := *changePosInOut
	subtractFeeCount := 0
	for _, recipient := range recipients {
		if recipient.Value < 0 {
			return nil, 0, errors.New("Transaction amounts must not be negative")
		}
		value += recipient.Value
		if recipient.SubtractFeeFromAmount {
			subtractFeeCount++
		}
	}

	// Discourage fee sniping.
	//
	// For a large miner the value of the transactions in the best block and the
	// mempool can exceed the cost of deliberately attempting to mine two blocks
	// to orphan the current best block. By setting nLockTime such that only the
	// next block can include the transaction, we discourage this practice as
	// the height restricted and limited blocksize gives miners considering fee
	// sniping fewer options for pulling off this attack.
	//
	// A simple way to think about this is from the wallet's point of view we
	// always want the blockchain to move forward. By setting nLockTime this way
	// we're basically making the statement that we only want this transaction
	// to appear in the next block; we don't want to potentially encourage
	// reorgs by allowing transactions to appear at lower heights than the next
	// block in forks of the best chain.
	//
	// Of course, the subsidy is high enough, and transaction volume low enough,
	// that fee sniping isn't a problem yet, but by implementing a fix now we
	// ensure code won't be written that makes assumptions about nLockTime that
	// preclude a fix later.
	lockTime := uint32(chain.GetInstance().Height())

	// Secondly occasionally randomly pick a nLockTime even further back, so
	// that transactions that are delayed after signing for whatever reason,
	// e.g. high-latency mix networks and some CoinJoin implementations, have
	// better privacy.
	if util.GetRand(10) == 0 {
		lockTime = util.MaxU32(0, lockTime-uint32(util.GetRandInt(100)))
	}

	var selectedCoins []*TxnCoin
	txn := tx.NewTx(lockTime, tx.DefaultVersion)
	coins := AvailableCoins(true, false)
	feeRet := amount.Amount(0)
	dustRelayFee := util.NewFeeRate(conf.Cfg.TxOut.DustRelayFee)

	// Start with no fee and loop until there is enough fee.
	for {
		*changePosInOut = changePosRequest
		txn = tx.NewTx(lockTime, tx.DefaultVersion)
		first := true

		valueToSelect := value
		if subtractFeeCount == 0 {
			valueToSelect += feeRet
		}

		// vouts to the payees
		for _, recipient := range recipients {
			outValue := recipient.Value

			if recipient.SubtractFeeFromAmount {
				// Subtract fee equally from each selected recipient.
				outValue -= feeRet / amount.Amount(subtractFeeCount)

				// First receiver pays the remainder not divisible by output count.
				if first {
					first = false
					outValue -= feeRet % amount.Amount(subtractFeeCount)
				}
			}
			txOut := txout.NewTxOut(outValue, recipient.ScriptPubKey)

			if txOut.IsDust(dustRelayFee) {
				var errMsg string
				if recipient.SubtractFeeFromAmount && feeRet > 0 {
					if txOut.GetValue() < 0 {
						errMsg = "The transaction amount is too small to pay the fee"
					} else {
						errMsg = "The transaction amount is too small to send after the fee has been deducted"
					}
				} else {
					errMsg = "Transaction amount too small"
				}
				return nil, 0, errors.New(errMsg)
			}
			txn.AddTxOut(txOut)
		}

		// Choose coins to use.
		var valueIn amount.Amount
		selectedCoins, valueIn = selectCoins(coins, valueToSelect)
		if selectedCoins == nil {
			return nil, 0, errors.New("Insufficient funds")
		}

		change := valueIn - valueToSelect
		if change > 0 {
			// Fill a vout to ourself.
			// TODO: pass in scriptChange instead of reservekey so change
			// transaction isn't always pay-to-bitcoin-address.

			// Note: We use a new key here to keep it from being obvious
			// which side is the change. The drawback is that by not
			// reusing a previous key, the change may be lost if a
			// backup is restored, if the backup doesn't have the new
			// private key for the change. If we reused the old key, it
			// would be possible to add code to look for and rediscover
			// unknown transactions that were written with keys of ours
			// to recover post-backup change.

			reservedKey, err := wallet.GetInstance().GetReservedKey()
			if err != nil || reservedKey == nil {
				return nil, 0, errors.New("Keypool ran out, please call keypoolrefill first")
			}
			scriptChange, err := getP2PKHScript(reservedKey.ToHash160())
			if err != nil {
				return nil, 0, err
			}

			newTxOut := txout.NewTxOut(change, scriptChange)

			// We do not move dust-change to fees, because the sender would
			// end up paying more than requested. This would be against the
			// purpose of the all-inclusive feature. So instead we raise the
			// change and deduct from the recipient.
			if subtractFeeCount > 0 && newTxOut.IsDust(dustRelayFee) {
				dust := amount.Amount(newTxOut.GetDustThreshold(dustRelayFee)) - newTxOut.GetValue()
				// Raise change until no more dust.
				newTxOut.SetValue(newTxOut.GetValue() + dust)
				// Subtract from first recipient.
				for index, recipient := range recipients {
					if recipient.SubtractFeeFromAmount {
						outValue := txn.GetTxOut(index).GetValue() - dust
						txn.GetTxOut(index).SetValue(outValue)
						if txn.GetTxOut(index).IsDust(dustRelayFee) {
							return nil, 0, errors.New("The transaction amount is too small " +
								"to send after the fee has been deducted")
						}
						break
					}
				}
			}

			// Never create dust outputs; if we would, just add the dust to
			// the fee.
			if newTxOut.IsDust(dustRelayFee) {
				*changePosInOut = -1
				feeRet += change
			} else {
				if *changePosInOut == -1 {
					// Insert change txn at random position:
					*changePosInOut = util.GetRandInt(txn.GetOutsCount() + 1)
				} else if *changePosInOut > txn.GetOutsCount() {
					return nil, 0, errors.New("Change index out of range")
				}

				txn.InsertTxOut(*changePosInOut, newTxOut)
			}
		}

		// Fill vin
		//
		// Note how the sequence number is set to non-maxint so that the
		// nLockTime set above actually works.
		coinsMap := utxo.NewEmptyCoinsMap()
		keyStore := crypto.NewKeyStore()
		pubKeyHashList := make([][]byte, 0)
		for _, txnCoin := range selectedCoins {
			txIn := txin.NewTxIn(txnCoin.OutPoint, script.NewEmptyScript(), math.MaxUint32-1)
			txn.AddTxIn(txIn)
			coinsMap.AddCoin(txnCoin.OutPoint, txnCoin.Coin, true)
			pubKeyHash := getPubKeyHash(txnCoin.Coin.GetScriptPubKey())
			pubKeyHashList = append(pubKeyHashList, pubKeyHash...)
		}
		keyPairs := GetKeyPairs(pubKeyHashList)
		keyStore.AddKeyPairs(keyPairs)

		// Fill in dummy signatures for fee calculation.
		txns := make([]*tx.Tx, 0, 1)
		txns = append(txns, txn)
		hashType := crypto.SigHashAll | crypto.SigHashForkID

		sigErrors := ltx.SignRawTransaction(txns, nil, keyStore, coinsMap, uint32(hashType))
		if len(sigErrors) > 0 {
			return nil, 0, errors.New("Signing transaction failed")
		}

		//CTransaction txNewConst(txNew);
		txSize := txn.SerializeSize()

		// Remove scriptSigs to eliminate the fee calculation dummy
		// signatures.
		for i := 0; i < txn.GetInsCount(); i++ {
			txn.UpdateInScript(i, script.NewEmptyScript())
		}

		feeNeeded := amount.Amount(wallet.GetInstance().GetMinimumFee(int(txSize)))

		// If we made it here and we aren't even able to meet the relay fee
		// on the next pass, give up because we must be at the maximum
		// allowed fee.
		minFee := amount.Amount(util.NewFeeRate(util.DefaultMinRelayTxFeePerK).GetFee(int(txSize)))
		if feeNeeded < minFee {
			return nil, 0, errors.New("Transaction too large for fee policy")
		}

		if feeRet >= feeNeeded {
			// Reduce fee to only the needed amount if we have change output
			// to increase. This prevents potential overpayment in fees if
			// the coins selected to meet nFeeNeeded result in a transaction
			// that requires less fee than the prior iteration.
			// TODO: The case where nSubtractFeeFromAmount > 0 remains to be
			// addressed because it requires returning the fee to the payees
			// and not the change output.
			// TODO: The case where there is no change output remains to be
			// addressed so we avoid creating too small an output.
			if feeRet > feeNeeded && *changePosInOut != -1 && subtractFeeCount == 0 {
				extraFeePaid := feeRet - feeNeeded
				newValue := txn.GetTxOut(*changePosInOut).GetValue() + extraFeePaid
				txn.GetTxOut(*changePosInOut).SetValue(newValue)
				feeRet -= extraFeePaid
			}

			// Done, enough fee included.
			break
		}

		// Try to reduce change to include necessary fee.
		if *changePosInOut != -1 && subtractFeeCount == 0 {
			minFinalChange := amount.Amount(amount.CENT / 2)
			additionalFeeNeeded := feeNeeded - feeRet
			// Only reduce change if remaining amount is still a large
			// enough output.
			if txn.GetTxOut(*changePosInOut).GetValue() >= minFinalChange+additionalFeeNeeded {
				newValue := txn.GetTxOut(*changePosInOut).GetValue() - additionalFeeNeeded
				txn.GetTxOut(*changePosInOut).SetValue(newValue)
				feeRet += additionalFeeNeeded
				// Done, able to increase fee from change.
				break
			}
		}

		// Include more fee and try again.
		feeRet = feeNeeded
		continue
	}

	if sign {
		txns := make([]*tx.Tx, 1)
		txns[0] = txn
		hashType := crypto.SigHashAll | crypto.SigHashForkID
		coinsMap := utxo.NewEmptyCoinsMap()
		keyStore := crypto.NewKeyStore()
		pubKeyHashList := make([][]byte, 0)
		for _, txnCoin := range selectedCoins {
			coinsMap.AddCoin(txnCoin.OutPoint, txnCoin.Coin, true)
			pubKeyHash := getPubKeyHash(txnCoin.Coin.GetScriptPubKey())
			pubKeyHashList = append(pubKeyHashList, pubKeyHash...)
		}
		keyPairs := GetKeyPairs(pubKeyHashList)
		keyStore.AddKeyPairs(keyPairs)

		sigErrors := ltx.SignRawTransaction(txns, nil, keyStore, coinsMap, uint32(hashType))
		if len(sigErrors) > 0 {
			return nil, 0, errors.New("Signing transaction failed")
		}
	}

	// Limit size.
	if txn.SerializeSize() >= uint32(tx.MaxStandardTxSize) {
		return nil, 0, errors.New("Transaction too large")
	}
	return txn, feeRet, nil
}

func selectCoins(coins []*TxnCoin, targetValue amount.Amount) ([]*TxnCoin, amount.Amount) {
	var selectedCoins []*TxnCoin
	var valueRet amount.Amount
	// TODO:a simple implementation just for testing, not support 'preset inputs'
	selectedCoins, valueRet = SelectCoinsMinConf(targetValue, 1, 6, 0, coins)
	if selectedCoins != nil {
		return selectedCoins, valueRet
	}
	selectedCoins, valueRet = SelectCoinsMinConf(targetValue, 1, 1, 0, coins)
	if selectedCoins != nil {
		return selectedCoins, valueRet
	}
	if SpendZeroConfChange {
		selectedCoins, valueRet = SelectCoinsMinConf(targetValue, 0, 1, 6, coins)
		if selectedCoins != nil {
			return selectedCoins, valueRet
		}
	}
	return nil, 0
}

func generateScript(data ...interface{}) (*script.Script, error) {
	sc := script.NewEmptyScript()
	for _, item := range data {
		switch item.(type) {
		case int:
			if err := sc.PushOpCode(item.(int)); err != nil {
				return nil, err
			}
		case []byte:
			if err := sc.PushSingleData(item.([]byte)); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("push unknown type")
		}
	}
	return sc, nil
}

func getP2PKHScript(pubkeyHash []byte) (*script.Script, error) {
	scriptPubKey, err := generateScript(opcodes.OP_DUP, opcodes.OP_HASH160, pubkeyHash,
		opcodes.OP_EQUALVERIFY, opcodes.OP_CHECKSIG)
	if err != nil {
		return nil, err
	}
	return scriptPubKey, nil
}

func getPubKeyHash(scriptPubKey *script.Script) [][]byte {
	pubKeyHash := make([][]byte, 0)
	pubKeyType, pubKeys, _ := scriptPubKey.IsStandardScriptPubKey()

	if pubKeyType == script.ScriptPubkey {
		pubKeyHash = append(pubKeyHash, util.Hash160(pubKeys[0]))

	} else if pubKeyType == script.ScriptPubkeyHash {
		pubKeyHash = append(pubKeyHash, pubKeys[0])

	} else if pubKeyType == script.ScriptMultiSig {
		for _, pubKey := range pubKeys[1:] {
			pubKeyHash = append(pubKeyHash, util.Hash160(pubKey))
		}
	}
	return pubKeyHash
}

func SelectCoinsMinConf(targetValue amount.Amount, confMine int, confTheirs int,
	maxAncestors int, coins []*TxnCoin) ([]*TxnCoin, amount.Amount) {
	// TODO:not support 'theirs' and 'maxAncestors'
	lowestLargerValue := amount.Amount(0)
	totalLowerValue := amount.Amount(0)

	var lowestLargerCoin *TxnCoin
	lowerCoins := make([]*TxnCoin, 0)
	retCoins := make([]*TxnCoin, 0)

	for _, txnCoin := range coins {
		coin := txnCoin.Coin
		if confMine > 0 && coin.IsMempoolCoin() {
			continue
		}

		coinValue := coin.GetAmount()
		if coinValue == targetValue {
			retCoins = append(retCoins, txnCoin)
			return retCoins, coinValue

		} else if coinValue > targetValue {
			if lowestLargerValue == 0 || lowestLargerValue > coinValue {
				lowestLargerValue = coinValue
				lowestLargerCoin = txnCoin
			}

		} else {
			totalLowerValue += coinValue
			lowerCoins = append(lowerCoins, txnCoin)
		}
	}

	if totalLowerValue < targetValue {
		if lowestLargerValue > 0 {
			retCoins = append(retCoins, lowestLargerCoin)
			return retCoins, lowestLargerValue
		}
		return nil, 0

	} else if totalLowerValue == targetValue {
		return lowerCoins, totalLowerValue
	}

	// TODO:a simple implementation just for testing
	retValue := amount.Amount(0)
	for _, txnCoin := range coins {
		coin := txnCoin.Coin
		retValue += coin.GetAmount()
		retCoins = append(retCoins, txnCoin)
		if retValue >= targetValue {
			break
		}
	}
	return retCoins, retValue
}

func CommitTransaction(txNew *tx.Tx, extInfo map[string]string) error {
	var err error
	txHash := txNew.GetHash()
	log.Info("CommitTransaction:%s", txHash)

	// Add tx to wallet, because if it has change it's also ours, otherwise just
	// for transaction history.
	AddToWallet(txNew, util.HashZero, extInfo)

	// Notify that old coins are spent.
	for _, txIn := range txNew.GetIns() {
		wallet.GetInstance().MarkSpent(txIn.PreviousOutPoint)
	}

	// Track how many getdata requests our transaction gets.
	//mapRequestCount[wtxNew.GetId()] = 0;

	// Broadcast
	if err = lmempool.AcceptTxToMemPool(txNew); err != nil {
		log.Error("CommitTransaction AcceptTxToMemPool fail. txid:%s, error:%s",
			txHash.String(), err.Error())
		// TODO: if we expect the failure to be long term or permanent,
		// instead delete wtx from the wallet and return failure.
		return err
	}

	if wallet.GetInstance().GetBroadcastTx() {
		txInvMsg := wire.NewInvVect(wire.InvTypeTx, &txHash)
		_, err = server.ProcessForRPC(txInvMsg)
		if err != nil {
			log.Error("CommitTransaction process InvTypeTx msg error:%s", err.Error())
		}
	}

	return err
}

func IsMine(sc *script.Script) bool {
	return wallet.IsUnlockable(sc)
}
