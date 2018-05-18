package utxo
//
//import (
//	"fmt"
//	"github.com/btcboost/copernicus/model/consensus"
//	"strconv"
//	"copernicus/core"
//	"copernicus/utils"
//	"github.com/btcboost/copernicus/model/tx"
//)
//
//func (coinsViewCache *CoinsViewCache) CheckTxInputs(tx *tx.Tx, state *core.ValidationState, spendHeight int) bool {
//	// This doesn't trigger the DoS code on purpose; if it did, it would make it
//	// easier for an attacker to attempt to split the network.
//	view := coinsViewCache
//	if !view.HaveInputs(tx) {
//		return state.Invalid(false, 0, "", "Inputs unavailable")
//	}
//
//	valueIn := utils.Amount(0)
//	fees := utils.Amount(0)
//	length := len(tx.Ins)
//	for i := 0; i < length; i++ {
//		prevout := tx.Ins[i].PreviousOutPoint
//		coin, _ := view.GetCoin(prevout)
//		if coin.IsSpent() {
//			panic("critical error")
//		}
//
//		// If prev is coinbase, check that it's matured
//		if coin.IsCoinBase() {
//			sub := spendHeight - int(coin.GetHeight())
//			if sub < consensus.CoinbaseMaturity {
//				return state.Invalid(false, core.RejectInvalid, "bad-txns-premature-spend-of-coinbase",
//					"tried to spend coinbase at depth"+strconv.Itoa(sub))
//			}
//		}
//
//		// Check for negative or overflow input values
//		valueIn += utils.Amount(coin.txOut.GetValue())
//		if !utils.MoneyRange(coin.txOut.GetValue()) || !utils.MoneyRange(int64(valueIn)) {
//			return state.Dos(100, false, core.RejectInvalid,
//				"bad-txns-inputvalues-outofrange", false, "")
//		}
//	}
//
//	if int64(valueIn) < tx.GetValueOut() {
//		return state.Dos(100, false, core.RejectInvalid, "bad-txns-in-belowout", false,
//			fmt.Sprintf("value in (%s) < value out (%s)", valueIn.String(), utils.Amount(tx.GetValueOut()).String()))
//	}
//
//	// Tally transaction fees
//	txFee := int64(valueIn) - tx.GetValueOut()
//	if txFee < 0 {
//		return state.Dos(100, false, core.RejectInvalid,
//			"bad-txns-fee-negative", false, "")
//	}
//
//	fees += utils.Amount(txFee)
//	if !utils.MoneyRange(int64(fees)) {
//		return state.Dos(100, false, core.RejectInvalid,
//			"bad-txns-fee-outofrange", false, "")
//	}
//
//	return true
//}
//
//func (coinsViewCache *CoinsViewCache) UpdateCoins(tx *core.Tx, height int) (undo []*Coin) {
//	inputs := coinsViewCache
//	// Mark inputs spent.
//	if !(tx.IsCoinBase()) {
//		undo = make([]*Coin, 0, len(tx.Ins))
//		for _, txin := range tx.Ins {
//			undo = append(undo, NewEmptyCoin())
//			isSpent := inputs.SpendCoin(txin.PreviousOutPoint, undo[len(undo)-1])
//			if !isSpent {
//				panic("the coin is spent ..")
//			}
//		}
//	}
//
//	// Add outputs.
//	AddCoins(*inputs, *tx, height)
//	return
//}
