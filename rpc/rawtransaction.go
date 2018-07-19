package rpc

import (
	"bytes"
	"encoding/hex"
	"math"

	"github.com/copernet/copernicus/crypto"
	mempool2 "github.com/copernet/copernicus/logic/mempool"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/server"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

var rawTransactionHandlers = map[string]commandHandler{
	"getrawtransaction":    handleGetRawTransaction,    // complete
	"createrawtransaction": handleCreateRawTransaction, // complete
	"decoderawtransaction": handleDecodeRawTransaction, // complete
	"decodescript":         handleDecodeScript,         // complete
	"sendrawtransaction":   handleSendRawTransaction,   // complete
	"signrawtransaction":   handleSignRawTransaction,   // partial complete
	"gettxoutproof":        handleGetTxoutProof,        // complete
	"verifytxoutproof":     handleVerifyTxoutProof,     // complete
}

func handleGetRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.GetRawTransactionCmd)

		// Convert the provided transaction hash hex to a Hash.
		txHash, err := util.GetHashFromStr(c.Txid)
		if err != nil {
			return nil, rpcDecodeHexError(c.Txid)
		}

		verbose := false
		if c.Verbose != nil {
			verbose = *c.Verbose != 0
		}

		tx, hashBlock, ok := GetTransaction(txHash, true)
		if !ok {
			if chain.GTxIndex { // todo define
				return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidAddressOrKey,
					"No such mempool or blockchain transaction")
			}
			return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidAddressOrKey,
				"No such mempool transaction. Use -txindex to enable blockchain transaction queries. Use gettransaction for wallet transactions.")
		}

		buf := bytes.NewBuffer(nil)
		err = tx.Serialize(buf)
		if err != nil {
			return nil, rpcDecodeHexError(c.Txid)
		}
		strHex := hex.EncodeToString(buf.Bytes())
		if !verbose {
			return strHex, nil
		}
		rawTxn, err := createTxRawResult(tx, hashBlock, consensus.ActiveNetParams)
		if err != nil {
			return nil, err
		}
		return *rawTxn, nil*///   TODO open
	return nil, nil
}

// createTxRawResult converts the passed transaction and associated parameters
// to a raw transaction JSON object.
/*func createTxRawResult(tx *tx.Tx, hashBlock *util.Hash, params *consensus.BitcoinParams) (*btcjson.TxRawResult, error) {

	hash := tx.GetHash()
	txReply := &btcjson.TxRawResult{
		TxID:     hash.String(),
		Hash:     hash.String(),
		Size:     int(tx.SerializeSize()),
		Version:  tx.GetVersion(),
		LockTime: tx.GetLockTime(),
		Vin:      createVinList(tx),
		Vout:     createVoutList(tx, params),
	}

	if !hashBlock.IsNull() {
		txReply.BlockHash = hashBlock.String()
		bindex := chain.GetInstance.FindBlockIndex(*hashBlock) // todo realise: get *BlockIndex by blockhash
		if bindex != nil {
			if chain.GetInstance.Contains(bindex) {
				txReply.Confirmations = chain.GetInstance.Height() - bindex.Height + 1
				txReply.Time = bindex.Header.Time
				txReply.Blocktime = bindex.Header.Time
			} else {
				txReply.Confirmations = 0
			}
		}
	}
	return txReply, nil
}*/// TODO open

// createVinList returns a slice of JSON objects for the inputs of the passed
// transaction.
/*func createVinList(tx *tx.Tx) []btcjson.Vin {
	vinList := make([]btcjson.Vin, len(tx.GetIns()))
	for index, in := range tx.GetIns() {
		if tx.IsCoinBase() {
			vinList[index].Coinbase = hex.EncodeToString(in.GetScriptSig().GetData())
		} else {
			vinList[index].Txid = in.PreviousOutPoint.Hash.String()
			vinList[index].Vout = in.PreviousOutPoint.Index
			vinList[index].ScriptSig.Asm = ScriptToAsmStr(in.GetScriptSig(), true)
			vinList[index].ScriptSig.Hex = hex.EncodeToString(in.GetScriptSig().GetData())
		}
		vinList[index].Sequence = in.Sequence
	}
	return vinList
}*/// TODO open

