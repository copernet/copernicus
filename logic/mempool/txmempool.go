package mempool

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"
)

const (
	MissParentCode = iota
	CorruptionCode
)

// AccpetTxToMemPool add one check corret transaction to mempool.
func AccpetTxToMemPool(tx *tx.Tx, activaChain *chain.Chain) error {

	//first : check transaction context And itself.
	if err := ltx.CheckRegularTransaction(tx); err != nil {
		return err
	}

	//second : check whether enter mempool.
	utxoTip := utxo.GetUtxoCacheInstance()
	tip := activaChain.Tip()
	mpHeight := 0
	allPreout := tx.GetAllPreviousOut()
	coins := make([]*utxo.Coin, len(allPreout))
	var txfee int64
	var inputValue int64
	for i, preout := range allPreout {
		if coin := utxoTip.GetCoin(&preout); coin != nil {
			coins[i] = coin
			inputValue += int64(coin.GetAmount())
		} else {
			if coin := mempool.Gpool.GetCoin(&preout); coin != nil {
				coins[i] = coin
				inputValue += int64(coin.GetAmount())
			} else {
				panic("the transaction in mempool, not found its parent " +
					"transaction in local node and utxo")
			}
		}
	}
	txfee = inputValue - tx.GetValueOut()
	ancestors, lp, err := mempool.Gpool.IsAcceptTx(tx, txfee, mpHeight, coins, tip)
	if err != nil {
		return err
	}

	//three : add transaction to mempool.
	txentry := mempool.NewTxentry(tx, txfee, 0, mpHeight, lp, 0, false)
	mempool.Gpool.AddTx(txentry, ancestors)

	return nil
}

func ProcessOrphan(tx *tx.Tx) []*tx.Tx {
	vWorkQueue := make([]outpoint.OutPoint, 0)
	acceptTx := make([]*tx.Tx, 0)

	// first collect this tx all outPoint.
	for i := 0; i < tx.GetOutsCount(); i++ {

		o := outpoint.OutPoint{Hash: tx.GetHash(), Index: uint32(i)}
		vWorkQueue = append(vWorkQueue, o)
	}

	//todo !!! modify this transaction send node time .
	//pfrom->nLastTXTime = GetTime();
	setMisbehaving := make(map[int64]struct{}, 0)
	for len(vWorkQueue) > 0 {
		prevOut := vWorkQueue[0]
		vWorkQueue = vWorkQueue[1:]
		if orphans, ok := mempool.Gpool.OrphanTransactionsByPrev[prevOut]; !ok {
			continue
		} else {
			for _, iOrphanTx := range orphans {
				fromPeer := iOrphanTx.NodeID
				if _, ok := setMisbehaving[fromPeer]; ok {
					continue
				}

				err2 := AccpetTxToMemPool(iOrphanTx.Tx, nil)
				if err2 == nil {
					acceptTx = append(acceptTx, iOrphanTx.Tx)
					for i := 0; i < iOrphanTx.Tx.GetOutsCount(); i++ {
						o := outpoint.OutPoint{Hash: iOrphanTx.Tx.GetHash(), Index: uint32(i)}
						vWorkQueue = append(vWorkQueue, o)
					}
					mempool.Gpool.EraseOrphanTx(iOrphanTx.Tx.GetHash(), false)
					break
				}

				if !errcode.IsErrorCode(err2, errcode.TxErrNoPreviousOut) {
					mempool.Gpool.EraseOrphanTx(iOrphanTx.Tx.GetHash(), true)
					if errcode.IsErrorCode(err2, errcode.RejectTx) {
						mempool.Gpool.RecentRejects[iOrphanTx.Tx.GetHash()] = struct{}{}
					}
					break
				}
			}
		}
	}

	return acceptTx
}
