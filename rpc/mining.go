package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/internal/btcjson"
	"github.com/btcboost/copernicus/mining"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
	"gopkg.in/fatih/set.v0"
)

var miningHandlers = map[string]commandHandler{
	"getnetworkhashps":      handleGetNetWorkhashPS,
	"getmininginfo":         handleGetMiningInfo,
	"prioritisetransaction": handlePrioritisetransaction,
	"getblocktemplate":      handleGetblocktemplate,
	"submitblock":           handleSubmitBlock,
	"generate":              handleGenerate,
	"generatetoaddress":     handleGenerateToAddress,
	"estimatefee":           handleEstimateFee,
	"estimatepriority":      handleEstimatePriority,
	"estimatesmartfee":      handleEstimateSmartFee,
	"estimatesmartpriority": handleEstimateSmartPriority,
}

func handleGetNetWorkhashPS(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetNetworkHashPSCmd)

	lookup := 120
	height := -1
	if c.Blocks != nil {
		lookup = *c.Blocks
	}

	if c.Height != nil {
		height = *c.Height
	}

	block := blockchain.GChainActive.Tip()
	if height > 0 || height < blockchain.GChainActive.Height() {
		block = blockchain.GChainActive.Chain[height]
	}

	if block == nil || block.Height != 0 {
		return 0, nil
	}

	if lookup <= 0 {
		lookup = block.Height%int(msg.ActiveNetParams.DifficultyAdjustmentInterval()) + 1
	}

	if lookup > block.Height {
		lookup = block.Height
	}

	b := block
	minTime := b.GetBlockTime()
	maxTime := minTime
	for i := 0; i < lookup; i++ {
		b = b.Prev
		// time := b.GetBlockTime()
		//minTime = utils.Min(time, minTime)          TODO
		//maxTime = utils.Max(time, maxTime)  		  TODO
	}

	if minTime == maxTime {
		return 0, nil
	}

	workDiff := new(big.Int).Sub(&block.ChainWork, &b.ChainWork)
	timeDiff := int64(maxTime - minTime)

	hashesPerSec := new(big.Int).Div(workDiff, big.NewInt(timeDiff))
	return hashesPerSec, nil
}

func handleGetMiningInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	gnhpsCmd := btcjson.NewGetNetworkHashPSCmd(nil, nil)
	networkHashesPerSecIface, err := handleGetNetWorkhashPS(s, gnhpsCmd,
		closeChan)
	if err != nil {
		return nil, err
	}
	networkHashesPerSec, ok := networkHashesPerSecIface.(int64)
	if !ok {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInternal.Code,
			Message: "networkHashesPerSec is not an int64",
		}
	}

	block := blockchain.GChainActive.Tip()
	result := btcjson.GetMiningInfoResult{
		Blocks:                  int64(block.Height),
		CurrentBlockSize:        mining.GetLastBlockSize(),
		CurrentBlockTx:          mining.GetLastBlockTx(),
		Difficulty:              getDifficulty(block),
		BlockPriorityPercentage: utils.GetArg("-blockprioritypercentage", 0),
		//Errors:              ,                            // TODO
		NetworkHashPS: networkHashesPerSec,
		//PooledTx:           uint64(mempool.Size()),              TODO
		Chain: msg.ActiveNetParams.Name,
	}
	return &result, nil
}

// priority transaction currently disabled
func handlePrioritisetransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

// global variable in package rpc
var (
	transactionsUpdatedLast uint64
	indexPrev               *core.BlockIndex
	start                   int64
	blocktemplate           *mining.BlockTemplate
)

func handleGetblocktemplate(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	// See https://en.bitcoin.it/wiki/BIP_0022 and
	// https://en.bitcoin.it/wiki/BIP_0023 for more details.
	c := cmd.(*btcjson.GetBlockTemplateCmd)
	request := c.Request

	// Set the default mode and override it if supplied.
	mode := "template"
	if request != nil && request.Mode != "" {
		mode = request.Mode
	}

	switch mode {
	case "template":
		return handleGetBlockTemplateRequest(request, closeChan)
	case "proposal":
		return handleGetBlockTemplateProposal(request)
	}

	return nil, &btcjson.RPCError{
		Code:    btcjson.ErrRPCInvalidParameter,
		Message: "Invalid mode",
	}
}

