package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcboost/copernicus/internal/btcjson"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

var blockchainHandlers = map[string]commandHandler{
	"getblockchaininfo":     handleGetBlockChainInfo,
	"getbestblockhash":      handleGetBestBlockHash, // complete
	"getblockcount":         handleGetBlockCount,    // complete
	"getblock":              handleGetBlock,         // complete
	"getblockhash":          handleGetBlockHash,     // complete
	"getblockheader":        handleGetBlockHeader,   // complete
	"getchaintips":          handleGetChainTips,
	"getdifficulty":         handleGetDifficulty,         //complete
	"getmempoolancestors":   handleGetMempoolAncestors,   // complete
	"getmempooldescendants": handleGetMempoolDescendants, //complete
	"getmempoolentry":       handleGetMempoolEntry,       // complete
	"getmempoolinfo":        handleGetMempoolInfo,        // complete
	"getrawmempool":         handleGetRawMempool,         // complete
	"gettxout":              handleGetTxOut,              // complete
	"gettxoutsetinfo":       handleGetTxoutSetInfo,
	"pruneblockchain":       handlePruneBlockChain, //complete
	"verifychain":           handleVerifyChain,     //complete
	"preciousblock":         handlePreciousblock,   //complete

	/*not shown in help*/
	"invalidateblock":    handlInvalidateBlock,  //complete
	"reconsiderblock":    handleReconsiderBlock, //complete
	"waitfornewblock":    handleWaitForNewBlock,
	"waitforblock":       handleWaitForBlock,
	"waitforblockheight": handleWaitForBlockHeight,
}

func handleGetBlockChainInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {

	/*	// Obtain a snapshot of the current best known blockchain state. We'll
		// populate the response to this call primarily from this snapshot.
		var headers int32
		if chain.GIndexBestHeader != nil {
			headers = int32(chain.GIndexBestHeader.Height)
		} else {
			headers = -1
		}


		tip := chain.GlobalChain.Tip()
		chainInfo := &btcjson.GetBlockChainInfoResult{
			//Chain:         Params().NetworkingIDString(),            // TODO
			Blocks:        int32(chain.GlobalChain.Height()),
			Headers:       headers,
			BestBlockHash: tip.GetBlockHash().ToString(),
			Difficulty:    getDifficulty(tip),
			MedianTime:    tip.GetMedianTimePast(),
			//VerificationProgress: chain.GuessVerificationProgress(Params().TxData(),
			//	chain.GlobalChain.Tip())            // TODO
			ChainWork:     tip.ChainWork.String(),
			Pruned:        false,
			Bip9SoftForks: make(map[string]*btcjson.Bip9SoftForkDescription),
		}

		// Next, populate the response with information describing the current
		// status of soft-forks deployed via the super-majority block
		// signalling mechanism.
		height := chainSnapshot.Height
		chainInfo.SoftForks = []*btcjson.SoftForkDescription{
			{
				ID:      "bip34",
				Version: 2,
				Reject: struct {
					Status bool `json:"status"`
				}{
					Status: height >= params.BIP0034Height,
				},
			},
			{
				ID:      "bip66", f
				Version: 3,
				Reject: struct {
					Status bool `json:"status"`
				}{
					Status: height >= params.BIP0066Height,
				},
			},
			{
				ID:      "bip65",
				Version: 4,
				Reject: struct {
					Status bool `json:"status"`
				}{
					Status: height >= params.BIP0065Height,
				},
			},
		}

		// Finally, query the BIP0009 version bits state for all currently
		// defined BIP0009 soft-fork deployments.
		for deployment, deploymentDetails := range params.Deployments {
			// Map the integer deployment ID into a human readable
			// fork-name.
			var forkName string
			switch deployment {
			case chaincfg.DeploymentTestDummy:
				forkName = "dummy"

			case chaincfg.DeploymentCSV:
				forkName = "csv"

			case chaincfg.DeploymentSegwit:
				forkName = "segwit"

			default:
				return nil, &btcjson.RPCError{
					Code: btcjson.ErrRPCInternal.Code,
					Message: fmt.Sprintf("Unknown deployment %v "+
						"detected", deployment),
				}
			}

			// Query the chain for the current status of the deployment as
			// identified by its deployment ID.
			deploymentStatus, err := chain.ThresholdState(uint32(deployment))
			if err != nil {
				context := "Failed to obtain deployment status"
				return nil, internalRPCError(err.Error(), context)
			}

			// Attempt to convert the current deployment status into a
			// human readable string. If the status is unrecognized, then a
			// non-nil error is returned.
			statusString, err := softForkStatus(deploymentStatus)
			if err != nil {
				return nil, &btcjson.RPCError{
					Code: btcjson.ErrRPCInternal.Code,
					Message: fmt.Sprintf("unknown deployment status: %v",
						deploymentStatus),
				}
			}

			// Finally, populate the soft-fork description with all the
			// information gathered above.
			chainInfo.Bip9SoftForks[forkName] = &btcjson.Bip9SoftForkDescription{
				Status:    strings.ToLower(statusString),
				Bit:       deploymentDetails.BitNumber,
				StartTime: int64(deploymentDetails.StartTime),
				Timeout:   int64(deploymentDetails.ExpireTime),
			}
		}

		return chainInfo, nil
	*/
	return nil, nil
}

