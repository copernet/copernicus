package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/versionbits"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/util"
	"gopkg.in/fatih/set.v0"
)

var blockchainHandlers = map[string]commandHandler{
	"getblockchaininfo":     handleGetBlockChainInfo,
	"getbestblockhash":      handleGetBestBlockHash,      // complete
	"getblockcount":         handleGetBlockCount,         // complete
	"getblock":              handleGetBlock,              // complete
	"getblockhash":          handleGetBlockHash,          // complete
	"getblockheader":        handleGetBlockHeader,        // complete
	"getchaintips":          handleGetChainTips,          // partial complete
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
	gChain := chain.GetInstance()
	params := gChain.GetParams()
	tip := gChain.Tip()

	chainInfo := &btcjson.GetBlockChainInfoResult{
		Chain:                params.Name,
		Blocks:               gChain.Height(),
		BestBlockHash:        tip.GetBlockHash().String(),
		Difficulty:           getDifficulty(tip),
		MedianTime:           tip.GetMedianTimePast(),
		VerificationProgress: lchain.GuessVerificationProgress(params.TxData(), tip),
		ChainWork:            tip.ChainWork.Text(16),
		Pruned:               false,
		Bip9SoftForks:        make(map[string]*btcjson.Bip9SoftForkDescription),
		//Headers:            lblockindex.indexBestHeader.Height,   // TODO: NOT support yet
	}

	// Next, populate the response with information describing the current
	// status of soft-forks deployed via the super-majority block
	// signalling mechanism.

	height := tip.Height
	chainInfo.SoftForks = []*btcjson.SoftForkDescription{
		{
			ID:      "bip34",
			Version: 2,
			Reject: struct {
				Status bool `json:"status"`
			}{
				Status: height >= params.BIP34Height,
			},
		},
		{
			ID:      "bip66",
			Version: 3,
			Reject: struct {
				Status bool `json:"status"`
			}{
				Status: height >= params.BIP66Height,
			},
		},
		{
			ID:      "bip65",
			Version: 4,
			Reject: struct {
				Status bool `json:"status"`
			}{
				Status: height >= params.BIP65Height,
			},
		},
	}

	// Finally, query the BIP0009 version bits state for all currently
	// defined BIP0009 soft-fork deployments.
	for i := 0; i < int(consensus.MaxVersionBitsDeployments); i++ {
		pos := consensus.DeploymentPos(i)
		state := versionbits.VersionBitsState(indexPrev, params, pos, versionbits.VBCache)
		forkName := getVbName(pos)

		// Attempt to convert the current deployment status into a
		// human readable string. If the status is unrecognized, then a
		// non-nil error is returned.
		statusString, err := softForkStatus(state)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInternal.Code,
				Message: fmt.Sprintf("unknown deployment status: %v",
					state),
			}
		}

		// Finally, populate the soft-fork description with all the
		// information gathered above.
		deploymentDetails := &params.Deployments[pos]
		chainInfo.Bip9SoftForks[forkName] = &btcjson.Bip9SoftForkDescription{
			Status:    strings.ToLower(statusString),
			Bit:       uint8(deploymentDetails.Bit),
			StartTime: deploymentDetails.StartTime,
			Timeout:   deploymentDetails.Timeout,
		}
	}

	return chainInfo, nil
}

// softForkStatus converts a ThresholdState state into a human readable string
// corresponding to the particular state.
func softForkStatus(state versionbits.ThresholdState) (string, error) {
	switch state {
	case versionbits.ThresholdDefined:
		return "defined", nil
	case versionbits.ThresholdStarted:
		return "started", nil
	case versionbits.ThresholdLockedIn:
		return "lockedin", nil
	case versionbits.ThresholdActive:
		return "active", nil
	case versionbits.ThresholdFailed:
		return "failed", nil
	default:
		return "", fmt.Errorf("unknown deployment state: %v", state)
	}
}

