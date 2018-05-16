package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/internal/btcjson"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
	"github.com/pkg/errors"
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
		if blockchain.GIndexBestHeader != nil {
			headers = int32(blockchain.GIndexBestHeader.Height)
		} else {
			headers = -1
		}


		tip := blockchain.GChainActive.Tip()
		chainInfo := &btcjson.GetBlockChainInfoResult{
			//Chain:         Params().NetworkingIDString(),            // TODO
			Blocks:        int32(blockchain.GChainActive.Height()),
			Headers:       headers,
			BestBlockHash: tip.GetBlockHash().ToString(),
			Difficulty:    getDifficulty(tip),
			MedianTime:    tip.GetMedianTimePast(),
			//VerificationProgress: blockchain.GuessVerificationProgress(Params().TxData(),
			//	blockchain.GChainActive.Tip())            // TODO
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
	return blockchain.GChainActive.Tip().GetBlockHash().ToString(), nil
}

func handleGetBlockCount(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return blockchain.GChainActive.Height(), nil
}

func handleGetBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {

	c := cmd.(*btcjson.GetBlockCmd)

	// Load the raw block bytes from the database.
	hash, err := utils.GetHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}

	index := blockchain.GChainActive.FetchBlockIndexByHash(hash)
	if index == nil {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Block not found",
		}
	}

	if blockchain.GHavePruned && (index.Status&core.BlockHaveData) == 0 && index.TxCount > 0 {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Block not available (pruned data)",
		}
	}

	block := core.Block{}
	if blockchain.ReadBlockFromDisk(&block, index, msg.ActiveNetParams) {
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

	return blockToJSON(&block, index, false), nil
}

func blockToJSON(block *core.Block, index *core.BlockIndex, txDetails bool) *btcjson.GetBlockVerboseResult {
	confirmations := -1
	// Only report confirmations if the block is on the main chain
	if blockchain.GChainActive.Contains(index) {
		confirmations = blockchain.GChainActive.Height() - index.Height + 1
	}

	txs := make([]btcjson.TxRawResult, len(block.Txs))
	for i, tx := range block.Txs {
		rawTx, err := createTxRawResult(tx, block.Hash, msg.ActiveNetParams)
		if err != nil {
			return nil
		}
		txs[i] = *rawTx
	}

	var previousHash string
	if index.Prev != nil {
		previousHash = index.Prev.BlockHash.ToString()
	}

	var nextBlockHash string
	next := core.ActiveChain.Next(index)
	if next != nil {
		nextBlockHash = next.BlockHash.ToString()
	}
	return &btcjson.GetBlockVerboseResult{
		Hash:          index.GetBlockHash().ToString(),
		Confirmations: uint64(confirmations),
		Size:          block.SerializeSize(),
		Height:        index.Height,
		Version:       block.BlockHeader.Version,
		MerkleRoot:    block.BlockHeader.MerkleRoot.ToString(),
		Tx:            txs,
		Time:          int64(block.BlockHeader.Time),
		Mediantime:    index.GetMedianTimePast(),
		Nonce:         block.BlockHeader.Nonce,
		Bits:          fmt.Sprintf("%08x", block.BlockHeader.Bits),
		Difficulty:    getDifficulty(index),
		ChainWork:     index.ChainWork.String(), // todo check
		PreviousHash:  previousHash,
		NextHash:      nextBlockHash,
	}
}

func handleGetBlockHash(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {

	c := cmd.(*btcjson.GetBlockHashCmd)

	height := c.Height
	if height < 0 || height > blockchain.GChainActive.Height() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCOutOfRange,
			Message: "Block number out of range",
		}
	}

	blockIndex := blockchain.GChainActive.GetSpecIndex(height)

	return blockIndex.GetBlockHash().ToString(), nil
}