func ScriptToAsmStr(s *script.Script, attemptSighashDecode bool) string { // todo complete
	/*	var str string
		var opcode byte
		vch := make([]byte, 0)
		b := s.GetData()
		for i := 0; i < len(b); i++ {
			if len(str) != 0 {
				str += " "
			}

			if !s.GetOp(&i, &opcode, &vch) {
				str += "[error]"
				return str
			}

			if opcode >= 0 && opcode <= opcodes.OP_PUSHDATA4 {
				if len(vch) <= 4 {
					num, _ := script.GetCScriptNum(vch, false, script.DefaultMaxNumSize)
					str += fmt.Sprintf("%d", num.Value)
				} else {
					// the IsUnspendable check makes sure not to try to decode
					// OP_RETURN data that may match the format of a signature
					if attemptSighashDecode && !s.IsUnspendable() {
						var strSigHashDecode string
						// goal: only attempt to decode a defined sighash type from
						// data that looks like a signature within a scriptSig. This
						// won't decode correctly formatted public keys in Pubkey or
						// Multisig scripts due to the restrictions on the pubkey
						// formats (see IsCompressedOrUncompressedPubKey) being
						// incongruous with the checks in CheckSignatureEncoding.
						flags := script.ScriptVerifyStrictEnc
						if vch[len(vch)-1]&script.SigHashForkID != 0 {
							// If the transaction is using SIGHASH_FORKID, we need
							// to set the apropriate flag.
							// TODO: Remove after the Hard Fork.
							flags |= script.ScriptEnableSigHashForkId
						}
						if ok, _ := crypto.CheckSignatureEncoding(vch, uint32(flags)); ok {
							//chsigHashType := vch[len(vch)-1]
							//if t, ok := crypto.MapSigHashTypes[chsigHashType]; ok { // todo realise define
							//	strSigHashDecode = "[" + t + "]"
							//	// remove the sighash type byte. it will be replaced
							//	// by the decode.
							//	vch = vch[:len(vch)-1]
							//}
						}

						str += hex.EncodeToString(vch) + strSigHashDecode
					} else {
						str += hex.EncodeToString(vch)
					}
				}
			} else {
				str += opcodes.GetOpName(int(opcode))
			}
		}
		return str*/
	return ""
}

// createVoutList returns a slice of JSON objects for the outputs of the passed
// transaction.
/*func createVoutList(tx *tx.Tx, params *consensus.BitcoinParams) []btcjson.Vout {
	voutList := make([]btcjson.Vout, tx.GetOutsCount())
	for i := 0; i < tx.GetOutsCount(); i++ {
		out := tx.GetTxOut(i)
		voutList[i].Value = out.GetValue()
		voutList[i].N = uint32(i)
		voutList[i].ScriptPubKey = ScriptPubKeyToJSON(out.GetScriptPubKey(), true)
	}

	return voutList
}*/// TODO open

/*func ScriptPubKeyToJSON(script *script.Script, includeHex bool) btcjson.ScriptPubKeyResult { // todo complete
	result := btcjson.ScriptPubKeyResult{}

	result.Asm = ScriptToAsmStr(script, includeHex)
	if includeHex {
		result.Hex = hex.EncodeToString(script.GetData())
	}

	t, addresses, required, ok := script.ExtractDestinations(script)
	if !ok {
		result.Type = script.GetTxnOutputType(t)
		return result
	}

	result.ReqSigs = required
	result.Type = script.GetTxnOutputType(t)

	result.Addresses = make([]string, 0, len(addresses))
	for _, address := range addresses {
		result.Addresses = append(result.Addresses, address.String())
	}

	return result
}*///TODO open