func handleGetBestBlockHash(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return chain.GetInstance().Tip().GetBlockHash().String(), nil
}

func handleGetBlockCount(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return chain.GetInstance().Height(), nil
}

func handleGetBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockCmd)

	// Load the raw block bytes from the database.
	hash, err := util.GetHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}

	blockIndex := chain.GetInstance().FindBlockIndex(*hash)
	if blockIndex == nil {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Block not found",
		}
	}

	pruneState := disk.GetPruneState()
	if pruneState.HavePruned && !blockIndex.HasData() && blockIndex.TxCount > 0 {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Block not available (pruned data)",
		}
	}

	blk, ret := disk.ReadBlockFromDisk(blockIndex, chain.GetInstance().GetParams())
	if !ret {
		return false, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Block not found on disk",
		}
	}

	if c.Verbose != nil && !*c.Verbose {
		blkBuf := bytes.NewBuffer(nil)
		blk.Serialize(blkBuf)
		strHex := hex.EncodeToString(blkBuf.Bytes())
		return strHex, nil
	}

	blockReply := blockToJSON(blk, blockIndex)

	return blockReply, nil
}

func blockToJSON(blk *block.Block, blockIndex *blockindex.BlockIndex) *btcjson.GetBlockVerboseResult {
	confirmations := int32(-1)
	// Only report confirmations if the block is on the main chain
	if chain.GetInstance().Contains(blockIndex) {
		confirmations = chain.GetInstance().TipHeight() - blockIndex.Height + 1
	}

	var previousHash string
	if blockIndex.Prev != nil {
		previousHash = blockIndex.Prev.GetBlockHash().String()
	}

	var nextHash string
	next := chain.GetInstance().Next(blockIndex)
	if next != nil {
		nextHash = next.GetBlockHash().String()
	}

	blockHeader := &blk.Header
	blockReply := &btcjson.GetBlockVerboseResult{
		Hash:          blockIndex.GetBlockHash().String(),
		Confirmations: confirmations,
		Size:          blk.SerializeSize(),
		Height:        blockIndex.Height,
		Version:       blockHeader.Version,
		VersionHex:    strconv.FormatInt(int64(blockHeader.Version), 16),
		MerkleRoot:    blockHeader.MerkleRoot.String(),
		Tx:            make([]string, len(blk.Txs)),
		Time:          int64(blockHeader.Time),
		Mediantime:    blockIndex.GetMedianTimePast(),
		Nonce:         blockHeader.Nonce,
		Bits:          strconv.FormatInt(int64(blockHeader.Bits), 16),
		Difficulty:    getDifficulty(blockIndex),
		ChainWork:     blockIndex.ChainWork.Text(16),
		PreviousHash:  previousHash,
		NextHash:      nextHash,
	}

	for i, tx := range blk.Txs {
		txHash := tx.GetHash()
		blockReply.Tx[i] = txHash.String()
	}
	return blockReply
}

func handleGetBlockHash(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockHashCmd)

	blockIndex := chain.GetInstance().GetIndex(c.Height)
	if blockIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCOutOfRange,
			Message: "Block height out of range",
		}
	}
	return blockIndex.GetBlockHash().String(), nil
}

