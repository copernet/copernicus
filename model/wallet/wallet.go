// Package wallet models the data for a wallet
// It is not a complete wallet and only provides basic testing capabilities for rpc currently
package wallet

import (
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util/amount"
	"sync"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
)

type AddressBook struct {
	Account string
	Purpose string
}

type Wallet struct {
	enable      bool
	broadcastTx bool

	keyStore     *crypto.KeyStore
	scriptStore  *ScriptStore
	reservedKeys []*crypto.PublicKey
	addressBooks map[string]*AddressBook

	txnLock     *sync.RWMutex
	walletTxns  map[util.Hash]*WalletTx
	lockedCoins map[outpoint.OutPoint]struct{}
	payTxFee    *util.FeeRate
}

var globalWallet *Wallet

/**
 * If fee estimation does not have enough data to provide estimates, use this
 * fee instead. Has no effect if not using fee estimation.
 * Override with -fallbackfee
 */
var fallbackFee = util.NewFeeRate(20000)

func GetInstance() *Wallet {
	if globalWallet == nil {
		globalWallet = &Wallet{
			enable:       true,
			broadcastTx:  false,
			keyStore:     crypto.NewKeyStore(),
			scriptStore:  NewScriptStore(),
			reservedKeys: make([]*crypto.PublicKey, 0),
			addressBooks: make(map[string]*AddressBook),
			txnLock:      new(sync.RWMutex),
			walletTxns:   make(map[util.Hash]*WalletTx),
			lockedCoins:  make(map[outpoint.OutPoint]struct{}),
			payTxFee:     util.NewFeeRate(0),
		}
	}
	return globalWallet
}

func (w *Wallet) IsEnable() bool {
	return w.enable
}

func (w *Wallet) SetEnable(enable bool) {
	w.enable = enable
}

func (w *Wallet) GenerateNewKey() *crypto.PublicKey {
	randBytes := util.GetRandHash()[:]
	privateKey := crypto.NewPrivateKeyFromBytes(randBytes, true)
	w.keyStore.AddKey(privateKey)
	return privateKey.PubKey()
}

func (w *Wallet) GetReservedKey() *crypto.PublicKey {
	// wallet function is only for testing. The keypool is not supported yet.
	// generate new key each time
	reservedKey := w.GenerateNewKey()
	w.reservedKeys = append(w.reservedKeys, reservedKey)
	return reservedKey

}

func (w *Wallet) GetAddressBook(keyHash []byte) *AddressBook {
	return w.addressBooks[string(keyHash)]
}

func (w *Wallet) SetAddressBook(keyHash []byte, account string, purpose string) {
	w.addressBooks[string(keyHash)] = &AddressBook{
		Account: account,
		Purpose: purpose,
	}
}

func (w *Wallet) GetKeyPairs(pubKeyHashList [][]byte) []*crypto.KeyPair {
	return w.keyStore.GetKeyPairs(pubKeyHashList)
}

func (w *Wallet) GetWalletTxns() []*WalletTx {
	walletTxns := make([]*WalletTx, 0, len(w.walletTxns))

	w.txnLock.RLock()
	defer w.txnLock.RUnlock()

	for _, walletTx := range w.walletTxns {
		walletTxns = append(walletTxns, walletTx)
	}
	return walletTxns
}

func (w *Wallet) IsTrusted(walletTx *WalletTx) bool {
	// Quick answer in most cases
	if !walletTx.CheckFinalForForCurrentBlock() {
		return false
	}

	depth := walletTx.GetDepthInMainChain()
	if depth >= 1 {
		return true
	}

	// Don't trust unconfirmed transactions from us unless they are in the
	// mempool.
	if !mempool.GetInstance().IsTransactionInPool(walletTx.Tx) {
		return false
	}

	w.txnLock.RLock()
	defer w.txnLock.RUnlock()

	// Trusted if all inputs are from us and are in the mempool:
	for _, txIn := range walletTx.Tx.GetIns() {
		// Transactions not sent by us: not trusted
		prevTxn, ok := w.walletTxns[txIn.PreviousOutPoint.Hash]
		if !ok {
			return false
		}
		prevOut := prevTxn.Tx.GetTxOut(int(txIn.PreviousOutPoint.Index))
		if !IsUnlockable(prevOut.GetScriptPubKey()) {
			return false
		}
	}

	return true
}

func (w *Wallet) GetScript(scriptHash []byte) *script.Script {
	return w.scriptStore.GetScript(scriptHash)
}

