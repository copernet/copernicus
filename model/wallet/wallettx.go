package wallet

import (
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util/amount"
	"time"
)

type Recipient struct {
	ScriptPubKey          *script.Script
	Value                 amount.Amount
	SubtractFeeFromAmount bool
}

type WalletTx struct {
	Tx *tx.Tx

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

	spentStatus []bool
}

func NewWalletTx(txn *tx.Tx, extInfo map[string]string, isFromMe bool, account string) *WalletTx {
	return &WalletTx{
		Tx:              txn,
		ExtInfo:         extInfo,
		TimeReceived:    time.Now().Unix(),
		IsFromMe:        isFromMe,
		FromAccount:     account,
		availableCredit: nil,
		blockHeight:     0,
		spentStatus:     make([]bool, txn.GetOutsCount()),
	}
}

func (wtx *WalletTx) GetDepthInMainChain() int32 {
	// TODO: simple implementation just for testing
	if wtx.blockHeight != 0 {
		return chain.GetInstance().Height() - wtx.blockHeight + 1
	}

	if mempool.GetInstance().HaveTransaction(wtx.Tx) {
		return 0
	}
	outPoint := outpoint.NewOutPoint(wtx.Tx.GetHash(), 0)
	coin := utxo.GetUtxoCacheInstance().GetCoin(outPoint)
	if coin == nil {
		return 0
	}
	wtx.blockHeight = coin.GetHeight()
	return chain.GetInstance().Height() - wtx.blockHeight + 1
}

func (wtx *WalletTx) CheckFinalForForCurrentBlock() bool {
	lockTimeCutoff := chain.GetInstance().Tip().GetMedianTimePast()
	height := chain.GetInstance().Height() + 1
	return wtx.Tx.IsFinal(height, lockTimeCutoff)
}

func (wtx *WalletTx) GetAvailableCredit(useCache bool) amount.Amount {
	// Must wait until coinbase is safely deep enough in the chain before
	// valuing it.
	if wtx.Tx.IsCoinBase() && wtx.GetDepthInMainChain() <= consensus.CoinbaseMaturity {
		return 0
	}

	if useCache && wtx.availableCredit != nil {
		return *wtx.availableCredit
	}

	credit := amount.Amount(0)
	for index := 0; index < wtx.Tx.GetOutsCount(); index++ {
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
	if index >= wtx.Tx.GetOutsCount() {
		return nil
	}
	if wtx.spentStatus[index] {
		return nil
	}
	outPoint := outpoint.NewOutPoint(wtx.Tx.GetHash(), uint32(index))
	if coin := mempool.GetInstance().GetCoin(outPoint); coin != nil {
		if mempool.GetInstance().HasSpentOut(outPoint) {
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
