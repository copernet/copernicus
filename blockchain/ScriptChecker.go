package blockchain

import (
	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
)

// ScriptCheck Closure representing one script verification.
// Note that this stores references to the spending transaction.
type ScriptCheck struct {
	scriptPubKey *model.Script
	amount       btcutil.Amount
	txTo         *model.Tx
	ins          int
	flags        uint32
	cacheStore   bool
	err          core.ScriptError
	txData       *model.PrecomputedTransactionData
}

func NewScriptCheck(script *model.Script, amount btcutil.Amount, tx *model.Tx, ins int, flags uint32,
	cacheStore bool, txData *model.PrecomputedTransactionData) *ScriptCheck {
	return &ScriptCheck{
		scriptPubKey: script,
		amount:       amount,
		txTo:         tx,
		ins:          ins,
		flags:        flags,
		cacheStore:   cacheStore,
		txData:       txData,
	}
}

func (sc *ScriptCheck) check() bool {
	//scriptSig := sc.txTo.Ins[sc.ins].Script
	//if !model.VerifyScript(scriptSig, sc.scriptPubKey, sc.flags,, sc.err) { // todo new a CachingTransactionSignatureChecker
	//	return false
	//}
	return true
}

func (sc *ScriptCheck) GetScriptError() core.ScriptError {
	return sc.err
}