func handleGetBestBlockHash(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return chain.GlobalChain.Tip().GetBlockHash().String(), nil
}

func handleGetBlockCount(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return chain.GlobalChain.Height(), nil
}

func handleGetBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
/*	c := cmd.(*btcjson.GetBlockCmd)

	// Load the raw block bytes from the database.
	hash, err := util.GetHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}

	index := chain.GlobalChain.FindBlockIndex(*hash)
	if index == nil {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Block not found",
		}
	}

	if chain.GHavePruned && !index.HaveData() && index.TxCount > 0 {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Block not available (pruned data)",
		}
	}

	block := block.Block{}
	if chain.ReadBlockFromDisk(&block, index, consensus.ActiveNetParams) {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Block not found on disk",
		}
	}

	if !*c.Verbose {
		buf := bytes.NewBuffer(nil)
		block.Serialize(buf)
		strHex := hex.EncodeToString(buf.Bytes())
		return strHex, nil
	}

	return blockToJSON(&block, index, false), nil*/           //TODO open
	return nil, nil
}

/*func blockToJSON(block *block.Block, index *blockindex.BlockIndex, txDetails bool) *btcjson.GetBlockVerboseResult {
	confirmations := -1
	// Only report confirmations if the block is on the main chain
	if chain.GlobalChain.Contains(index) {
		confirmations = chain.GlobalChain.Height() - index.Height + 1
	}

	txs := make([]btcjson.TxRawResult, len(block.Txs))
	for i, tx := range block.Txs {
		// todo: I think GetHash() function should return the pointer of hash install of the value
		hash := block.Header.GetHash()
		rawTx, err := createTxRawResult(tx, &hash, consensus.ActiveNetParams)
		if err != nil {
			return nil
		}
		txs[i] = *rawTx
	}

	var previousHash string
	if index.Prev != nil {
		previousHash = index.Prev.BlockHash.String()
	}

	var nextBlockHash string
	next := chain.GlobalChain.Next(index)
	if next != nil {
		nextBlockHash = next.BlockHash.String()
	}
	return &btcjson.GetBlockVerboseResult{
		Hash:          index.GetBlockHash().String(),
		Confirmations: uint64(confirmations),
		Size:          block.SerializeSize(),
		Height:        index.Height,
		Version:       block.Header.Version,
		MerkleRoot:    block.Header.MerkleRoot.String(),
		Tx:            txs,
		Time:          int64(block.Header.Time),
		Mediantime:    index.GetMedianTimePast(),
		Nonce:         block.Header.Nonce,
		Bits:          fmt.Sprintf("%08x", block.Header.Bits),
		Difficulty:    getDifficulty(index),
		ChainWork:     index.ChainWork.String(), // todo check
		PreviousHash:  previousHash,
		NextHash:      nextBlockHash,
	}
}*/                  // TODO open

func handleGetBlockHash(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
/*
	c := cmd.(*btcjson.GetBlockHashCmd)

	height := c.Height
	if height < 0 || height > chain.GlobalChain.Height() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCOutOfRange,
			Message: "Block number out of range",
		}
	}

	blockIndex := chain.GlobalChain.GetSpecIndex(height) // todo realise

	return blockIndex.BlockHash, nil*/           //TODO open
	return nil, nil
}