func handleGetBlockTemplateRequest(request *btcjson.TemplateRequest, closeChan <-chan struct{}) (interface{}, error) {
	var maxVersionVb uint32
	setClientRules := set.New()
	if len(request.Rules) > 0 { // todo check
		for _, str := range request.Rules {
			setClientRules.Add(str)
		}
	} else {
		// NOTE: It is important that this NOT be read if versionbits is supported
		maxVersionVb = request.MaxVersion
	}

	// todo handle connMan exception
	if blockchain.IsInitialBlockDownload() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCClientInInitialDownload,
			Message: "Bitcoin is downloading blocks...",
		}
	}

	if request.LongPollID != "" {
		// Wait to respond until either the best block changes, OR a minute has
		// passed and there are more transactions
		//var hashWatchedChain utils.Hash
		//checktxtime := time.Now()
		//transactionsUpdatedLastLP := 0
		// todo complete
	}

	if indexPrev != core.ActiveChain.Tip() ||
		blockchain.GMemPool.TransactionsUpdated != transactionsUpdatedLast &&
			utils.GetMockTime()-start > 5 {

		// Clear pindexPrev so future calls make a new block, despite any
		// failures from here on
		indexPrev = nil
		// Store the pindexBest used before CreateNewBlock, to avoid races
		transactionsUpdatedLast = blockchain.GMemPool.TransactionsUpdated
		indexPrevNew := blockchain.GChainActive.Tip()
		start = utils.GetMockTime()

		// Create new block
		scriptDummy := core.Script{}
		scriptDummy.PushOpCode(core.OP_TRUE)
		ba := mining.NewBlockAssembler(msg.ActiveNetParams)
		blocktemplate = ba.CreateNewBlock()
		if blocktemplate == nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "Out of memory",
			}
		}

		// Need to update only after we know CreateNewBlock succeeded
		indexPrev = indexPrevNew
	}
	block := blocktemplate.Block
	block.UpdateTime(indexPrev)
	block.BlockHeader.Nonce = 0

	return blockTemplateResult(blocktemplate, setClientRules, maxVersionVb, transactionsUpdatedLast)
}

