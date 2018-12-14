// Package wallet models the data for a wallet
// It is not a complete wallet and only provides basic testing capabilities for rpc currently
package wallet

import (
	"crypto/rand"
	"io"
	"sync"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

type Wallet struct {
	enable       bool
	broadcastTx  bool
	reservedKeys []*crypto.PublicKey
	txnLock      *sync.RWMutex
	walletTxns   map[util.Hash]*WalletTx
	lockedCoins  map[outpoint.OutPoint]struct{}
	payTxFee     *util.FeeRate
	wdb          WalletDB

	*crypto.KeyStore
	*ScriptStore
	*AddressBook
}

var globalWallet *Wallet

/**
 * If fee estimation does not have enough data to provide estimates, use this
 * fee instead. Has no effect if not using fee estimation.
 * Override with -fallbackfee
 */
var fallbackFee = util.NewFeeRate(20000)

func InitWallet() {
	defer func() {
		if globalWallet == nil {
			globalWallet = &Wallet{enable: false}
		}
	}()

	if !conf.Cfg.Wallet.Enable {
		return
	}

	walletInstance := &Wallet{
		enable:      true,
		broadcastTx: conf.Cfg.Wallet.Broadcast,
		txnLock:     new(sync.RWMutex),
		walletTxns:  make(map[util.Hash]*WalletTx),
		lockedCoins: make(map[outpoint.OutPoint]struct{}),
		payTxFee:    util.NewFeeRate(0),
	}

	if err := walletInstance.Init(); err != nil {
		return
	}

	chain.GetInstance().Subscribe(walletInstance.handleBlockChainNotification)

	globalWallet = walletInstance
}

func GetInstance() *Wallet {
	if globalWallet == nil {
		globalWallet = &Wallet{enable: false}
	}
	return globalWallet
}

func (w *Wallet) Init() error {
	w.KeyStore = crypto.NewKeyStore()
	w.ScriptStore = NewScriptStore()
	w.AddressBook = NewAddressBook()

	w.wdb.initDB()
	if err := w.loadFromDB(); err != nil {
		log.Error("Load wallet fail. error:" + err.Error())
		return err
	}
	return nil
}

func (w *Wallet) IsEnable() bool {
	return w.enable
}

func (w *Wallet) loadFromDB() error {
	secrets := w.wdb.loadSecrets()
	for _, secret := range secrets {
		privateKey := crypto.NewPrivateKeyFromBytes(secret, true)
		w.KeyStore.AddKey(privateKey)
	}

	scripts, err := w.wdb.loadScripts()
	if err != nil {
		return err
	}
	for _, sc := range scripts {
		w.ScriptStore.AddScript(sc)
	}

	addressBook, err := w.wdb.loadAddressBook()
	if err != nil {
		return err
	}
	for key, data := range addressBook {
		addressBookData := NewAddressBookData(data.Account, data.Purpose)
		w.AddressBook.SetAddressBook([]byte(key), addressBookData)
	}

	transactions, err := w.wdb.loadTransactions()
	if err != nil {
		return err
	}
	for _, wtx := range transactions {
		w.walletTxns[wtx.Tx.GetHash()] = wtx
	}
	log.Info("load wallet from db successfully. keys:%v, scripts:%v, addressbook:%v, txns:%v",
		len(secrets), len(scripts), len(addressBook), len(transactions))
	return nil
}

func (w *Wallet) GenerateNewKey() (*crypto.PublicKey, error) {
	secret := make([]byte, 32)
	io.ReadFull(rand.Reader, secret)
	privateKey := crypto.NewPrivateKeyFromBytes(secret, true)
	w.AddKey(privateKey)
	err := w.wdb.saveSecret(secret)
	if err != nil {
		log.Error("GenerateNewKey save to db fail. error:%s", err.Error())
		return nil, err
	}
	return privateKey.PubKey(), nil
}

func (w *Wallet) GetReservedKey() (*crypto.PublicKey, error) {
	// wallet function is only for testing. The keypool is not supported yet.
	// generate new key each time
	reservedKey, err := w.GenerateNewKey()
	if err != nil {
		return nil, err
	}
	w.reservedKeys = append(w.reservedKeys, reservedKey)
	return reservedKey, nil
}

func (w *Wallet) AddScript(s *script.Script) error {
	w.ScriptStore.AddScript(s)
	err := w.wdb.saveScript(s)
	if err != nil {
		log.Error("AddScript save to db fail. error:%s", err.Error())
		return err
	}
	return nil
}

func (w *Wallet) SetAddressBook(keyHash []byte, account string, purpose string) error {
	addressBookData := NewAddressBookData(account, purpose)
	w.AddressBook.SetAddressBook(keyHash, addressBookData)
	err := w.wdb.saveAddressBook(keyHash, addressBookData)
	if err != nil {
		log.Error("SetAddressBook save to db fail. error:%s", err.Error())
		return err
	}
	return nil
}

func (w *Wallet) AddToWallet(txn *tx.Tx, blockhash util.Hash, extInfo map[string]string) error {
	txHash := txn.GetHash()
	log.Info("AddToWallet tx:%s", txHash.String())

	walletTx := NewWalletTx(txn, blockhash, extInfo, true, "")

	w.txnLock.Lock()
	defer w.txnLock.Unlock()
	w.walletTxns[txHash] = walletTx

	err := w.wdb.saveWalletTx(walletTx)
	if err != nil {
		log.Error("AddToWallet save to db fail. error:%s", err.Error())
		return err
	}
	return nil
}

func (w *Wallet) addTxnsToWallet(txns []*tx.Tx, blockhash util.Hash) {
	w.txnLock.Lock()
	defer w.txnLock.Unlock()

	for _, txn := range txns {
		txHash := txn.GetHash()
		log.Info("AddTxnsToWallet tx:%s, hash:%v", txHash.String(), blockhash.String())

		walletTx := NewWalletTx(txn, blockhash, nil, true, "")
		w.walletTxns[txHash] = walletTx
		err := w.wdb.saveWalletTx(walletTx)
		if err != nil {
			log.Error("AddTxnsToWallet save to db fail. tx:%s, error:%s", txHash.String(), err.Error())
			continue
		}
	}
}

func (w *Wallet) RemoveFromWallet(txn *tx.Tx) error {
	txHash := txn.GetHash()
	log.Info("RemoveFromWallet tx:%s", txHash.String())

	w.txnLock.Lock()
	defer w.txnLock.Unlock()
	delete(w.walletTxns, txHash)

	err := w.wdb.removeWalletTx(&txHash)
	if err != nil {
		log.Error("AddToWallet remove from db fail. error:%s", err.Error())
		return err
	}
	return nil
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

func (w *Wallet) GetWalletTx(txhash util.Hash) *WalletTx {

	w.txnLock.RLock()
	defer w.txnLock.RUnlock()
	if tx, ok := w.walletTxns[txhash]; ok {
		return tx
	}

	return nil
}

// isTrusted is non-thread safe (without lock)
func (w *Wallet) isTrustedTx(walletTx *WalletTx) bool {
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

	// Trusted if all inputs are from us and are in the mempool:
	for _, txIn := range walletTx.GetIns() {
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

func (w *Wallet) IsTrusted(walletTx *WalletTx) bool {
	w.txnLock.RLock()
	defer w.txnLock.RUnlock()

	return w.isTrustedTx(walletTx)
}

func (w *Wallet) GetBalance() amount.Amount {
	balance := amount.Amount(0)

	w.txnLock.RLock()
	defer w.txnLock.RUnlock()

	for _, walletTx := range w.walletTxns {
		if w.isTrustedTx(walletTx) {
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
		redeemScript := globalWallet.GetScript(pubKeys[0])
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
		return globalWallet.GetKeyPair(pubKeyHash) != nil

	} else if pubKeyType == script.ScriptPubkeyHash {
		return globalWallet.GetKeyPair(pubKeys[0]) != nil

	} else if pubKeyType == script.ScriptMultiSig {
		// Only consider transactions "mine" if we own ALL the keys
		// involved. Multi-signature transactions that are partially owned
		// (somebody else has a key that can spend them) enable
		// spend-out-from-under-you attacks, especially in shared-wallet
		// situations.
		for _, pubKey := range pubKeys {
			if len(pubKey) >= 32 {
				pubKeyHash := util.Hash160(pubKey)
				if globalWallet.GetKeyPair(pubKeyHash) == nil {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (w *Wallet) GetDebitTx(walletTx *WalletTx, filter uint8) amount.Amount {
	var debit amount.Amount
	for _, in := range walletTx.GetIns() {
		debit += w.GetDebitIn(in, filter)
		if !amount.MoneyRange(debit) {
			log.Error("Wallet debit value out of range")
			return 0
		}
	}

	return debit
}

func (w *Wallet) GetCreditTx(walletTx *WalletTx, filter uint8) amount.Amount {
	var credit amount.Amount
	for _, out := range walletTx.GetOuts() {
		credit += w.GetCreditOut(out, filter)
		if !amount.MoneyRange(credit) {
			log.Error("Wallet credit value out of range")
			return 0
		}
	}

	return credit
}

func (w *Wallet) GetDebitIn(in *txin.TxIn, filter uint8) amount.Amount {
	w.txnLock.RLock()
	defer w.txnLock.RUnlock()

	prev, ok := w.walletTxns[in.PreviousOutPoint.Hash]
	if !ok {
		return 0
	}

	if int(in.PreviousOutPoint.Index) < prev.GetOutsCount() {
		prevTxOut := prev.GetTxOut(int(in.PreviousOutPoint.Index))
		if w.IsMine(prevTxOut)&filter != 0 {
			return prevTxOut.GetValue()
		}
	}

	return 0
}

func (w *Wallet) GetCreditOut(out *txout.TxOut, filter uint8) amount.Amount {
	if !amount.MoneyRange(out.GetValue()) {
		log.Error("Wallet getCreditOut value out of range")
		return 0
	}
	var value amount.Amount
	if (w.IsMine(out) & filter) != 0 {
		value = out.GetValue()
	}
	return value
}

func (w *Wallet) IsMine(out *txout.TxOut) uint8 {
	//TODO wallet add ISMINE_WATCH_ONLY
	if IsUnlockable(out.GetScriptPubKey()) {
		return ISMINE_SPENDABLE
	}

	return ISMINE_NO
}

func (w *Wallet) getRelatedTxns(txns []*tx.Tx) []*tx.Tx {
	relatedTxns := make([]*tx.Tx, 0)

	w.txnLock.RLock()
	defer w.txnLock.RUnlock()

	for _, tx := range txns {
		hasAdded := false
		for _, in := range tx.GetIns() {
			prev := in.PreviousOutPoint
			if prevTx, ok := w.walletTxns[prev.Hash]; ok {
				if w.IsMine(prevTx.GetTxOut(int(prev.Index))) != ISMINE_NO {
					relatedTxns = append(relatedTxns, tx)
					hasAdded = true
					continue
				}
			}
		}
		if hasAdded {
			continue
		}
		for _, out := range tx.GetOuts() {
			if w.IsMine(out) != ISMINE_NO {
				relatedTxns = append(relatedTxns, tx)
				continue
			}
		}
	}
	return relatedTxns
}

func (w *Wallet) HandleRelatedMempoolTx(txe *tx.Tx) {
	// TODO: simple implementation just for testing, remove this after complete wallet
	txes := []*tx.Tx{txe}
	relatedTxns := w.getRelatedTxns(txes)
	if len(relatedTxns) > 0 {
		w.addTxnsToWallet(relatedTxns, util.HashZero)
	}
}

func (w *Wallet) handleBlockChainNotification(notification *chain.Notification) {
	switch notification.Type {

	case chain.NTChainTipUpdated:

	case chain.NTNewPoWValidBlock:

	case chain.NTBlockConnected:
		block, ok := notification.Data.(*block.Block)
		if !ok {
			log.Warn("Chain connected notification is not a block.")
			break
		}
		blockHash := block.GetHash()
		log.Info("wallet process block connect event. block:%s", blockHash.String())

		relatedTxns := w.getRelatedTxns(block.Txs)
		if len(relatedTxns) > 0 {
			w.addTxnsToWallet(relatedTxns, blockHash)
		}

	case chain.NTBlockDisconnected:
		block, ok := notification.Data.(*block.Block)
		if !ok {
			log.Warn("Chain disconnected notification is not a block.")
			break
		}
		blockHash := block.GetHash()
		log.Info("wallet process block disconnect event. block:%s", blockHash.String())

		w.txnLock.RLock()
		defer w.txnLock.RUnlock()
		for _, tx := range block.Txs {
			if wtx, ok := w.walletTxns[tx.GetHash()]; ok {
				wtx.blockHeight = 0
				wtx.blockHash = util.HashZero
				w.wdb.saveWalletTx(wtx)
			}
		}
	}
}