/*func GetTransaction(hash *util.Hash, allowSlow bool) (*tx.Tx, *util.Hash, bool) {
	entry := mempool.Gpool.FindTx(*hash) // todo realize: in mempool get *core.Tx by hash
	if entry != nil {
		return entry.Tx, nil, true
	}

	if chain.GTxIndex {
		chain.GBlockTree.ReadTxIndex(hash)
		//blockchain.OpenBlockFile(, true)
		// todo complete
	}

	// use coin database to locate block that contains transaction, and scan it
	var indexSlow *blockindex.BlockIndex
	if allowSlow {
		coin := utxo2.AccessByTxid(utxo.GetUtxoCacheInstance(), hash)
		if !coin.IsSpent() {
			indexSlow = chain.GetInstance.GetIndex(int(coin.GetHeight())) // todo realise : get *BlockIndex by height
		}
	}

	if indexSlow != nil {
		var bk *block.Block
		if chain.ReadBlockFromDisk(bk, indexSlow, consensus.ActiveNetParams) {
			for _, item := range bk.Txs {
				if *hash == item.GetHash() {
					return item, &indexSlow.BlockHash, true
				}
			}
		}
	}

	return nil, nil, false
}*///TODO open

func handleCreateRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.CreateRawTransactionCmd)

	var transaction *tx.Tx
	// Validate the locktime, if given.
	transaction = tx.NewTx(uint32(*c.LockTime), 0)
	if c.LockTime != nil &&
		(*c.LockTime < 0 || *c.LockTime > int64(script.SequenceFinal)) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Locktime out of range",
		}
	}

	for _, input := range c.Inputs {
		hash, err := util.GetHashFromStr(input.Txid)
		if err != nil {
			return nil, rpcDecodeHexError(input.Txid)
		}

		if input.Vout < 0 {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrInvalidParameter,
				Message: "Invalid parameter, vout must be positive",
			}
		}

		sequence := uint32(math.MaxUint32)
		if transaction.GetLockTime() != 0 {
			sequence = math.MaxUint32 - 1
		}

		// todo lack handle with sequence parameter(optional), is reasonable?
		in := txin.NewTxIn(outpoint.NewOutPoint(*hash, input.Vout), &script.Script{}, sequence)
		transaction.AddTxIn(in)
	}

	for address, cost := range c.Amounts {
		// todo do not support the key named 'data' in btcd
		addr, err := script.AddressFromString(address)
		if err != nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid Bitcoin address: " + address,
			}
		}

		outValue := int64(cost * 1e8)
		if !amount.MoneyRange(amount.Amount(outValue)) {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrInvalidParameter,
				Message: "Invalid amount",
			}
		}
		out := txout.NewTxOut(amount.Amount(outValue), script.NewScriptRaw(addr.EncodeToPubKeyHash()))
		transaction.AddTxOut(out)
	}

	buf := bytes.NewBuffer(nil)
	transaction.Serialize(buf)

	return hex.EncodeToString(buf.Bytes()), nil
}

func handleDecodeRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.DecodeRawTransactionCmd)

		// Deserialize the transaction.
		serializedTx, err := hex.DecodeString(c.HexTx)
		if err != nil {
			return nil, rpcDecodeHexError(c.HexTx)
		}

		var transaction tx.Tx
		err = transaction.Unserialize(bytes.NewReader(serializedTx))
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCDeserialization,
				Message: "TX decode failed: " + err.Error(),
			}
		}

		// Create and return the result.
		txReply := btcjson.TxRawDecodeResult{
			Txid:     transaction.Hash.String(),
			Hash:     transaction.Hash.String(),
			Size:     transaction.SerializeSize(),
			Version:  transaction.GetVersion(),
			Locktime: transaction.GetLockTime(),
			Vin:      createVinList(&transaction),
			Vout:     createVoutList(&transaction, consensus.ActiveNetParams),
		}

		return txReply, nil*/
	return nil, nil
}

func handleDecodeScript(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.DecodeScriptCmd)

		// Convert the hex script to bytes.
		scriptByte, err := hex.DecodeString(c.HexScript)
		if err != nil {
			return nil, rpcDecodeHexError(c.HexScript)
		}
		st := script.NewScriptRaw(scriptByte)

		ret := ScriptPubKeyToJSON(st, false)

		if ret.Type != "scripthash" {
			// P2SH cannot be wrapped in a P2SH. If this script is already a P2SH,
			// don't return the address for a P2SH of the P2SH.
			ret.P2SH = EncodeDestination(scriptByte) // todo realise
		}

		return ret, nil*/// TODO open
	return nil, nil
}

func handleSendRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SendRawTransactionCmd)

	buf := bytes.NewBufferString(c.HexTx)
	transaction := tx.Tx{}
	err := transaction.Unserialize(buf)
	if err != nil {
		return nil, rpcDecodeHexError(c.HexTx)
	}

	hash := transaction.GetHash()

	// todo open
	//maxRawTxFee := mining.MaxTxFee
	//if c.AllowHighFees != nil && *c.AllowHighFees {
	//	maxRawTxFee = 0
	//}

	view := utxo.GetUtxoCacheInstance()
	var haveChain bool
	for i := 0; !haveChain && i < transaction.GetOutsCount(); i++ {
		existingCoin := view.GetCoin(outpoint.NewOutPoint(hash, uint32(i)))
		haveChain = !existingCoin.IsSpent()
	}

	entry := mempool.GetInstance().FindTx(hash)
	if entry == nil && !haveChain {
		err = mempool2.AcceptTxToMemPool(&transaction)
		if err != nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "mempool reject the specified transaction for undefined reason",
			}
		}
	}

	txInvMsg := wire.NewInvVect(wire.InvTypeTx, &hash)
	_, err = server.ProcessForRpc(txInvMsg)
	if err != nil {
		return nil, btcjson.ErrRPCInternal
	}

	return hash.String(), nil
}

var mapSigHashValues = map[string]int{
	"ALL":                        crypto.SigHashAll,
	"ALL|ANYONECANPAY":           crypto.SigHashAll | crypto.SigHashAnyoneCanpay,
	"ALL|FORKID":                 crypto.SigHashAll | crypto.SigHashForkID,
	"ALL|FORKID|ANYONECANPAY":    crypto.SigHashAll | crypto.SigHashForkID | crypto.SigHashAnyoneCanpay,
	"NONE":                       crypto.SigHashNone,
	"NONE|ANYONECANPAY":          crypto.SigHashNone | crypto.SigHashAnyoneCanpay,
	"NONE|FORKID":                crypto.SigHashNone | crypto.SigHashForkID,
	"NONE|FORKID|ANYONECANPAY":   crypto.SigHashNone | crypto.SigHashForkID | crypto.SigHashAnyoneCanpay,
	"SINGLE":                     crypto.SigHashSingle,
	"SINGLE|ANYONECANPAY":        crypto.SigHashSingle | crypto.SigHashAnyoneCanpay,
	"SINGLE|FORKID":              crypto.SigHashSingle | crypto.SigHashForkID,
	"SINGLE|FORKID|ANYONECANPAY": crypto.SigHashSingle | crypto.SigHashForkID | crypto.SigHashAnyoneCanpay,
}

func handleSignRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SignRawTransactionCmd)

	txData, err := hex.DecodeString(c.RawTx)
	if err != nil {
		return nil, rpcDecodeHexError(c.RawTx)
	}

	transactions := make([]*tx.Tx, 0)
	r := bytes.NewReader(txData)
	for r.Len() > 0 {
		transaction := tx.Tx{}
		err = transaction.Unserialize(r)
		if err != nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCDeserializationError,
				Message: "Tx decode failed",
			}
		}
		transactions = append(transactions, &transaction)
	}

	if len(transactions) == 0 {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCDeserializationError,
			Message: "Missing transaction",
		}
	}

	// mergedTx will end up with all the signatures; it starts as a clone of the rawtx
	//mergedTx := transactions[0]
	// todo Fetch previous transactions (inputs) to cache <FetchCoin() function>
	// todo do not support this feature at current utxo version

	givenKeys := false
	keyStore := make([]*crypto.PrivateKey, 0)
	scriptStore := make([]*script.Script, 0)
	if c.PrivKeys != nil {
		givenKeys = true
		for _, key := range *c.PrivKeys {
			privKey, err := crypto.DecodePrivateKey(key)
			if err != nil {
				return nil, btcjson.RPCError{
					Code:    btcjson.RPCInvalidAddressOrKey,
					Message: "Invalid private key",
				}
			}

			keyStore = append(keyStore, privKey)
		}
	}

	for _, input := range *c.Inputs {
		hash, err := util.GetHashFromStr(input.Txid)
		if err != nil {
			return nil, rpcDecodeHexError(input.Txid)
		}

		if input.Vout < 0 {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCDeserializationError,
				Message: "vout must be positive",
			}
		}
		out := outpoint.NewOutPoint(*hash, input.Vout)

		scriptPubKey, err := hex.DecodeString(input.ScriptPubKey)
		if err != nil {
			return nil, rpcDecodeHexError(input.ScriptPubKey)
		}

		view := utxo.GetUtxoCacheInstance()
		coin := view.GetCoin(out)

		coinOut := coin.GetTxOut()
		if !coin.IsSpent() && !coinOut.GetScriptPubKey().IsEqual(script.NewScriptRaw(scriptPubKey)) {
			return nil, btcjson.RPCError{
				Code: btcjson.RPCDeserializationError,
				Message: "Previous output scriptPubKey mismatch:\n" +
					ScriptToAsmStr(coinOut.GetScriptPubKey(), false) +
					"\nvs:\n" + ScriptToAsmStr(script.NewScriptRaw(scriptPubKey), false),
			}
		}

		txOut := txout.NewTxOut(0, script.NewScriptRaw(scriptPubKey))
		if input.Amount != nil {
			cost := int64(*input.Amount) * util.COIN
			if !amount.MoneyRange(amount.Amount(cost)) {
				return nil, btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParameter,
					Message: "Amount out of range",
				}
			}
			txOut.SetValue(amount.Amount(cost))
		} else {
			// amount param is required in replay-protected txs.
			// Note that we must check for its presence here rather
			// than use RPCTypeCheckObj() above, since UniValue::VNUM
			// parser incorrectly parses numerics with quotes, eg
			// "3.12" as a string when JSON allows it to also parse
			// as numeric. And we have to accept numerics with quotes
			// because our own dogfood (our rpc results) always
			// produces decimal numbers that are quoted
			// eg getbalance returns "3.14152" rather than 3.14152
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: "Missing amount",
			}
		}

		// todo confirm
		coinsMap := utxo.NewEmptyCoinsMap()
		coinsMap.AddCoin(out, utxo.NewCoin(txOut, 1, false))
		view.UpdateCoins(coinsMap, hash)

		// If redeemScript given and not using the local wallet (private
		// keys given), add redeemScript to the tempKeystore so it can be
		// signed:
		if givenKeys && script.NewScriptRaw(scriptPubKey).IsPayToScriptHash() {
			if input.RedeemScript != "" {
				rsData, err := hex.DecodeString(input.RedeemScript)
				if err != nil {
					return nil, rpcDecodeHexError(input.RedeemScript)
				}
				scriptStore = append(scriptStore, script.NewScriptRaw(rsData))
			}
		}
	}

	hashType, ok := mapSigHashValues[*c.Flags]
	if !ok {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Invalid sighash param",
		}
	}

	if hashType&crypto.SigHashForkID == 0 {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Signature must use SIGHASH_FORKID",
		}
	}

	hashSingle := hashType & ^(crypto.SigHashAnyoneCanpay | crypto.SigHashForkID) == crypto.SigHashSingle

	errors := make([]*btcjson.SignRawTransactionError, 0)
	for index, in := range transactions[0].GetIns() {
		view := utxo.GetUtxoCacheInstance()
		coin := view.GetCoin(in.PreviousOutPoint)
		if coin.IsSpent() {
			errors = append(errors, TxInErrorToJSON(in, "Input not found or already spent"))
			continue
		}

		//prevPubKey := coin.GetTxOut().GetScriptPubKey()
		//cost := coin.GetTxOut().GetValue()

		if !hashSingle || index < transactions[0].GetOutsCount() {
			// todo make and check signature
		}

	}

	complete := len(errors) == 0
	buf := bytes.NewBuffer(nil)
	transactions[0].Serialize(buf)
	return btcjson.SignRawTransactionResult{
		Hex:      hex.EncodeToString(buf.Bytes()),
		Complete: complete,
		Errors:   errors,
	}, err
}

func TxInErrorToJSON(in *txin.TxIn, errorMessage string) *btcjson.SignRawTransactionError {
	return &btcjson.SignRawTransactionError{
		TxID:      in.PreviousOutPoint.Hash.String(),
		Vout:      in.PreviousOutPoint.Index,
		ScriptSig: hex.EncodeToString(in.GetScriptSig().GetData()),
		Sequence:  in.Sequence,
		Error:     errorMessage,
	}
}

