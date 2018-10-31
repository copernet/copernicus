package rpc

import (
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lwallet"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/wallet"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/util/amount"
	"github.com/pkg/errors"
)

var walletHandlers = map[string]commandHandler{
	"getnewaddress": handleGetNewAddress,
	"listunspent":   handleListUnspent,
	"settxfee":      handleSetTxFee,
	"sendtoaddress": handleSendToAddress,
	"getbalance":    handleGetBalance,
}

var walletDisableRPCError = &btcjson.RPCError{
	Code:    btcjson.ErrRPCMethodNotFound.Code,
	Message: "Method not found (wallet method is disabled because no wallet is loaded)",
}

func handleGetNewAddress(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	if !lwallet.IsWalletEnable() {
		return nil, walletDisableRPCError
	}

	c := cmd.(*btcjson.GetNewAddressCmd)

	account := *c.Account
	address, err := lwallet.GetNewAddress(account, false)
	if err != nil {
		log.Info("GetNewAddress error:%s", err.Error())
		return nil, btcjson.ErrRPCInternal
	}

	return address, nil
}

func handleListUnspent(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	if !lwallet.IsWalletEnable() {
		return nil, walletDisableRPCError
	}

	c := cmd.(*btcjson.ListUnspentCmd)

	minDepth := *c.MinConf
	maxDepth := *c.MaxConf
	includeUnsafe := *c.IncludeUnsafe
	addresses := make(map[string]string)
	if c.Addresses != nil {
		for _, address := range *c.Addresses {
			_, keyHash, rpcErr := decodeAddress(address)
			if rpcErr != nil {
				return nil, rpcErr
			}
			if _, ok := addresses[string(keyHash)]; ok {
				return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidParameter,
					"Invalid parameter, duplicated address: "+address)
			}
			addresses[string(keyHash)] = address
		}
	}

	results := make([]*btcjson.ListUnspentResult, 0)

	coins := lwallet.AvailableCoins(!includeUnsafe, true)
	for _, txnCoin := range coins {
		depth := int32(0)
		if !txnCoin.Coin.IsMempoolCoin() {
			depth = chain.GetInstance().Height() - txnCoin.Coin.GetHeight() + 1
		}
		if depth < minDepth || depth > maxDepth {
			continue
		}
		scriptPubKey := txnCoin.Coin.GetScriptPubKey()
		scriptType, scriptAddresses, _, err := scriptPubKey.ExtractDestinations()
		if err != nil || len(scriptAddresses) != 1 {
			continue
		}
		keyHash := scriptAddresses[0].EncodeToPubKeyHash()

		var address string
		if len(addresses) > 0 {
			var ok bool
			if address, ok = addresses[string(keyHash)]; !ok {
				continue
			}
		} else {
			address = scriptAddresses[0].String()
		}
		unspentInfo := &btcjson.ListUnspentResult{
			TxID:          txnCoin.OutPoint.Hash.String(),
			Vout:          txnCoin.OutPoint.Index,
			Address:       address,
			ScriptPubKey:  hex.EncodeToString(scriptPubKey.Bytes()),
			Amount:        valueFromAmount(int64(txnCoin.Coin.GetAmount())),
			Confirmations: depth,
			Spendable:     true, //TODO
			Solvable:      true, //TODO
			Safe:          txnCoin.IsSafe,
		}

		if account := lwallet.GetAddressAccount(keyHash); account != "" {
			unspentInfo.Account = account
		}
		if scriptType == script.ScriptHash {
			if redeemScript := lwallet.GetScript(keyHash); redeemScript != nil {
				scriptHexString := hex.EncodeToString(redeemScript.Bytes())
				unspentInfo.RedeemScript = scriptHexString
			}
		}
		results = append(results, unspentInfo)
	}
	return results, nil
}

func handleSetTxFee(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	if !lwallet.IsWalletEnable() {
		return nil, walletDisableRPCError
	}

	c := cmd.(*btcjson.SetTxFeeCmd)

	feePaid, rpcErr := amountFromValue(c.Amount)
	if rpcErr != nil {
		return false, rpcErr
	}

	lwallet.SetFeeRate(int64(feePaid), 1000)

	return true, nil
}

func handleSendToAddress(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	if !lwallet.IsWalletEnable() {
		return nil, walletDisableRPCError
	}

	c := cmd.(*btcjson.SendToAddressCmd)

	scriptPubKey, rpcErr := getStandardScriptPubKey(c.Address, nil)
	if rpcErr != nil {
		return nil, rpcErr
	}

	// Amount
	value, rpcErr := amountFromValue(c.Amount)
	if rpcErr != nil {
		return false, rpcErr
	}

	// Wallet comments
	extInfo := make(map[string]string)
	if c.Comment != nil {
		extInfo["comment"] = *c.Comment
	}
	if c.CommentTo != nil {
		extInfo["to"] = *c.CommentTo
	}

	subtractFeeFromAmount := *c.SubtractFeeFromAmount

	txn, rpcErr := sendMoney(scriptPubKey, value, subtractFeeFromAmount, extInfo)
	if rpcErr != nil {
		return false, rpcErr
	}
	txHash := txn.GetHash()
	return txHash.String(), nil
}

func handleGetBalance(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	if !lwallet.IsWalletEnable() {
		return nil, walletDisableRPCError
	}
	//TODO add Confirmation
	balance := wallet.GetInstance().GetBalance()

	return balance.ToBTC(), nil
}

func sendMoney(scriptPubKey *script.Script, value amount.Amount, subtractFeeFromAmount bool,
	extInfo map[string]string) (*tx.Tx, *btcjson.RPCError) {

	curBalance := wallet.GetInstance().GetBalance()

	// Check amount
	if value <= 0 {
		return nil, btcjson.NewRPCError(btcjson.RPCInvalidParameter, "Invalid amount")
	}
	if value > curBalance {
		return nil, btcjson.NewRPCError(btcjson.RPCWalletInsufficientFunds, "Insufficient funds")
	}

	/*
		"Error: Peer-to-peer functionality missing or disabled");
		}*/

	// Create and send the transaction
	recipients := make([]*wallet.Recipient, 1)
	recipients[0] = &wallet.Recipient{
		ScriptPubKey:          scriptPubKey,
		Value:                 value,
		SubtractFeeFromAmount: subtractFeeFromAmount,
	}
	changePosRet := -1
	txn, feeRequired, err := lwallet.CreateTransaction(recipients, &changePosRet, true)
	if err != nil {
		if !subtractFeeFromAmount && value+feeRequired > curBalance {
			errMsg := fmt.Sprintf("Error: This transaction requires a "+
				"transaction fee of at least %s", feeRequired.String())
			err = errors.New(errMsg)
		}
		return nil, btcjson.NewRPCError(btcjson.RPCWalletError, err.Error())
	}

	err = lwallet.CommitTransaction(txn, extInfo)
	if err != nil {
		errMsg := "Error: The transaction was rejected! Reason given: " + err.Error()
		return nil, btcjson.NewRPCError(btcjson.RPCWalletError, errMsg)
	}
	return txn, nil
}

func registerWalletRPCCommands() {
	for name, handler := range walletHandlers {
		appendCommand(name, handler)
	}
}