// blockTemplateResult returns the current block template associated with the
// state as a btcjson.GetBlockTemplateResult that is ready to be encoded to JSON
// and returned to the caller.
//
// This function MUST be called with the state locked.
func blockTemplateResult(bt *mining.BlockTemplate, s *set.Set, maxVersionVb uint32, transactionsUpdatedLast uint64) (*btcjson.GetBlockTemplateResult, error) {
	setTxIndex := make(map[utils.Hash]int)
	var i int
	transactions := make([]btcjson.GetBlockTemplateResultTx, 0, len(bt.Block.Txs))
	for _, tx := range bt.Block.Txs {
		txID := tx.TxHash()
		setTxIndex[txID] = i
		i++

		if tx.IsCoinBase() {
			continue
		}

		entry := btcjson.GetBlockTemplateResultTx{}

		dataBuf := bytes.NewBuffer(nil)
		tx.Serialize(dataBuf)
		entry.Data = hex.EncodeToString(dataBuf.Bytes())

		entry.TxID = txID.ToString()
		entry.Hash = txID.ToString()

		deps := make([]int, 0)
		for _, in := range tx.Ins {
			if ele, ok := setTxIndex[in.PreviousOutPoint.Hash]; ok {
				deps = append(deps, ele)
			}
		}
		entry.Depends = deps

		indexInTemplate := i - 1
		entry.Fee = int64(blocktemplate.TxFees[indexInTemplate])
		entry.SigOps = int64(blocktemplate.TxSigOpsCount[indexInTemplate])

		transactions = append(transactions, entry)
	}

	vbAvailable := make(map[string]int)
	rules := make([]string, 0)
	for i := 0; i < int(consensus.MaxVersionBitsDeployments); i++ {
		pos := consensus.DeploymentPos(i)
		state := blockchain.VersionBitsState(indexPrev, msg.ActiveNetParams, pos, blockchain.VBCache)
		switch state {
		case blockchain.ThresholdDefined:
			fallthrough
		case blockchain.ThresholdFailed:
			// Not exposed to GBT at all and break
		case blockchain.ThresholdLockedIn:
			// Ensure bit is set in block version, then fallthrough to get
			// vbavailable set.
			bt.Block.BlockHeader.Version |= int32(blockchain.VersionBitsMask(msg.ActiveNetParams, pos))
			fallthrough
		case blockchain.ThresholdStarted:
			vbinfo := blockchain.VersionBitsDeploymentInfo[pos]
			vbAvailable[getVbName(pos)] = msg.ActiveNetParams.Deployments[pos].Bit
			if !s.Has(vbinfo.Name) {
				if !vbinfo.GbtForce {
					// If the client doesn't support this, don't indicate it
					// in the [default] version
					bt.Block.BlockHeader.Version &= int32(^blockchain.VersionBitsMask(msg.ActiveNetParams, pos))
				}
			}
		case blockchain.ThresholdActive:
			// Add to rules only
			vbinfo := blockchain.VersionBitsDeploymentInfo[pos]
			rules = append(rules, getVbName(pos))
			if !s.Has(vbinfo.Name) {
				// Not supported by the client; make sure it's safe to proceed
				if !vbinfo.GbtForce {
					// If we do anything other than throw an exception here,
					// be sure version/force isn't sent to old clients
					return nil, btcjson.RPCError{
						Code:    btcjson.ErrInvalidParameter,
						Message: fmt.Sprintf("Support for '%s' rule requires explicit client support", vbinfo.Name),
					}
				}
			}
		}

	}
	mutable := make([]string, 3, 4)
	mutable[0] = "time"
	mutable[1] = "transactions"
	mutable[2] = "prevblock"
	if maxVersionVb >= 2 {
		// If VB is supported by the client, nMaxVersionPreVB is -1, so we won't
		// get here. Because BIP 34 changed how the generation transaction is
		// serialized, we can only use version/force back to v2 blocks. This is
		// safe to do [otherwise-]unconditionally only because we are throwing
		// an exception above if a non-force deployment gets activated. Note
		// that this can probably also be removed entirely after the first BIP9
		// non-force deployment (ie, probably segwit) gets activated.
		mutable = append(mutable, "version/force")
	}

	return &btcjson.GetBlockTemplateResult{
		Capabilities:  []string{"proposal"},
		Version:       bt.Block.BlockHeader.Version,
		Rules:         rules,
		VbAvailable:   vbAvailable,
		VbRequired:    0,
		PreviousHash:  bt.Block.Hash.ToString(),
		Transactions:  transactions,
		CoinbaseAux:   &btcjson.GetBlockTemplateResultAux{Flags: mining.CoinbaseFlag},
		CoinbaseValue: &bt.Block.Txs[0].Outs[0].Value,
		LongPollID:    core.ActiveChain.Tip().GetBlockHash().ToString() + fmt.Sprintf("%d", transactionsUpdatedLast),
		Target:        blockchain.CompactToBig(bt.Block.BlockHeader.Bits).String(),
		MinTime:       indexPrev.GetMedianTimePast() + 1,
		Mutable:       mutable,
		NonceRange:    "00000000ffffffff",
		// FIXME: Allow for mining block greater than 1M.
		SigOpLimit: int64(consensus.GetMaxBlockSigOpsCount(consensus.DefaultMaxBlockSize)),
		SizeLimit:  consensus.DefaultMaxBlockSize,
		CurTime:    int64(bt.Block.BlockHeader.Time),
		Bits:       fmt.Sprintf("%08x", bt.Block.BlockHeader.Bits),
		Height:     int64(indexPrev.Height) + 1,
	}, nil
}

func getVbName(pos consensus.DeploymentPos) string {
	if int(pos) >= len(blockchain.VersionBitsDeploymentInfo) {
		logs.Error("the parameter's value out of the range of VersionBitsDeploymentInfo")
		return ""
	}
	vbinfo := blockchain.VersionBitsDeploymentInfo[pos]
	s := vbinfo.Name
	if !vbinfo.GbtForce {
		s = "!" + s
	}
	return s
}