func handleGetBlockHeader(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockHeaderCmd)

	// Fetch the header from chain.
	hash, err := util.GetHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}
	blockIndex := chain.GetInstance().FindBlockIndex(*hash)

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

	confirmations := int32(-1)
	// Only report confirmations if the block is on the main chain
	if chain.GetInstance().Contains(blockIndex) {
		confirmations = chain.GetInstance().TipHeight() - blockIndex.Height + 1
	}

	var previousblockhash string
	if blockIndex.Prev != nil {
		previousblockhash = blockIndex.Prev.GetBlockHash().String()
	}

	var nextblockhash string
	next := chain.GetInstance().Next(blockIndex)
	if next != nil {
		nextblockhash = next.GetBlockHash().String()
	}

	blockHeaderReply := btcjson.GetBlockHeaderVerboseResult{
		Hash:          c.Hash,
		Confirmations: uint64(confirmations),
		Height:        blockIndex.Height,
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
	// Idea:  the set of chain tips is chainActive.tip, plus orphan blocks which
	// do not have another orphan building off of them.
	// Algorithm:
	//	- Make one pass through mapBlockIndex, picking out the orphan blocks,
	//	  and also storing a set of the orphan block's pprev pointers.
	//  - Iterate through the orphan blocks. If the block isn't pointed to by
	//	  another orphan, it is a chain tip.
	//	- add chainActive.Tip()
	setTips := set.New() // element type:

	// todo add orphan blockindex, lack of chain's support<orphan>
	setTips.Add(chain.GetInstance().Tip())

	ret := btcjson.GetChainTipsResult{
		Tips: make([]btcjson.ChainTipsInfo, 0, setTips.Size()),
	}
	setTips.Each(func(item interface{}) bool {
		bindex := item.(*blockindex.BlockIndex)
		tipInfo := btcjson.ChainTipsInfo{
			Height:    bindex.Height,
			Hash:      bindex.GetBlockHash().String(),
			BranchLen: bindex.Height - chain.GetInstance().FindFork(bindex).Height,
		}

		var status string
		if chain.GetInstance().Contains(bindex) {
			// This block is part of the currently active chain.
			status = "active"
		} else if bindex.IsInvalid() {
			// This block or one of its ancestors is invalid.
			status = "invalid"
		} else if bindex.ChainTxCount == 0 {
			// This block cannot be connected because full block data for it or
			// one of its parents is missing.
			status = "headers-only"
		} else if bindex.IsValid(blockindex.BlockValidScripts) {
			// This block is fully validated, but no longer part of the active
			// chain. It was probably the active block once, but was
			// reorganized.
			status = "valid-fork"
		} else if bindex.IsValid(blockindex.BlockValidTree) {
			// The headers for this block are valid, but it has not been
			// validated. It was probably never part of the most-work chain.
			status = "valid-headers"
		} else {
			// No clue
			status = "unknown"
		}
		tipInfo.Status = status

		ret.Tips = append(ret.Tips, tipInfo)

		return true
	})

	return ret, nil
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
	shift := (bits >> 24) & 0xff
	diff := float64(0x0000ffff) / float64(bits&0x00ffffff)
	const factor = float64(256.0)

	for shift < 29 {
		diff *= factor
		shift++
	}

	for shift > 29 {
		diff /= factor
		shift--
	}

	return diff
}

func handleGetDifficulty(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := chain.GetInstance().Tip()
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
	entry := mempool.GetInstance().FindTx(*hash)
	if entry == nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	h := entry.Tx.GetHash()
	txSet := mempool.GetInstance().CalculateMemPoolAncestorsWithLock(&h)

	if !c.Verbose {
		s := make([]string, len(txSet))
		i := 0
		for index := range txSet {
			hash := index.Tx.GetHash()
			s[i] = hash.String()
			i++
		}
		return s, nil
	}

	infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
	for index := range txSet {
		hash := index.Tx.GetHash()
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
		if txItem := mempool.GetInstance().FindTx(in.PreviousOutPoint.Hash); txItem != nil {
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

	entry := mempool.GetInstance().FindTx(*hash)
	if entry == nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	descendants := mempool.GetInstance().CalculateDescendantsWithLock(hash)
	// CTxMemPool::CalculateDescendants will include the given tx
	delete(descendants, entry)

	if !c.Verbose {
		des := make([]string, 0)
		for item := range descendants {
			hash := item.Tx.GetHash()
			des = append(des, hash.String())
		}
		return des, nil
	}

	infos := make(map[string]*btcjson.GetMempoolEntryRelativeInfoVerbose)
	h := entry.Tx.GetHash()
	infos[h.String()] = entryToJSON(entry)
	return infos, nil
}

func handleGetMempoolEntry(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetMempoolEntryCmd)

	hash, err := util.GetHashFromStr(c.TxID)
	if err != nil {
		return nil, rpcDecodeHexError(c.TxID)
	}

	mempool.GetInstance().Lock()
	defer mempool.GetInstance().Unlock()

	entry := mempool.GetInstance().FindTx(*hash)
	if entry == nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Transaction not in mempool",
		}
	}

	return entryToJSON(entry), nil
}

func handleGetMempoolInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	ret := &btcjson.GetMempoolInfoResult{
		Size:          mempool.GetInstance().Size(),
		Bytes:         mempool.GetInstance().GetPoolAllTxSize(),
		Usage:         mempool.GetInstance().GetPoolUsage(),
		MaxMempool:    mempool.GetInstance().MaxMemPoolSize,
		MempoolMinFee: valueFromAmount(mempool.GetInstance().GetMinFeeRate().SataoshisPerK),
	}
	return ret, nil
}

func valueFromAmount(sizeLimit int64) float64 {
	sign := sizeLimit < 0
	var nAbs int64
	if sign {
		nAbs = -sizeLimit
	} else {
		nAbs = sizeLimit
	}

	quotient := nAbs / util.COIN
	remainder := nAbs % util.COIN

	var strValue string
	if sign {
		strValue = fmt.Sprintf("-%d.%08d", quotient, remainder)
	}
	strValue = fmt.Sprintf("%d.%08d", quotient, remainder)

	result, err := strconv.ParseFloat(strValue, 64)
	if err == nil {
		return 0
	}
	return result
}

func handleGetRawMempool(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetRawMempoolCmd)

	pool := mempool.GetInstance()
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
		index := chain.GetInstance().FindBlockIndex(bestHash)

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

		return &txOutReply, nil*/ // todo open
	return nil, nil
}

func handleGetTxoutSetInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func getPrunMode() (bool, error) {
	/*	pruneArg := util.GetArg("-prune", 0)
		if pruneArg < 0 {
			return false, errors.New("Prune cannot be configured with a negative value")
		}*/ // todo open
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
			index := chain.GetInstance().FindEarliestAtLeast(int64(*height - 72000))
			if index != nil {
				return false, &btcjson.RPCError{
					Code:    btcjson.ErrRPCType,
					Message: fmt.Sprintf("Could not find block with at least the specified timestamp."),
				}
			}
			height = &index.Height
		}

		h := *height
		chainHeight := chain.GetInstance().Height()
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
		return uint64(*height), nil*/ // todo realise

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

		return VerifyDB(consensus.ActiveNetParams, utxo.GetUtxoCacheInstance(), checkLevel, checkDepth), nil*/ // todo open
	return nil, nil
}

func handlePreciousblock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.PreciousBlockCmd)
		hash, err := util.GetHashFromStr(c.BlockHash)
		if err != nil {
			return nil, err
		}
		blockIndex := chain.GetInstance().FindBlockIndex(*hash)
		if blockIndex == nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCBlockNotFound,
				Message: "Block not found",
			}
		}
		state := valistate.ValidationState{}
		chain.PreciousBlock(consensus.ActiveNetParams, &state, blockIndex)
		if !state.IsValid() {

		}*/ // todo open
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
			//chain.InvalidateBlock()                  // todo
		}
		if state.IsValid() {
			//chain.ActivateBestChain()        // todo
		}

		if state.IsInvalid() {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCDatabase,
				Message: state.GetRejectReason(),
			}
		}*/ // todo open

	return nil, nil
}

func handleReconsiderBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.ReconsiderBlockCmd)
		hash, _ := util.GetHashFromStr(c.BlockHash)

		index := chain.GetInstance().FindBlockIndex(*hash)
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
		}*/ // todo open
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
