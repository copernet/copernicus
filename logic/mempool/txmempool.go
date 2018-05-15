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
	"github.com/astaxie/beego/logs"
)

const	(
	MissParentCode = iota
	CorruptionCode
)

// AccpetTxToMemPool add one check corret transaction to mempool.
func AccpetTxToMemPool(tx *tx.Tx, activaChain *chain.Chain) error {

	//first : check transaction context And itself.
	if !ltx.CheckRegularTransaction(tx, nil, false) {
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

func ProcessTransaction(tx *tx.Tx, nodeID int64) error {

	err := AccpetTxToMemPool(tx, nil)
	if err == nil{
		//todo !!! replay this transaction
		//	RelayTransaction(tx, connman)
		processOrphan(tx)
	}

	proErr := err.(*util.ProjectError)
	if proErr.ErrorCode == MissParentCode {
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
			logs.Debug("")
		}
	}else{
		if proErr.ErrorCode == CorruptionCode {
			mempool.Gpool.RecentRejects[tx.Hash] = struct{}{}
		}
	}

	return nil
}

func processOrphan(tx *tx.Tx)  {
	vWorkQueue := make([]outpoint.OutPoint, 0)
	vEraseQueue := make([]util.Hash, 0)

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
		if byPrev, ok := mempool.Gpool.OrphanTransactionsByPrev[prevOut]; !ok{
			continue
		}else {
			for iHash, iOrphanTx := range byPrev{
				fromPeer := iOrphanTx.NodeID
				if _, ok := setMisbehaving[fromPeer]; ok{
					continue
				}

				err2 := AccpetTxToMemPool(iOrphanTx.Tx, nil)
				if err2 == nil{
					//	todo.. relay this transaction
					//	RelayTransaction(orphanTx, connman);
				}
				for i := 0; i < iOrphanTx.Tx.GetOutsCount(); i++{
					o := outpoint.OutPoint{Hash:iOrphanTx.Tx.Hash, Index:uint32(i)}
					vWorkQueue = append(vWorkQueue, o)
				}
				vEraseQueue = append(vEraseQueue, iOrphanTx.Tx.Hash)

				errCode := err2.(*util.ProjectError)
				if errCode.ErrorCode != MissParentCode {
					// todo !!!  punish peer that gave us an invalid orphan tx
					if errCode.ErrorCode > 0{

					}
					vEraseQueue = append(vEraseQueue, iHash)
					if errCode.ErrorCode == CorruptionCode {
						mempool.Gpool.RecentRejects[iOrphanTx.Tx.Hash] = struct{}{}
					}
				}
			}
		}
	}
	for _, eraseHash := range vEraseQueue{
		mempool.Gpool.EraseOrphanTx(eraseHash)
	}
}