func handleGetBlockTemplateProposal(request *btcjson.TemplateRequest) (interface{}, error) {
	hexData := request.Data
	if hexData == "" {
		return false, &btcjson.RPCError{
			Code: btcjson.ErrRPCType,
			Message: fmt.Sprintf("Data must contain the " +
				"hex-encoded serialized block that is being " +
				"proposed"),
		}
	}

	// Ensure the provided data is sane and deserialize the proposed block.
	// todo check: whether the length of data source is multiples of 2. That is to say if it is necessary for the following branch
	if len(hexData)%2 != 0 {
		hexData = "0" + hexData
	}
	dataBytes, err := hex.DecodeString(hexData)
	if err != nil {
		return false, &btcjson.RPCError{
			Code: btcjson.ErrRPCDeserialization,
			Message: fmt.Sprintf("Data must be "+
				"hexadecimal string (not %q)", hexData),
		}
	}
	var block core.Block
	if err := block.Deserialize(bytes.NewReader(dataBytes)); err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block decode failed: " + err.Error(),
		}
	}

	hash := block.Hash
	bindex := blockchain.GChainActive.FetchBlockIndexByHash(hash) // todo realise
	if bindex != nil {
		if bindex.IsValid(core.BlockValidScripts) {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "duplicate",
			}
		}
		if bindex.Status&core.BlockFailedMask != 0 {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "duplicate-invalid",
			}
		}
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrUnDefined,
			Message: "duplicate-inconclusive",
		}
	}

	indexPrev := core.ActiveChain.Tip()
	// TestBlockValidity only supports blocks built on the current Tip
	if block.BlockHeader.HashPrevBlock != indexPrev.BlockHash {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrUnDefined,
			Message: "inconclusive-not-best-prevblk",
		}
	}
	state := core.ValidationState{}
	blockchain.TestBlockValidity(msg.ActiveNetParams, &state, &block, indexPrev, false, true)
	return BIP22ValidationResult(&state)
}

func BIP22ValidationResult(state *core.ValidationState) (interface{}, error) {
	if state.IsValid() {
		return nil, nil
	}

	strRejectReason := state.GetRejectReason()
	if state.IsError() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCVerify,
			Message: strRejectReason,
		}
	}

	if state.IsInvalid() {
		if strRejectReason == "" {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "rejected",
			}
		}
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrUnDefined,
			Message: strRejectReason,
		}
	}

	// Should be impossible
	return nil, &btcjson.RPCError{
		Code:    btcjson.ErrUnDefined,
		Message: "valid?",
	}
}

// handleSubmitBlock implements the submitblock command.
func handleSubmitBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {

	c := cmd.(*btcjson.SubmitBlockCmd)

	// Deserialize the submitted block.
	hexStr := c.HexBlock
	if len(hexStr)%2 != 0 {
		hexStr = "0" + c.HexBlock
	}
	serializedBlock, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, rpcDecodeHexError(hexStr)
	}

	block := &core.Block{}
	err = block.Deserialize(bytes.NewBuffer(serializedBlock))

	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block decode failed: " + err.Error(),
		}
	}

	// Process this block using the same rules as blocks coming from other
	// nodes.  This will in turn relay it to the network like normal.
	//_, err = peer.SubmitBlock(block, blockchain.BFNone)       // TODO
	if err != nil {
		return fmt.Sprintf("rejected: %s", err.Error()), nil
	}

	logs.Info("Accepted block %s via submitblock", block.Hash)

	return nil, nil
}

func handleGenerate(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	//c := cmd.(*btcjson.GenerateCmd)
	//
	//var maxTries uint64
	//maxTries = 1000000
	//if c.MaxTries != 0 {
	//	maxTries = c.MaxTries
	//}

	//core.Script{}
	/*	// Respond with an error if the client is requesting 0 blocks to be generated.
		if c.NumBlocks == 0 {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInternal.Code,
				Message: "Please request a nonzero number of blocks to generate.",
			}
		}

		// Create a reply
		reply := make([]string, c.NumBlocks)

		blockHashes, err := s.cfg.CPUMiner.GenerateNBlocks(c.NumBlocks)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInternal.Code,
				Message: err.Error(),
			}
		}

		// Mine the correct number of blocks, assigning the hex representation of the
		// hash of each one to its place in the reply.
		for i, hash := range blockHashes {
			reply[i] = hash.String()
		}

		return reply, nil*/
	return nil, nil
}

func handleGenerateToAddress(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleEstimateFee(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleEstimatePriority(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleEstimateSmartFee(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleEstimateSmartPriority(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func registerMiningRPCCommands() {
	for name, handler := range miningHandlers {
		appendCommand(name, handler)
	}
}