func handleGetTxoutProof(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.GetTxOutProofCmd)

		setTxIds := set.New() // element type: util.Hash
		var oneTxId util.Hash
		txIds := c.TxIDs

		for idx := 0; idx < len(txIds); idx++ {
			txId := txIds[idx]
			hash, err := util.GetHashFromStr(txId)
			if len(txId) != 64 || err != nil {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParameter,
					Message: "Invalid txid " + txId,
				}
			}

			if setTxIds.Has(*hash) {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParameter,
					Message: "Invalid parameter, duplicated txid: " + txId,
				}
			}
			setTxIds.Add(*hash)
			oneTxId = *hash
		}

		var bindex *blockindex.BlockIndex
		var hashBlock *util.Hash
		if c.BlockHash != nil {
			var err error
			hashBlock, err = util.GetHashFromStr(*c.BlockHash)
			if err != nil {
				return nil, rpcDecodeHexError(*c.BlockHash)
			}

			bindex = chain.GetInstance.FindBlockIndex(*hashBlock)
			if bindex == nil {
				return nil, btcjson.RPCError{
					Code:    btcjson.RPCInvalidAddressOrKey,
					Message: "Block not found",
				}
			}
		} else {
			view := utxo.GetUtxoCacheInstance()
			coin := utxo2.AccessByTxid(view, &oneTxId)
			if !coin.IsSpent() && coin.GetHeight() > 0 && int(coin.GetHeight()) <= chain.GetInstance.Height() {
				bindex = chain.GetInstance.GetIndex(int(coin.GetHeight()))
			}
		}

		if bindex == nil {
			tx, hashBlock, ok := GetTransaction(&oneTxId, false)
			if !ok || hashBlock.IsNull() {
				return nil, btcjson.RPCError{
					Code:    btcjson.RPCInvalidAddressOrKey,
					Message: "Transaction not yet in block",
				}
			}

			bindex = chain.GetInstance.FindBlockIndex(*hashBlock)
			if bindex == nil {
				return nil, btcjson.RPCError{
					Code:    btcjson.RPCInternalError,
					Message: "Transaction index corrupt",
				}
			}
		}

		bk, ok := ReadBlockFromDisk(bindex)
		if !ok {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCInternalError,
				Message: "Can not read block from disk",
			}
		}

		found := 0
		for _, transaction := range bk.Txs {
			if setTxIds.Has(transaction.GetHash()) {
				found++
			}
		}

		if found != setTxIds.Size() {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCInvalidAddressOrKey,
				Message: "(Not all) transactions not found in specified block",
			}
		}

		mb := mblock.NewMerkleBlock(bk, setTxIds)
		buf := bytes.NewBuffer(nil)
		mb.Serialize(buf)
		return hex.EncodeToString(buf.Bytes()), nil*///TODO open
	return nil, nil
}

func handleVerifyTxoutProof(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.VerifyTxoutProofCmd)

		b, err := hex.DecodeString(c.Proof)
		if err != nil {
			return nil, rpcDecodeHexError(c.Proof)
		}

		mb := mblock.MerkleBlock{}
		err = mb.Unserialize(bytes.NewReader(b))
		if err != nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCDeserializationError,
				Message: "MerkleBlock Unserialize error",
			}
		}

		matches := make([]util.Hash, 0)
		items := make([]int, 0)
		if mb.Txn.ExtractMatches(matches, items).IsEqual(&mb.Header.MerkleRoot) {
			return nil, nil
		}

		bindex := LookupBlockIndex(mb.Header.GetHash()) // todo realize
		if bindex == nil || !chain.GetInstance.Contains(bindex) {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCInvalidAddressOrKey,
				Message: "Block not found in chain",
			}
		}

		ret := make([]string, 0, len(matches))
		for _, hash := range matches {
			ret = append(ret, hash.String())
		}
		return ret, nil*///TODO open
	return nil, nil
}

func registeRawTransactionRPCCommands() {
	for name, handler := range rawTransactionHandlers {
		appendCommand(name, handler)
	}
}