func handleGetBlockHeader(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockHeaderCmd)

	// Fetch the header from chain.
	hash, err := util.GetHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}
	blockIndex := chain.GlobalChain.FindBlockIndex(*hash)

	if blockIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}

	// When the verbose flag is set false
	if c.Verbose != nil && !*c.Verbose {
		var headerBuf bytes.Buffer
		err := blockIndex.Header.Serialize(&headerBuf)
		if err != nil {
			context := "Failed to serialize block header"
			return nil, internalRPCError(err.Error(), context)
		}
		return hex.EncodeToString(headerBuf.Bytes()), nil
	}

	//best := s.cfg.Chain.BestSnapshot()
	best := chain.GlobalChain.Tip()
	confirmations := -1
	// Only report confirmations if the block is on the main chain
	if chain.GlobalChain.Contains(blockIndex) {
		confirmations = best.Height - blockIndex.Height + 1
	}

	var previousblockhash string
	if blockIndex.Prev != nil {
		previousblockhash = blockIndex.Prev.BlockHash.String()
	}

	var nextblockhash string
	next := chain.GlobalChain.Next(blockIndex)
	if next != nil {
		nextblockhash = next.BlockHash.String()
	}

	blockHeaderReply := btcjson.GetBlockHeaderVerboseResult{
		Hash:          c.Hash,
		Confirmations: uint64(confirmations),
		Height:        int32(blockIndex.Height),
		Version:       blockIndex.Header.Version,
		VersionHex:    fmt.Sprintf("%08x", blockIndex.Header.Version),
		MerkleRoot:    blockIndex.Header.MerkleRoot.String(),
		Time:          blockIndex.Header.Time,
		Mediantime:    blockIndex.GetMedianTimePast(),
		Nonce:         uint64(blockIndex.Header.Nonce),
		Bits:          fmt.Sprintf("%8x", blockIndex.Header.Bits),
		Difficulty:    getDifficulty(blockIndex),
		Chainwork:     blockIndex.ChainWork.Text(16),
		PreviousHash:  previousblockhash,
		NextHash:      nextblockhash,
	}
	return blockHeaderReply, nil
}

func handleGetChainTips(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func getDifficulty(bi *blockindex.BlockIndex) float64 {
	if bi == nil {
		return 1.0
	}
	return getDifficultyFromBits(bi.GetBlockHeader().Bits)
}

// getDifficultyRatio returns the proof-of-work difficulty as a multiple of the
// minimum difficulty using the passed bits field from the header of a block.
func getDifficultyFromBits(bits uint32) float64 {
	shift := bits >> 24 & 0xff
	diff := 0x0000ffff / float64(bits&0x00ffffff)

	for shift < 29 {
		diff *= 256
		shift++
	}

	for shift > 29 {
		diff /= 256
		shift--
	}

	return diff
}

func handleGetDifficulty(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := chain.GlobalChain.Tip()
	return getDifficulty(best), nil
}

func handleGetMempoolAncestors(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMempoolAncestorsCmd)
	hash, err := util.GetHashFromStr(c.TxID)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "the string " + c.TxID + " is not a standard hash",
		}
	}
	entry := mempool.Gpool.FindTx(*hash)
	if entry == nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	h := entry.Tx.TxHash()
	txSet := mempool.Gpool.CalculateMemPoolAncestors(&h)

	if !c.Verbose {
		s := make([]string, len(txSet))
		i := 0
		for index := range txSet {
			s[i] = index.Tx.Hash.String()
			i++
		}
		return s, nil
	}

	infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
	for index := range txSet {
		hash := index.Tx.Hash
		infos[hash.String()] = entryToJSON(index)
	}
	return infos, nil
}

func entryToJSON(entry *mempool.TxEntry) *btcjson.GetMempoolEntryRelativeInfoVerbose {
	result := btcjson.GetMempoolEntryRelativeInfoVerbose{}
	result.Size = entry.TxSize
	result.Fee = valueFromAmount(entry.TxFee)
	result.ModifiedFee = valueFromAmount(entry.SumFeeWithAncestors) // todo check: GetModifiedFee() is equal to SumFeeWithAncestors
	result.Time = entry.GetTime()
	result.Height = entry.TxHeight
	// remove priority at current version
	result.StartingPriority = 0
	result.CurrentPriority = 0
	result.DescendantCount = entry.SumTxCountWithDescendants
	result.DescendantSize = entry.SumSizeWithDescendants
	result.DescendantFees = entry.SumFeeWithDescendants
	result.AncestorCount = entry.SumTxCountWithAncestors
	result.AncestorSize = entry.SumSizeWitAncestors
	result.AncestorFees = entry.SumFeeWithAncestors

	setDepends := make([]string, 0)
	for _, in := range entry.Tx.GetIns() {
		if txItem := mempool.Gpool.FindTx(in.PreviousOutPoint.Hash); txItem != nil {
			setDepends = append(setDepends, in.PreviousOutPoint.Hash.String())
		}
	}
	result.Depends = setDepends

	return &result
}

