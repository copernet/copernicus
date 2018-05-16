package tx

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"

	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/errcode"
)

func CheckRegularTransaction(tx *tx.Tx, allowLargeOpReturn bool) bool {
	return true
}

func CheckBlockCoinBaseTransaction(tx *tx.Tx, allowLargeOpReturn bool) bool {
	return true
}

func CheckBlockRegularTransaction(tx *tx.Tx, allowLargeOpReturn bool) bool {
	tempCoinsMap :=  utxo.NewEmptyCoinsMap()

	err := checkInputs(tx, tempCoinsMap, 1)
	if err != nil{
		return false
	}

	return true
}

func SubmitTransaction(txs []*tx.Tx) bool {
	return true
}
/*
func UndoTransaction(txs []*txundo.TxUndo) bool {
	return true
}
*/

func checkInputs(tx *tx.Tx, tempCoinMap *utxo.CoinsMap, flags uint32) error {
	ins := tx.GetIns()
	for _, in := range ins {
		coin := tempCoinMap.GetCoin(in.PreviousOutPoint)
		if coin == nil {
			return errcode.New(errcode.ErrorNoPreviousOut)
		}
		scriptPubKey := coin.GetTxOut().GetScriptPubKey()
		scriptSig := in.GetScriptSig()
		if flags & script.ScriptEnableSigHashForkId != 0 {
			flags |= script.ScriptVerifyStrictEnc
		}
		if flags & script.ScriptVerifySigPushOnly != 0 && !scriptSig.IsPushOnly() {
			return errcode.New(errcode.ScriptErrSigPushOnly)
		}
		stack := util.NewStack()
		err := scriptSig.Eval(stack, flags)
		if err != nil {
			return err
		}

		err = scriptPubKey.Eval(stack, flags)
		if err != nil {
			return err
		}
		if stack.Empty() {
			return errcode.New(errcode.ScriptErrEvalFalse)
		}

	}
	return nil
}