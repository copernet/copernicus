package undo

import (
	"io"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

const MaxInputPerTx = tx.MaxTxInPerMessage

type DisconnectResult int

const (
	// DisconnectOk All good.
	DisconnectOk DisconnectResult = iota
	// DisconnectUnclean Rolled back, but UTXO set was inconsistent with block.
	DisconnectUnclean
	// DisconnectFailed Something else went wrong.
	DisconnectFailed
)



type TxUndo struct {
	undoCoins []*utxo.Coin
}

func (tu *TxUndo) AddUndoCoin(coin *utxo.Coin){
	tu.undoCoins = append(tu.undoCoins, coin)
}

func (tu *TxUndo) SetUndoCoins(coins []*utxo.Coin) []*utxo.Coin{
	 tu.undoCoins = coins
}

func (tu *TxUndo) GetUndoCoins() []*utxo.Coin{
	return tu.undoCoins
}

func (tu *TxUndo) Serialize(w io.Writer) error {
	err := util.WriteVarInt(w, uint64(len(tu.undoCoins)))
	if err != nil {
		return err
	}
	for _, coin := range tu.undoCoins {
		err = coin.Serialize(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tu *TxUndo)Unserialize(r io.Reader) error {

	 count, err := util.ReadVarInt(r)
	 if err != nil{
	 	return err
	 }
	if count > MaxInputPerTx {
		panic("Too many input undo records")
	}
	preouts := make([]*utxo.Coin, count,count)
	for i:=0; i<int(count); i++{
		coin := utxo.NewEmptyCoin()
		err := coin.Unserialize(r)

		if err != nil {
			return err
		}
		preouts[i] = coin
	}
	tu.PrevOut = preouts
	return nil
}

func (tu *TxUndo) NewEmptyObj() *TxUndo {
	return &TxUndo{}
}

type BlockUndo struct {
	txundo []*TxUndo
}

func (bu *BlockUndo)GetTxundo()[]*TxUndo{
	return bu.txundo
}

func NewBlockUndo() *BlockUndo {
	return &BlockUndo{
		txundo: make([]*TxUndo, 0),
	}
}

func (bu *BlockUndo) Serialize(w io.Writer) error {
	count := len(bu.txundo)
	util.WriteVarLenInt(w, uint64(count))
	for _, obj := range bu.txundo{
		err := obj.Serialize(w)
		return err
	}
	return nil

}

func (bu *BlockUndo) Unserialize(r io.Reader) error {
	count, err := util.ReadVarLenInt(r)
	txundos := make([]*TxUndo, count, count)
	for i := 0; i<int(count); i++{
		obj := newTxUndo()
		err = obj.Unserialize(r)
		if err != nil{
			return err
		}
		txundos[i] =  obj
	}
	bu.txundo = txundos
	return nil
}


func newTxUndo() *TxUndo {
	return &TxUndo{
		PrevOut: make([]*utxo.Coin, 0),
	}
}
