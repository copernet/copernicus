package blockchain

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
)

// ScriptCheck Closure representing one script verification.
// Note that this stores references to the spending transaction.
type ScriptCheck struct {
	scriptPubKey *core.Script
	amount       utils.Amount
	txTo         *core.Tx
	ins          int
	flags        uint32
	cacheStore   bool
	err          crypto.ScriptError
	txData       *core.PrecomputedTransactionData
}

func NewScriptCheck(script *core.Script, amount utils.Amount, tx *core.Tx, ins int, flags uint32,
	cacheStore bool, txData *core.PrecomputedTransactionData) *ScriptCheck {
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
	// scriptSig := sc.txTo.Ins[sc.ins].Script
	// if !core.VerifyScript(scriptSig, sc.scriptPubKey, sc.flags,, sc.err) { // todo new a CachingTransactionSignatureChecker
	// 	return false
	// }
	return true
}

func (sc *ScriptCheck) GetScriptError() crypto.ScriptError {
	return sc.err
}