func (w *Wallet) GetBalance() amount.Amount {
	balance := amount.Amount(0)

	w.txnLock.RLock()
	defer w.txnLock.RUnlock()

	for _, walletTx := range w.walletTxns {
		if w.IsTrusted(walletTx) {
			balance += walletTx.GetAvailableCredit(true)
		}
	}
	return balance
}

func (w *Wallet) GetBroadcastTx() bool {
	return w.broadcastTx
}

func (w *Wallet) SetBroadcastTx(broadcastTx bool) {
	w.broadcastTx = broadcastTx
}

func (w *Wallet) SetFeeRate(feePaid int64, byteSize int64) {
	w.payTxFee = util.NewFeeRateWithSize(feePaid, byteSize)
}

func (w *Wallet) AddToWallet(txn *tx.Tx, extInfo map[string]string) {
	if extInfo == nil {
		extInfo = make(map[string]string)
	}
	walletTxn := NewWalletTx(txn, extInfo, true, "")
	w.txnLock.Lock()
	defer w.txnLock.Unlock()
	w.walletTxns[txn.GetHash()] = walletTxn
}

func (w *Wallet) GetMinimumFee(byteSize int) int64 {
	feeNeeded := w.payTxFee.GetFee(byteSize)
	// User didn't set tx fee
	if feeNeeded == 0 {
		minFeeRate := mempool.GetInstance().GetMinFeeRate()
		feeNeeded = minFeeRate.GetFee(byteSize)

		// ... unless we don't have enough mempool data for estimatefee, then
		// use fallbackFee.
		if feeNeeded == 0 {
			feeNeeded = fallbackFee.GetFee(byteSize)
		}
	}

	// Prevent user from paying a fee below minRelayTxFee or minTxFee.
	cfgMinFeeRate := util.NewFeeRate(conf.Cfg.Mempool.MinFeeRate)
	feeNeeded = util.MaxI(feeNeeded, cfgMinFeeRate.GetFee(byteSize))

	// But always obey the maximum.
	feeNeeded = util.MinI(feeNeeded, util.MaxFee)

	return feeNeeded
}

func (w *Wallet) GetUnspentCoin(outPoint *outpoint.OutPoint) *utxo.Coin {
	w.txnLock.RLock()
	defer w.txnLock.RUnlock()
	if wtx, ok := w.walletTxns[outPoint.Hash]; ok {
		return wtx.GetUnspentCoin(int(outPoint.Index))
	}
	return nil
}

func (w *Wallet) MarkSpent(outPoint *outpoint.OutPoint) {
	w.txnLock.RLock()
	defer w.txnLock.RUnlock()
	if wtx, ok := w.walletTxns[outPoint.Hash]; ok {
		wtx.MarkSpent(int(outPoint.Index))
	}
}

func IsUnlockable(scriptPubKey *script.Script) bool {
	if globalWallet == nil || scriptPubKey == nil {
		return false
	}

	pubKeyType, pubKeys, isStandard := scriptPubKey.IsStandardScriptPubKey()
	if !isStandard || pubKeyType == script.ScriptNonStandard || pubKeyType == script.ScriptNullData {
		return false
	}

	if pubKeyType == script.ScriptHash {
		redeemScript := globalWallet.scriptStore.GetScript(pubKeys[0])
		if redeemScript == nil {
			return false
		}
		pubKeyType, pubKeys, isStandard = redeemScript.IsStandardScriptPubKey()
		if !isStandard || pubKeyType == script.ScriptNonStandard || pubKeyType == script.ScriptNullData {
			return false
		}
	}

	if pubKeyType == script.ScriptPubkey {
		pubKeyHash := util.Hash160(pubKeys[0])
		return globalWallet.keyStore.GetKeyPair(pubKeyHash) != nil

	} else if pubKeyType == script.ScriptPubkeyHash {
		return globalWallet.keyStore.GetKeyPair(pubKeys[0]) != nil

	} else if pubKeyType == script.ScriptMultiSig {
		// Only consider transactions "mine" if we own ALL the keys
		// involved. Multi-signature transactions that are partially owned
		// (somebody else has a key that can spend them) enable
		// spend-out-from-under-you attacks, especially in shared-wallet
		// situations.
		for _, pubKey := range pubKeys[1:] {
			pubKeyHash := util.Hash160(pubKey)
			if globalWallet.keyStore.GetKeyPair(pubKeyHash) == nil {
				return false
			}
		}
		return true
	}
	return false
}
