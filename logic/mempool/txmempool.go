package mempool

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/pkg/errors"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/outpoint"
	ltx "github.com/btcboost/copernicus/logic/tx"
	//"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/utxo"
	//"github.com/btcboost/copernicus/log"
)

const (
	OrphanTxExpireTime = 20 * 60
	OrphanTxExpireInterval = 5 * 60
	DefaultMaxOrphanTransaction = 100
)

var mapOrphanTransactionsByPrev map[outpoint.OutPoint]map[util.Hash]orphanTx
var mapOrphanTransactions	map[util.Hash]orphanTx
var RecentRejects			map[util.Hash]struct{}

type orphanTx struct {
	tx         *tx.Tx
	nodeID        int64
	expiration int
}

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
	if !err {
		return errors.New("")
	}

	//three : add transaction to mempool.
	txentry := mempool.NewTxentry(tx, txfee, 0, mpHeight, lp,0, false )
	mempool.Gpool.AddTx(txentry, ancestors)

	return nil
}

func processOrphan(work []outpoint.OutPoint)  {

}

/*

func ProcessTransaction(tx *tx.Tx, node Node) error {
	vWorkQueue := make([]outpoint.OutPoint, 0)
	vEraseQueue := make([]util.Hash, 0)

	AccpetTxToMemPool(tx, nil)
	err := util.ErrToProject(1, "")
	if err == nil{
		//todo !!! replay this transaction
		//	RelayTransaction(tx, connman)

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
			if byPrev, ok := mapOrphanTransactionsByPrev[prevOut]; !ok{
				continue
			}else {
				for iHash, iOrphanTx := range byPrev{
					fromPeer := iOrphanTx.nodeID
					if _, ok := setMisbehaving[fromPeer]; ok{
						continue
					}

					err2 := AccpetTxToMemPool(tx, nil)
					if err2 == nil{
						//	todo.. relay this transaction
						//	RelayTransaction(orphanTx, connman);
					}
					for i := 0; i < iOrphanTx.tx.GetOutsCount(); i++{
						o := outpoint.OutPoint{Hash:iOrphanTx.tx.Hash, Index:uint32(i)}
						vWorkQueue = append(vWorkQueue, o)
					}
					vEraseQueue = append(vEraseQueue, iOrphanTx.tx.Hash)

					errCode := err2.(Type)
					if errCode != MissParentCode {
						// todo !!!  punish peer that gave us an invalid orphan tx
						if errCode > 0{

						}
						vEraseQueue = append(vEraseQueue, iHash)
						if errCode == CorruptCode {
							RecentRejects[iOrphanTx.tx.Hash] = struct{}{}
						}
					}
				}
			}
		}
		for _, eraseHash := range vEraseQueue{
			eraseOrphanTx(eraseHash)
		}
	}

	proErr := err.(util.ProjectError)

	if proErr.ErrorCode == MissParentCode {
		fRejectedParents := false
		for _, preOut := range tx.GetAllPreviousOut() {
			if _, ok := RecentRejects[preOut.Hash]; ok {
				fRejectedParents = true
				break
			}
		}
		if !fRejectedParents {
			for _, preOut := range tx.GetAllPreviousOut() {
				//	require its parent transaction for all connect net node.
			}
			addOrphanTx(tx, nodeID)
		}
		evicted := limitOrphanTx()
		if evicted > 0 {
			//todo add log
			log.()
		}
	}else{
		if proErr.ErrorCode == CorruptionCode {
			RecentRejects[tx.Hash] = struct{}{}
		}
	}

	return nil
}

func addOrphanTx(orphantx *tx.Tx, nodeID int64)  {
	if _, ok := mapOrphanTransactions[orphantx.Hash]; ok{
		return
	}
	sz := orphantx.SerializeSize()
	if sz >= consensus.MaxTxSize {
		return
	}
	o := orphanTx{tx:orphantx, nodeID: nodeID, expiration:time.Now().Second() + OrphanTxExpireTime}
	mapOrphanTransactions[orphantx.Hash] = o
	for _, preout := range orphantx.GetAllPreviousOut(){
		if exsit, ok := mapOrphanTransactionsByPrev[preout]; ok {
			exsit[orphantx.Hash] = o
		}else{
			m := make(map[util.Hash]orphanTx)
			m[orphantx.Hash] = o
			mapOrphanTransactionsByPrev[preout] = m
		}
	}
}

func eraseOrphanTx(txHash util.Hash) int {
	if orphanTx, ok := mapOrphanTransactions[txHash]; ok{
		for _, preout := range orphanTx.tx.GetAllPreviousOut(){
			if m, exsit := mapOrphanTransactionsByPrev[preout]; exsit {
				delete(m, txHash)
				if len(m) == 0{
					delete(mapOrphanTransactionsByPrev, preout)
				}
			}
		}
		delete(mapOrphanTransactions, txHash)
		return 1
	}
	return 0
}

var nextSweep int
func limitOrphanTx() int {

	removeNum := 0
	now := time.Now().Second()
	if nextSweep <= now{
		minExpTime := now + OrphanTxExpireTime - OrphanTxExpireInterval
		for hash, orphan := range mapOrphanTransactions{
			if orphan.expiration <= now{
				removeNum += eraseOrphanTx(hash)
			}else {
				if minExpTime > orphan.expiration{
					minExpTime = orphan.expiration
				}
			}
		}
		nextSweep = minExpTime + OrphanTxExpireInterval
	}

	for {
		if len(mapOrphanTransactions) < DefaultMaxOrphanTransaction{
			break
		}
		for hash := range mapOrphanTransactions{
			removeNum += eraseOrphanTx(hash)
		}
	}
	return removeNum
}
*/

