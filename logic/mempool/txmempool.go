package mempool

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/pkg/errors"
)


// AccpetTxToMemPool add one check corret transaction to mempool.
func AccpetTxToMemPool(tx *tx.Tx) error {

	//first : check transaction context And itself.

	//second : check whether enter mempool.
	tip := chain.Tip()
	mpHeight := 0
	allPreout := tx.GetAllPreviousOut()
	coins := make([]*Coin, len(allPreout))
	var txfee int64
	var inputValue int64
	for i, preout := range allPreout{
		if coin := pcoins.GetCoin(preout); coin != nil{
			coins[i] = coin
			inputValue += coin.Value
		} else {
			if coin := m.GetCoin(&preout); coin != nil{
				coins[i] = coin
				inputValue += coin.Value
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








