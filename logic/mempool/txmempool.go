package mempool

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/pkg/errors"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/outpoint"
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/errcode"
	"fmt"
)

const	(
	MissParentCode = iota
	CorruptionCode
)

// AccpetTxToMemPool add one check corret transaction to mempool.
func accpetTxToMemPool(tx *tx.Tx, activaChain *chain.Chain) error {

	//first : check transaction context And itself.
	if !ltx.CheckRegularTransaction(tx, true) {
		return errors.Errorf("")
	}

	//second : check whether enter mempool.
	utxoTip := utxo.GetUtxoCacheInstance()
	tip := activaChain.Tip()
	mpHeight := 0
	allPreout := tx.GetAllPreviousOut()
	coins := make([]*utxo.Coin, len(allPreout))
	var txfee int64
	var inputValue int64
	for i, preout := range allPreout{
		if coin, err := utxoTip.GetCoin(&preout); err == nil{
			coins[i] = coin
			inputValue += int64(coin.GetAmount())
		} else {
			if coin := mempool.Gpool.GetCoin(&preout); coin != nil{
				coins[i] = coin
				inputValue += int64(coin.GetAmount())
			}else {
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
	txentry := mempool.NewTxentry(tx, txfee, 0, mpHeight, lp,0, false )
	mempool.Gpool.AddTx(txentry, ancestors)

	return nil
}

func processOrphan(tx *tx.Tx) []*tx.Tx {
	vWorkQueue := make([]outpoint.OutPoint, 0)
	acceptTx := make([]*tx.Tx, 0)

	// first collect this tx all outPoint.
	for i := 0; i < tx.GetOutsCount(); i++{
		o := outpoint.OutPoint{Hash:tx.Hash, Index:uint32(i)}
		vWorkQueue = append(vWorkQueue, o)
	}

	//todo !!! modify this transaction send node time .
	//pfrom->nLastTXTime = GetTime();
	setMisbehaving := make(map[int64]struct{}, 0)
	for len(vWorkQueue) > 0{
		prevOut := vWorkQueue[0]
		vWorkQueue = vWorkQueue[1:]
		if orphans, ok := mempool.Gpool.OrphanTransactionsByPrev[prevOut]; !ok{
			continue
		}else {
			for _, iOrphanTx := range orphans{
				fromPeer := iOrphanTx.NodeID
				if _, ok := setMisbehaving[fromPeer]; ok{
					continue
				}

				err2 := accpetTxToMemPool(iOrphanTx.Tx, nil)
				if err2 == nil{
					acceptTx = append(acceptTx, iOrphanTx.Tx)
					for i := 0; i < iOrphanTx.Tx.GetOutsCount(); i++{
						o := outpoint.OutPoint{Hash:iOrphanTx.Tx.Hash, Index:uint32(i)}
						vWorkQueue = append(vWorkQueue, o)
					}
					mempool.Gpool.EraseOrphanTx(iOrphanTx.Tx.Hash, false)
					break
				}

				errCode := err2.(errcode.ProjectError)
				if errCode.Code != MissParentCode {
					// todo !!!  punish peer that gave us an invalid orphan tx
					if errCode.Code > 0{

					}
					mempool.Gpool.EraseOrphanTx(iOrphanTx.Tx.Hash, true)
					if errCode.Code == CorruptionCode {
						mempool.Gpool.RecentRejects[iOrphanTx.Tx.Hash] = struct{}{}
					}
					break
				}
			}
		}
	}

	return acceptTx
}

func ProcessTransaction(tx *tx.Tx, nodeID int64)([]*tx.Tx ,error ){

	acceptTx := make([]*tx.Tx, 0)
	err := accpetTxToMemPool(tx, nil)
	if err == nil{
		acceptTx = append(acceptTx, tx)
		acc := processOrphan(tx)
		if len(acc) > 0{
			temAccept := make([]*tx.Tx, len(acc) + 1)
			temAccept[0] = tx
			copy(temAccept[1:], acc[:])
			return temAccept, nil
		}
		return acceptTx, nil
	}

	proErr := err.(errcode.ProjectError)
	if proErr.Code == MissParentCode {
		fRejectedParents := false
		for _, preOut := range tx.GetAllPreviousOut() {
			if _, ok := mempool.Gpool.RecentRejects[preOut.Hash]; ok {
				fRejectedParents = true
				break
			}
		}
		if !fRejectedParents {
			for _, preOut := range tx.GetAllPreviousOut() {
				//todo... require its parent transaction for all connect net node.
				_ = preOut
			}
			mempool.Gpool.AddOrphanTx(tx, nodeID)
		}
		evicted := mempool.Gpool.LimitOrphanTx()
		if evicted > 0 {
			//todo add log
			log.Debug("")
		}
	}else{
		if proErr.Code == CorruptionCode {
			mempool.Gpool.RecentRejects[tx.Hash] = struct{}{}
		}
	}

	return nil, fmt.Errorf("")
}