func handleGetBlockHeader(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockHeaderCmd)

	// Fetch the header from chain.
	hash, err := utils.GetHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}
	blockIndex := blockchain.GChainActive.FetchBlockIndexByHash(hash) // todo realise: get BlockIndex by hash

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
	best := blockchain.GChainActive.Tip()
	confirmations := -1
	// Only report confirmations if the block is on the main chain
	if blockchain.GChainActive.Contains(blockIndex) {
		confirmations = best.Height - blockIndex.Height + 1
	}

	var previousblockhash string
	if blockIndex.Prev != nil {
		previousblockhash = blockIndex.Prev.BlockHash.ToString()
	}

	var nextblockhash string
	next := blockchain.GChainActive.Next(blockIndex)
	if next != nil {
		nextblockhash = next.BlockHash.ToString()
	}

	blockHeaderReply := btcjson.GetBlockHeaderVerboseResult{
		Hash:          c.Hash,
		Confirmations: uint64(confirmations),
		Height:        int32(blockIndex.Height),
		Version:       blockIndex.Header.Version,
		VersionHex:    fmt.Sprintf("%08x", blockIndex.Header.Version),
		MerkleRoot:    blockIndex.Header.MerkleRoot.ToString(),
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

func getDifficulty(bi *core.BlockIndex) float64 {
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
	best := blockchain.GChainActive.Tip()
	return getDifficulty(best), nil
}

func handleGetMempoolAncestors(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMempoolAncestorsCmd)
	hash, err := utils.GetHashFromStr(c.TxID)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "the string " + c.TxID + " is not a standard hash",
		}
	}
	txEntry, ok := blockchain.GMemPool.PoolData[*hash]
	if !ok {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	noLimit := uint64(math.MaxUint64)
	txSet, err := blockchain.GMemPool.CalculateMemPoolAncestors(txEntry.Tx, noLimit, noLimit, noLimit, noLimit, false)

	if !c.Verbose {
		s := make([]string, len(txSet))
		i := 0
		for index := range txSet {
			s[i] = index.Tx.Hash.ToString()
			i++
		}
		return s, nil
	}

	infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
	for index := range txSet {
		hash := index.Tx.Hash
		infos[hash.ToString()] = entryToJSON(index)
	}
	return infos, nil
}

func entryToJSON(entry *mempool.TxEntry) *btcjson.GetMempoolEntryRelativeInfoVerbose {
	result := btcjson.GetMempoolEntryRelativeInfoVerbose{}
	result.Size = entry.TxSize
	result.Fee = valueFromAmount(entry.TxFee)
	result.ModifiedFee = valueFromAmount(entry.SumFeeWithAncestors) // todo check: GetModifiedFee() is equal to SumFeeWithAncestors
	result.Time = entry.Time
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
	for _, in := range entry.Tx.Ins {
		if _, ok := blockchain.GMemPool.PoolData[in.PreviousOutPoint.Hash]; ok {
			setDepends = append(setDepends, in.PreviousOutPoint.Hash.ToString())
		}
	}
	result.Depends = setDepends

	return &result
}

func handleGetMempoolDescendants(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMempoolDescendantsCmd)

	hash, err := utils.GetHashFromStr(c.TxID)
	if err != nil {
		return nil, rpcDecodeHexError(c.TxID)
	}

	entry, ok := blockchain.GMemPool.PoolData[*hash]
	if !ok {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	descendants := make(map[*mempool.TxEntry]struct{})

	// todo CalculateMemPoolAncestors() and CalculateDescendants() is different API form
	blockchain.GMemPool.CalculateDescendants(entry, descendants)
	// CTxMemPool::CalculateDescendants will include the given tx
	delete(descendants, entry)

	if !c.Verbose {
		des := make([]string, 0)
		for item := range descendants {
			des = append(des, item.Tx.Hash.ToString())
		}
		return des, nil
	}

	infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
	infos[entry.Tx.Hash.ToString()] = entryToJSON(entry)
	return infos, nil
}

func handleGetMempoolEntry(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMempoolEntryCmd)

	hash, err := utils.GetHashFromStr(c.TxID)
	if err != nil {
		return nil, rpcDecodeHexError(c.TxID)
	}

	blockchain.GMemPool.Lock()
	defer blockchain.GMemPool.Unlock()

	entry, ok := blockchain.GMemPool.PoolData[*hash]
	if !ok {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	return entryToJSON(entry), nil
}

func handleGetMempoolInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	maxMempool := utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize))
	ret := &btcjson.GetMempoolInfoResult{
		Size:       len(blockchain.GMemPool.PoolData),
		Bytes:      blockchain.GMemPool.TotalTxSize,
		Usage:      blockchain.GMemPool.GetCacheUsage(),
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

	quotient := nAbs / utils.COIN
	remainder := nAbs % utils.COIN

	if sign {
		return fmt.Sprintf("-%d.%08d", quotient, remainder)
	}
	return fmt.Sprintf("%d.%08d", quotient, remainder)
}

