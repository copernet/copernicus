package undo

import (
	"bytes"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util"
	"io"
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

func (tu *TxUndo) SetUndoCoins(coins []*utxo.Coin) {
	tu.undoCoins = coins
}

func (tu *TxUndo) GetUndoCoins() []*utxo.Coin {
	return tu.undoCoins
}

func (tu *TxUndo) Serialize(w io.Writer) error {
	err := util.WriteVarInt(w, uint64(len(tu.undoCoins)))
	if err != nil {
		log.Error("TxUndo Serialize: serialize error: %v", err)
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

func (tu *TxUndo) Unserialize(r io.Reader) error {
	count, err := util.ReadVarInt(r)
	if err != nil {
		log.Error("TxUndo UnSerialize: the read count is: %d, error: %v", count, err)
		return err
	}
	if count > MaxInputPerTx {
		panic("Too many input undo records")
	}
	preouts := make([]*utxo.Coin, count)
	for i := 0; i < int(count); i++ {
		coin := utxo.NewFreshEmptyCoin()
		err := coin.Unserialize(r)

		if err != nil {
			log.Error("TxUndo UnSerialize: the utxo Unserialize error: %v", err)
			return err
		}
		preouts[i] = coin
	}
	tu.undoCoins = preouts
	return nil
}

type BlockUndo struct {
	txundo []*TxUndo
}

func (bu *BlockUndo) GetTxundo() []*TxUndo {
	return bu.txundo
}

func NewBlockUndo(count int) *BlockUndo {
	return &BlockUndo{
		txundo: make([]*TxUndo, 0, count),
	}
}

func (bu *BlockUndo) Serialize(w io.Writer) error {
	count := len(bu.txundo)
	err := util.WriteVarLenInt(w, uint64(count))
	if err != nil {
		log.Error("BlockUndo Serialize: serialize block undo failed:%v", err)
		return err
	}
	for _, obj := range bu.txundo {
		err := obj.Serialize(w)
		if err != nil {
			return err
		}
	}
	return nil

}

func (bu *BlockUndo) SerializeSize() int {
	buf := bytes.NewBuffer(nil)
	err := bu.Serialize(buf)
	if err != nil {
		log.Error("BlockUndo SerializeSize: serialize block undo failed:%v", err)
		return 0
	}
	return buf.Len()

}

func (bu *BlockUndo) Unserialize(r io.Reader) error {
	count, err := util.ReadVarLenInt(r)
	if err != nil {
		log.Error("BlockUndo UnSerialize: the read count is: %d, error: %v", count, err)
		return err
	}
	txundos := make([]*TxUndo, count)
	for i := 0; i < int(count); i++ {
		obj := NewTxUndo()
		err = obj.Unserialize(r)
		if err != nil {
			log.Error("BlockUndo UnSerialize: the txUndo UnSerialize error: %v", err)
			return err
		}
		txundos[i] = obj
	}
	bu.txundo = txundos
	return nil
}

func (bu *BlockUndo) SetTxUndo(txUndo []*TxUndo) {
	bu.txundo = txUndo
}
func (bu *BlockUndo) AddTxUndo(txUndo *TxUndo) {
	bu.txundo = append(bu.txundo, txUndo)
}

func NewTxUndo() *TxUndo {
	return &TxUndo{
		undoCoins: make([]*utxo.Coin, 0),
	}
}
