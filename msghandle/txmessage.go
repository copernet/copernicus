package msghandle

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/blockchain"
)

type OrphanTx struct {
	tx *core.Tx
	nodeId int
	TimeExpire int64
}


var mapOrphanTransactionsByPrev map[utils.Hash]OrphanTx
var mapRejectTransaction  map[utils.Hash]struct{}

func ProcessTxMessage(tx *core.Tx) {

	v := blockchain.Validation{}
	missParent, err := v.CheckTx(tx, 0)
	if err != nil{

	}
	if missParent {
		// It may be the case that the orphans parents have all been
		// rejected.
		fRejectedParents := false
		for _, txin := range tx.Ins{
			if _, ok:= mapRejectTransaction[txin.PreviousOutPoint.Hash]; ok{
				fRejectedParents = true
				break
			}
		}
		if !fRejectedParents{
			msgInv := NewInv()
		}

	}


	AccpetTxToMemPool(...)

}


// AccpetTxToMemPool add one check corret transaction to mempool.
func AccpetTxToMemPool(tx *core.Tx, allowOrphans bool, rateLimit bool, peerID int) ([]*core.Tx, error) {



	return nil, nil
}