func handleGetRawMempool(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetRawMempoolCmd)

	pool := blockchain.GMemPool
	pool.Lock()
	defer pool.Unlock()

	if *c.Verbose {
		infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
		for hash, entry := range pool.PoolData {
			infos[hash.ToString()] = entryToJSON(entry)
		}
		return infos, nil
	}

	// CompareEntryByDepthAndScore() txenry in mempool sorted by depth and score
	//return mempool.CompareEntryByDepthAndScore(), nil // todo mempool to realise (open)
	return nil, nil
}

func handleGetTxOut(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetTxOutCmd)

	// Convert the provided transaction hash hex to a Hash.
	hash, err := utils.GetHashFromStr(c.Txid)
	if err != nil {
		return nil, rpcDecodeHexError(c.Txid)
	}

	out := core.OutPoint{Hash: *hash, Index: c.Vout}

	coin := &utxo.Coin{}
	if *c.IncludeMempool {
		// todo realise CoinsViewMemPool{} in mempool
	} else {
		if !blockchain.GCoinsTip.GetCoin(&out, coin) {
			return nil, nil
		}

	}

	bestHash := blockchain.GCoinsTip.GetBestBlock()
	index := blockchain.GChainActive.FetchBlockIndexByHash(&bestHash)

	var confirmations int
	if coin.GetHeight() == mempool.MEMPOOL_HEIGHT {
		confirmations = 0
	} else {
		confirmations = index.Height - int(coin.GetHeight()) + 1
	}
	txOutReply := &btcjson.GetTxOutResult{
		BestBlock:     index.BlockHash.ToString(),
		Confirmations: int64(confirmations),
		Value:         valueFromAmount(coin.TxOut.Value),
		ScriptPubKey:  ScriptPubKeyToJSON(coin.TxOut.Script, true),
		Coinbase:      coin.IsCoinBase(),
	}

	return &txOutReply, nil
}

func handleGetTxoutSetInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func getPrunMode() (bool, error) {
	pruneArg := utils.GetArg("-prune", 0)
	if pruneArg < 0 {
		return false, errors.New("Prune cannot be configured with a negative value")
	}
	return true, nil
}

func handlePruneBlockChain(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	pruneMode, err := getPrunMode()

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
		index := blockchain.GChainActive.FindEarliestAtLeast(int64(*height - 72000))
		if index != nil {
			return false, &btcjson.RPCError{
				Code:    btcjson.ErrRPCType,
				Message: fmt.Sprintf("Could not find block with at least the specified timestamp."),
			}
		}
		height = &index.Height
	}

	h := *height
	chainHeight := blockchain.GChainActive.Height()
	if chainHeight < msg.ActiveNetParams.PruneAfterHeight {
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
	}*/ // TODO realise

	blockchain.PruneBlockFilesManual(*height)
	return uint64(*height), nil
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

		err := verifyChain(s, checkLevel, checkDepth)

		return err == nil, nil*/ // TODO realise
	return nil, nil
}

func handlePreciousblock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.PreciousBlockCmd)
	hash, err := utils.GetHashFromStr(c.BlockHash)
	if err != nil {
		return nil, err
	}
	blockIndex := blockchain.GChainActive.FetchBlockIndexByHash(hash)
	if blockIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}
	state := core.ValidationState{}
	blockchain.PreciousBlock(msg.ActiveNetParams, &state, blockIndex)
	if !state.IsValid() {

	}
	return nil, nil
}

func handlInvalidateBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	//c := cmd.(*btcjson.InvalidateBlockCmd)
	//hash, _ := utils.GetHashFromStr(c.BlockHash)
	state := core.ValidationState{}

	if len(blockchain.MapBlockIndex.Data) == 0 {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Block not found",
		}

		//blkIndex := blockchain.MapBlockIndex.Data[*hash]
		//blockchain.InvalidateBlock()                  // TODO
	}
	if state.IsValid() {
		//blockchain.ActivateBestChain()        // TODO
	}

	if state.IsInvalid() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDatabase,
			Message: state.GetRejectReason(),
		}
	}

	return nil, nil
}

func handleReconsiderBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.ReconsiderBlockCmd)
	hash, _ := utils.GetHashFromStr(c.BlockHash)

	blockindex := core.ActiveChain.FetchBlockIndexByHash(hash)
	if blockindex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Block not found",
		}
	}
	blockchain.ResetBlockFailureFlags(blockindex)

	state := core.ValidationState{}
	blockchain.ActivateBestChain(msg.ActiveNetParams, &state, nil)

	if state.IsInvalid() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDatabase,
			Message: blockchain.FormatStateMessage(&state),
		}
	}
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