func handleGetMempoolDescendants(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMempoolDescendantsCmd)

	hash, err := util.GetHashFromStr(c.TxID)
	if err != nil {
		return nil, rpcDecodeHexError(c.TxID)
	}

	entry := mempool.Gpool.FindTx(*hash)
	if entry == nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	descendants := mempool.Gpool.CalculateDescendants(hash)
	// CTxMemPool::CalculateDescendants will include the given tx
	delete(descendants, entry)

	if !c.Verbose {
		des := make([]string, 0)
		for item := range descendants {
			des = append(des, item.Tx.Hash.String())
		}
		return des, nil
	}

	infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
	infos[entry.Tx.Hash.String()] = entryToJSON(entry)
	return infos, nil
}

func handleGetMempoolEntry(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMempoolEntryCmd)

	hash, err := util.GetHashFromStr(c.TxID)
	if err != nil {
		return nil, rpcDecodeHexError(c.TxID)
	}

	mempool.Gpool.Lock()
	defer mempool.Gpool.Unlock()

	entry := mempool.Gpool.FindTx(*hash)
	if entry == nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	return entryToJSON(entry), nil
}

func handleGetMempoolInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	maxMempool := util.GetArg("-maxmempool", int64(tx.DefaultMaxMemPoolSize))
	ret := &btcjson.GetMempoolInfoResult{
		Size:       mempool.Gpool.Size(),
		Bytes:      mempool.Gpool.GetPoolAllTxSize(),
		Usage:      mempool.Gpool.GetPoolUsage(),
		MaxMempool: maxMempool,
		//MempoolMinFee: valueFromAmount(mempool.GetMinFee(maxMempool)),		// todo realise
	}

	return ret, nil
}

func valueFromAmount(sizeLimit int64) string {
	sign := sizeLimit < 0
	var nAbs int64
	if sign {
		nAbs = -sizeLimit
	} else {
		nAbs = sizeLimit
	}

	quotient := nAbs / util.COIN
	remainder := nAbs % util.COIN

	if sign {
		return fmt.Sprintf("-%d.%08d", quotient, remainder)
	}
	return fmt.Sprintf("%d.%08d", quotient, remainder)
}

func handleGetRawMempool(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetRawMempoolCmd)

	pool := mempool.Gpool
	pool.Lock()
	defer pool.Unlock()

	if *c.Verbose {
		infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
		for hash, entry := range pool.GetAllTxEntry() {
			infos[hash.String()] = entryToJSON(entry)
		}
		return infos, nil
	}

	// CompareEntryByDepthAndScore() txenry in mempool sorted by depth and score
	//return mempool.CompareEntryByDepthAndScore(), nil // todo mempool to realise (open)
	return nil, nil
}

func handleGetTxOut(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
/*	c := cmd.(*btcjson.GetTxOutCmd)

	// Convert the provided transaction hash hex to a Hash.
	hash, err := util.GetHashFromStr(c.Txid)
	if err != nil {
		return nil, rpcDecodeHexError(c.Txid)
	}

	out := outpoint.OutPoint{Hash: *hash, Index: c.Vout}

	coin := &utxo.Coin{}
	coinView := utxo.GetUtxoCacheInstance()
	if *c.IncludeMempool {
		// todo realise CoinsViewMemPool{} in mempool
	} else {
		if c, err := coinView.GetCoin(&out); err != nil || c == nil {
			return nil, err
		}

	}

	bestHash := coinView.GetBestBlock()
	index := chain.GlobalChain.FindBlockIndex(bestHash)

	var confirmations int
	if coin.GetHeight() == mempool.MEMPOOL_HEIGHT {
		confirmations = 0
	} else {
		confirmations = index.Height - int(coin.GetHeight()) + 1
	}

	txout := coin.GetTxOut()
	txOutReply := &btcjson.GetTxOutResult{
		BestBlock:     index.BlockHash.String(),
		Confirmations: int64(confirmations),
		Value:         valueFromAmount(int64(coin.GetAmount())),
		ScriptPubKey:  ScriptPubKeyToJSON(txout.GetScriptPubKey(), true),
		Coinbase:      coin.IsCoinBase(),
	}

	return &txOutReply, nil*/                // TODO open
	return nil, nil
}

func handleGetTxoutSetInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func getPrunMode() (bool, error) {
/*	pruneArg := util.GetArg("-prune", 0)
	if pruneArg < 0 {
		return false, errors.New("Prune cannot be configured with a negative value")
	}*/                 // TODO open
	return true, nil
}

func handlePruneBlockChain(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
/*	pruneMode, err := getPrunMode()

	if err != nil && !pruneMode {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCType,
			Message: fmt.Sprintf("Cannot prune blocks because node is not in prune mode."),
		}
	}

	c := cmd.(*btcjson.PruneBlockChainCmd)
	height := c.Height
	if *height < 0 {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCType,
			Message: fmt.Sprintf("Negative block height."),
		}
	}

	if *height > 1000000000 {
		index := chain.GlobalChain.FindEarliestAtLeast(int64(*height - 72000))
		if index != nil {
			return false, &btcjson.RPCError{
				Code:    btcjson.ErrRPCType,
				Message: fmt.Sprintf("Could not find block with at least the specified timestamp."),
			}
		}
		height = &index.Height
	}

	h := *height
	chainHeight := chain.GlobalChain.Height()
	if chainHeight < consensus.ActiveNetParams.PruneAfterHeight {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: fmt.Sprintf("Blockchain is too short for pruning."),
		}
	} else if h > chainHeight {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: fmt.Sprintf("Blockchain is shorter than the attempted prune height."),
		}
	} /*else if h > chainHeight - MIN_BLOCKS_TO_KEEP {
		h = chainHeight - MIN_BLOCKS_TO_KEEP
	}

	chain.PruneBlockFilesManual(*height)
	return uint64(*height), nil*/                // TODO realise

	return nil, nil
}

// handleVerifyChain implements the verifychain command.
func handleVerifyChain(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
/*	c := cmd.(*btcjson.VerifyChainCmd)

	var checkLevel, checkDepth int32
	if c.CheckLevel != nil {
		checkLevel = *c.CheckLevel
	}
	if c.CheckDepth != nil {
		checkDepth = *c.CheckDepth
	}

	return VerifyDB(consensus.ActiveNetParams, utxo.GetUtxoCacheInstance(), checkLevel, checkDepth), nil*/ // TODO open
}

func handlePreciousblock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
/*	c := cmd.(*btcjson.PreciousBlockCmd)
	hash, err := util.GetHashFromStr(c.BlockHash)
	if err != nil {
		return nil, err
	}
	blockIndex := chain.GlobalChain.FindBlockIndex(*hash)
	if blockIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}
	state := valistate.ValidationState{}
	chain.PreciousBlock(consensus.ActiveNetParams, &state, blockIndex)
	if !state.IsValid() {

	}*/                        // TODO open
	return nil, nil
}

func handlInvalidateBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	//c := cmd.(*btcjson.InvalidateBlockCmd)
	//hash, _ := util.GetHashFromStr(c.BlockHash)
/*	state := valistate.ValidationState{}

	if len(chain.MapBlockIndex.Data) == 0 {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Block not found",
		}

		//blkIndex := chain.MapBlockIndex.Data[*hash]
		//chain.InvalidateBlock()                  // TODO
	}
	if state.IsValid() {
		//chain.ActivateBestChain()        // TODO
	}

	if state.IsInvalid() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDatabase,
			Message: state.GetRejectReason(),
		}
	}*/                  // TODO open

	return nil, nil
}

func handleReconsiderBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
/*	c := cmd.(*btcjson.ReconsiderBlockCmd)
	hash, _ := util.GetHashFromStr(c.BlockHash)

	index := chain.GlobalChain.FindBlockIndex(*hash)
	if index == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Block not found",
		}
	}
	chain.ResetBlockFailureFlags(index)

	state := valistate.ValidationState{}
	chain.ActivateBestChain(consensus.ActiveNetParams, &state, nil)

	if state.IsInvalid() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDatabase,
			Message: state.FormatStateMessage(),
		}
	}*/                 // TODO open
	return nil, nil
}

func handleWaitForNewBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleWaitForBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleWaitForBlockHeight(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func registerBlockchainRPCCommands() {
	for name, handler := range blockchainHandlers {
		appendCommand(name, handler)
	}
}
