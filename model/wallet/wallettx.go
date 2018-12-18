package wallet

import (
	"bytes"
	"io"
	"time"

	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

type Recipient struct {
	ScriptPubKey          *script.Script
	Value                 amount.Amount
	SubtractFeeFromAmount bool
}

type WalletTx struct {
	*tx.Tx

	ExtInfo map[string]string

	TimeReceived int64

	/**
	 * FromMe flag is set to true for transactions that were created by the wallet
	 * on this bitcoin node, and set to 0 for transactions that were created
	 * externally and came in through the network or sendrawtransaction RPC.
	 */
	IsFromMe bool

	FromAccount string

	availableCredit *amount.Amount

	blockHeight int32
	blockHash   util.Hash

	spentStatus []bool

	fDebitCached       bool
	fCreditCached      bool
	fWatchDebitCached  bool
	fWatchCreditCached bool
	debitCached        amount.Amount
	creditCached       amount.Amount
	watchDebitCached   amount.Amount
	watchCreditCached  amount.Amount
}

func NewEmptyWalletTx() *WalletTx {
	return &WalletTx{}
}

func NewWalletTx(txn *tx.Tx, blockhash util.Hash, extInfo map[string]string, isFromMe bool, account string) *WalletTx {
	if extInfo == nil {
		extInfo = make(map[string]string)
	}
	return &WalletTx{
		Tx:              txn,
		ExtInfo:         extInfo,
		TimeReceived:    time.Now().Unix(),
		IsFromMe:        isFromMe,
		FromAccount:     account,
		availableCredit: nil,
		blockHeight:     0,
		blockHash:       blockhash,
		spentStatus:     make([]bool, txn.GetOutsCount()),
	}
}

func (wtx *WalletTx) Serialize(writer io.Writer) error {
	var err error

	if err = wtx.Tx.Serialize(writer); err != nil {
		return err
	}

	if err = util.WriteElements(writer, wtx.TimeReceived, wtx.IsFromMe, &wtx.blockHash, wtx.blockHeight); err != nil {
		return err
	}

	if err = util.WriteVarString(writer, wtx.FromAccount); err != nil {
		return err
	}

	if err = util.WriteVarInt(writer, uint64(len(wtx.ExtInfo))); err != nil {
		return err
	}
	for key, value := range wtx.ExtInfo {
		if err = util.WriteVarString(writer, key); err != nil {
			return err
		}
		if err = util.WriteVarString(writer, value); err != nil {
			return err
		}
	}
	return nil
}

func (wtx *WalletTx) Unserialize(reader io.Reader) error {
	var err error

	wtx.Tx = tx.NewEmptyTx()
	if err = wtx.Tx.Unserialize(reader); err != nil {
		return err
	}

	if err = util.ReadElements(reader, &wtx.TimeReceived, &wtx.IsFromMe, &wtx.blockHash, &wtx.blockHeight); err != nil {
		return err
	}

	if wtx.FromAccount, err = util.ReadVarString(reader); err != nil {
		return err
	}

	extInfoSize, err := util.ReadVarInt(reader)
	if err != nil {
		return err
	}
	wtx.ExtInfo = make(map[string]string)
	for i := 0; i < int(extInfoSize); i++ {
		key, err := util.ReadVarString(reader)
		if err != nil {
			return err
		}
		value, err := util.ReadVarString(reader)
		if err != nil {
			return err
		}
		wtx.ExtInfo[key] = value
	}
	return nil
}

func (wtx *WalletTx) SerializeSize() int {
	buf := bytes.NewBuffer(nil)
	wtx.Serialize(buf)
	return buf.Len()
}

func (wtx *WalletTx) GetDepthInMainChain() int32 {
	if wtx.blockHeight != 0 {
		return chain.GetInstance().Height() - wtx.blockHeight + 1
	}

	if mempool.GetInstance().HaveTransaction(wtx.Tx) {
		return 0
	}

	blockIndex := chain.GetInstance().FindBlockIndex(wtx.blockHash)
	if blockIndex != nil {
		wtx.blockHeight = blockIndex.Height
		return chain.GetInstance().Height() - wtx.blockHeight + 1
	}

	outPoint := outpoint.NewOutPoint(wtx.GetHash(), 0)
	coin := utxo.GetUtxoCacheInstance().GetCoin(outPoint)
	if coin != nil {
		wtx.blockHeight = coin.GetHeight()
		return chain.GetInstance().Height() - wtx.blockHeight + 1
	}

	return 0

}

func (wtx *WalletTx) CheckFinalForForCurrentBlock() bool {
	lockTimeCutoff := chain.GetInstance().Tip().GetMedianTimePast()
	height := chain.GetInstance().Height() + 1
	return wtx.IsFinal(height, lockTimeCutoff)
}

func (wtx *WalletTx) GetAvailableCredit(useCache bool) amount.Amount {
	// Must wait until coinbase is safely deep enough in the chain before
	// valuing it.
	if wtx.IsCoinBase() && wtx.GetDepthInMainChain() <= consensus.CoinbaseMaturity {
		return 0
	}

	if useCache && wtx.availableCredit != nil {
		return *wtx.availableCredit
	}

	credit := amount.Amount(0)
	for index := 0; index < wtx.GetOutsCount(); index++ {
		// check coin is unspent
		coin := wtx.GetUnspentCoin(index)
		if coin == nil {
			continue
		}
		if IsUnlockable(coin.GetScriptPubKey()) {
			credit += coin.GetAmount()
		}
	}

	wtx.availableCredit = &credit
	return credit
}

func (wtx *WalletTx) MarkSpent(index int) {
	if index < len(wtx.spentStatus) {
		wtx.spentStatus[index] = true
	}
}

func (wtx *WalletTx) GetUnspentCoin(index int) *utxo.Coin {
	if index >= wtx.GetOutsCount() {
		return nil
	}
	if wtx.spentStatus[index] {
		return nil
	}
	outPoint := outpoint.NewOutPoint(wtx.GetHash(), uint32(index))
	if coin := mempool.GetInstance().GetCoin(outPoint); coin != nil {
		if mempool.GetInstance().HasSpentOut(outPoint, true) {
			return nil
		}
		return coin
	}
	if coin := utxo.GetUtxoCacheInstance().GetCoin(outPoint); coin != nil {
		if coin.IsSpent() {
			return nil
		}
		return coin
	}
	return nil
}

func (wtx *WalletTx) GetBlokHeight() int32 {
	return wtx.blockHeight
}

func (wtx *WalletTx) GetDebit(filter uint8) amount.Amount {
	if len(wtx.GetIns()) == 0 {
		return 0
	}
	pwallet := GetInstance()
	var debit amount.Amount
	if (filter & ISMINE_SPENDABLE) != 0 {
		if wtx.fDebitCached {
			debit += wtx.debitCached
		} else {
			wtx.debitCached = pwallet.GetDebitTx(wtx, ISMINE_SPENDABLE)
			wtx.fDebitCached = true
			debit += wtx.debitCached
		}
	}

	if (filter & ISMINE_WATCH_ONLY) != 0 {
		if wtx.fWatchDebitCached {
			debit += wtx.watchDebitCached
		} else {
			wtx.watchDebitCached = pwallet.GetDebitTx(wtx, ISMINE_WATCH_ONLY)
			wtx.fWatchDebitCached = true
			debit += wtx.watchDebitCached
		}
	}

	return debit
}

func (wtx *WalletTx) GetCredit(filter uint8) amount.Amount {
	// Must wait until coinbase is safely deep enough in the chain before
	// valuing it.
	if wtx.IsCoinBase() && wtx.GetDepthInMainChain() <= consensus.CoinbaseMaturity {
		return 0
	}

	pwallet := GetInstance()
	var credit amount.Amount
	if (filter & ISMINE_SPENDABLE) != 0 {
		if wtx.fCreditCached {
			credit += wtx.creditCached
		} else {
			wtx.creditCached = pwallet.GetCreditTx(wtx, ISMINE_SPENDABLE)
			wtx.fCreditCached = true
			credit += wtx.creditCached
		}
	}

	if (filter & ISMINE_WATCH_ONLY) != 0 {
		if wtx.fWatchCreditCached {
			credit += wtx.watchCreditCached
		} else {
			wtx.watchCreditCached = pwallet.GetCreditTx(wtx, ISMINE_WATCH_ONLY)
			wtx.fWatchCreditCached = true
			credit += wtx.watchCreditCached
		}
	}

	return credit
}
