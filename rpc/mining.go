package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	block2 "github.com/btcboost/copernicus/logic/block"
	blockindex2 "github.com/btcboost/copernicus/logic/blockindex"
	"github.com/btcboost/copernicus/logic/undo"
	"github.com/btcboost/copernicus/model/bitaddr"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/opcodes"
	"github.com/btcboost/copernicus/model/pow"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/model/versionbits"
	"github.com/btcboost/copernicus/rpc/btcjson"
	"github.com/btcboost/copernicus/service/mining"
	"github.com/btcboost/copernicus/util"
	"gopkg.in/fatih/set.v0"
)

var miningHandlers = map[string]commandHandler{
	"getnetworkhashps":      handleGetNetWorkhashPS,      // complete
	"getmininginfo":         handleGetMiningInfo,         // complete
	"prioritisetransaction": handlePrioritisetransaction, // do not support at this version
	"getblocktemplate":      handleGetblocktemplate,      // complete
	"submitblock":           handleSubmitBlock,           // complete
	"generate":              handleGenerate,              // deprecated at new version<v0.17.1>
	"generatetoaddress":     handleGenerateToAddress,     // complete
	"estimatefee":           handleEstimateFee,           // do not support at this version
	"estimatepriority":      handleEstimatePriority,      // do not support at this version
	"estimatesmartfee":      handleEstimateSmartFee,      // do not support at this version
	"estimatesmartpriority": handleEstimateSmartPriority, // do not support at this version
}

func handleGetNetWorkhashPS(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetNetworkHashPSCmd)

	lookup := 120
	height := int32(-1)
	if c.Blocks != nil {
		lookup = *c.Blocks
	}

	if c.Height != nil {
		height = *c.Height
	}

	index := chain.GetInstance().Tip()
	if height > 0 || height < chain.GetInstance().Height() {
		index = chain.GetInstance().GetIndex(height)
	}

	if index == nil || index.Height != 0 {
		return 0, nil
	}

	if lookup <= 0 {
		lookup = int(index.Height%int32(chainparams.ActiveNetParams.DifficultyAdjustmentInterval()) + 1)
	}

	if lookup > int(index.Height) {
		lookup = int(index.Height)
	}

	b := index
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

	workDiff := new(big.Int).Sub(&index.ChainWork, &b.ChainWork)
	timeDiff := int64(maxTime - minTime)

	hashesPerSec := new(big.Int).Div(workDiff, big.NewInt(timeDiff))
	return hashesPerSec, nil
}

func handleGetMiningInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	gnhpsCmd := btcjson.NewGetNetworkHashPSCmd(nil, nil)
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

		index := chain.GetInstance().Tip()
		result := btcjson.GetMiningInfoResult{
			Blocks:                  int64(index.Height),
			CurrentBlockSize:        mining.GetLastBlockSize(),
			CurrentBlockTx:          mining.GetLastBlockTx(),
			Difficulty:              getDifficulty(index),
			BlockPriorityPercentage: util.GetArg("-blockprioritypercentage", 0),
			//Errors:              ,                            // TODO
			NetworkHashPS: networkHashesPerSec,
			//PooledTx:           uint64(mempool.Size()),              TODO
			Chain: consensus.MainNetParams.Name,
		}
		return &result, nil*/ // todo open
	return nil, nil
}

// priority transaction currently disabled
func handlePrioritisetransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

// global variable in package rpc
var (
	transactionsUpdatedLast uint64
	indexPrev               *blockindex.BlockIndex
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

	if undo.IsInitialBlockDownload() {
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

	if indexPrev != chain.GetInstance().Tip() ||
		mempool.GetInstance().TransactionsUpdated != transactionsUpdatedLast &&
			util.GetMockTime()-start > 5 {

		// Clear pindexPrev so future calls make a new block, despite any
		// failures from here on
		indexPrev = nil
		// Store the pindexBest used before CreateNewBlock, to avoid races
		transactionsUpdatedLast = mempool.GetInstance().TransactionsUpdated
		indexPrevNew := chain.GetInstance().Tip()
		start = util.GetMockTime()

		// Create new block
		ba := mining.NewBlockAssembler(chainparams.ActiveNetParams)
		blocktemplate = ba.CreateNewBlock(script.NewScriptRaw([]byte{opcodes.OP_TRUE}))
		if blocktemplate == nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "Out of memory",
			}
		}

		// Need to update only after we know CreateNewBlock succeeded
		indexPrev = indexPrevNew
	}
	bk := blocktemplate.Block
	mining.UpdateTime(bk, indexPrev)
	bk.Header.Nonce = 0

	return blockTemplateResult(blocktemplate, setClientRules, maxVersionVb, transactionsUpdatedLast)
}

// blockTemplateResult returns the current block template associated with the
// state as a btcjson.GetBlockTemplateResult that is ready to be encoded to JSON
// and returned to the caller.
//
// This function MUST be called with the state locked.
func blockTemplateResult(bt *mining.BlockTemplate, s *set.Set, maxVersionVb uint32, transactionsUpdatedLast uint64) (*btcjson.GetBlockTemplateResult, error) {
	setTxIndex := make(map[util.Hash]int)
	var i int
	transactions := make([]btcjson.GetBlockTemplateResultTx, 0, len(bt.Block.Txs))
	for _, tx := range bt.Block.Txs {
		txID := tx.GetHash()
		setTxIndex[txID] = i
		i++

		if tx.IsCoinBase() {
			continue
		}

		entry := btcjson.GetBlockTemplateResultTx{}

		dataBuf := bytes.NewBuffer(nil)
		tx.Serialize(dataBuf)
		entry.Data = hex.EncodeToString(dataBuf.Bytes())

		entry.TxID = txID.String()
		entry.Hash = txID.String()

		deps := make([]int, 0)
		for _, in := range tx.GetIns() {
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
		state := versionbits.VersionBitsState(indexPrev, chainparams.ActiveNetParams, pos, versionbits.VBCache)
		switch state {
		case versionbits.ThresholdDefined:
			fallthrough
		case versionbits.ThresholdFailed:
			// Not exposed to GBT at all and break
		case versionbits.ThresholdLockedIn:
			// Ensure bit is set in block version, then fallthrough to get
			// vbavailable set.
			bt.Block.Header.Version |= int32(versionbits.VersionBitsMask(chainparams.ActiveNetParams, pos))
			fallthrough
		case versionbits.ThresholdStarted:
			vbinfo := versionbits.VersionBitsDeploymentInfo[pos]
			vbAvailable[getVbName(pos)] = chainparams.ActiveNetParams.Deployments[pos].Bit
			if !s.Has(vbinfo.Name) {
				if !vbinfo.GbtForce {
					// If the client doesn't support this, don't indicate it
					// in the [default] version
					bt.Block.Header.Version &= int32(^versionbits.VersionBitsMask(chainparams.ActiveNetParams, pos))
				}
			}
		case versionbits.ThresholdActive:
			// Add to rules only
			vbinfo := versionbits.VersionBitsDeploymentInfo[pos]
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

	v := bt.Block.Txs[0].GetTxOut(0).GetValue()
	return &btcjson.GetBlockTemplateResult{
		Capabilities:  []string{"proposal"},
		Version:       bt.Block.Header.Version,
		Rules:         rules,
		VbAvailable:   vbAvailable,
		VbRequired:    0,
		PreviousHash:  bt.Block.Header.HashPrevBlock.String(),
		Transactions:  transactions,
		CoinbaseAux:   &btcjson.GetBlockTemplateResultAux{Flags: mining.CoinbaseFlag},
		CoinbaseValue: (*int64)(&v),
		LongPollID:    chain.GetInstance().Tip().GetBlockHash().String() + fmt.Sprintf("%d", transactionsUpdatedLast),
		Target:        pow.CompactToBig(bt.Block.Header.Bits).String(),
		MinTime:       indexPrev.GetMedianTimePast() + 1,
		Mutable:       mutable,
		NonceRange:    "00000000ffffffff",
		// FIXME: Allow for mining block greater than 1M.
		SigOpLimit: int64(consensus.GetMaxBlockSigOpsCount(consensus.DefaultMaxBlockSize)),
		SizeLimit:  consensus.DefaultMaxBlockSize,
		CurTime:    int64(bt.Block.Header.Time),
		Bits:       fmt.Sprintf("%08x", bt.Block.Header.Bits),
		Height:     int64(indexPrev.Height) + 1,
	}, nil
}

func getVbName(pos consensus.DeploymentPos) string {
	if int(pos) >= len(versionbits.VersionBitsDeploymentInfo) {
		log.Error("the parameter's value out of the range of VersionBitsDeploymentInfo")
		return ""
	}
	vbinfo := versionbits.VersionBitsDeploymentInfo[pos]
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
	// todo check: whether the length of data source is multiples of 2.
	// That is to say if it is necessary for the following branch
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
	var bk block.Block
	if err := bk.Unserialize(bytes.NewReader(dataBytes)); err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block decode failed: " + err.Error(),
		}
	}

	hash := bk.Header.GetHash()
	bindex := chain.GetInstance().FindBlockIndex(hash)
	if bindex != nil {
		if bindex.IsValid(blockindex2.BlockValidScripts) {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "duplicate",
			}
		}
		if bindex.Status&blockindex2.BlockFailedMask != 0 {
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

	indexPrev := chain.GetInstance().Tip()
	// TestBlockValidity only supports blocks built on the current Tip
	if bk.Header.HashPrevBlock != *indexPrev.GetBlockHash() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrUnDefined,
			Message: "inconclusive-not-best-prevblk",
		}
	}

	// TODO realise in block model
	err = block2.Check(&bk)
	return BIP22ValidationResult(err)
}

func BIP22ValidationResult(err error) (interface{}, error) {
	projectError, ok := err.(errcode.ProjectError) // todo warning: TestBlockValidity should return type errcode.ProjectError
	if ok {
		if projectError.Code == int(errcode.ModelValid) {
			return nil, nil
		}

		strRejectReason := projectError.Desc

		if projectError.Code == int(errcode.ModelError) {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCVerify,
				Message: strRejectReason,
			}
		}

		if projectError.Code == int(errcode.ModelInvalid) {
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

	bk := &block.Block{}
	err = bk.Unserialize(bytes.NewBuffer(serializedBlock))

	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block decode failed: " + err.Error(),
		}
	}

	// Process this block using the same rules as blocks coming from other
	// nodes.  This will in turn relay it to the network like normal.
	//_, err = peer.SubmitBlock(block, chain.BFNone)       // TODO
	if err != nil {
		return fmt.Sprintf("rejected: %s", err.Error()), nil
	}

	log.Info("Accepted block %s via submitblock", bk.Header.GetHash())

	return nil, nil
}

func handleGenerate(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleGenerateToAddress(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GenerateToAddressCmd)

	addr, err := bitaddr.AddressFromString(c.Address)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Error: Invalid address",
		}
	}

	coinbaseScript := script.NewScriptRaw(addr.EncodeToPubKeyHash())
	return generateBlocks(coinbaseScript, int(c.NumBlocks), c.MaxTries)
}

const nInnerLoopCount = 0x100000

func generateBlocks(coinbaseScript *script.Script, generate int, maxTries uint64) (interface{}, error) {
	/*	heightStart := chain.GetInstance().Height()
		heightEnd := heightStart + generate
		height := heightStart

		ret := make([]string, 0)
		var extraNonce uint
		for height < heightEnd {
			ba := mining.NewBlockAssembler(consensus.ActiveNetParams)
			bt := ba.CreateNewBlock(coinbaseScript)
			if bt == nil {
				return nil, btcjson.RPCError{
					Code:    btcjson.RPCInternalError,
					Message: "Could not create new block",
				}
			}

			extraNonce = mining.IncrementExtraNonce(bt.Block, chain.GetInstance().Tip())
			// blockchain.CheckProofOfWork(bt.Block, consensus.ActiveNetParams.PowLimit) != nil // todo realise <CheckProofOfWork()>
			for maxTries > 0 && bt.Block.Header.Nonce < nInnerLoopCount {
				bt.Block.Header.Nonce++
				maxTries--
			}

			if maxTries == 0 {
				break
			}
			if bt.Block.Header.Nonce == nInnerLoopCount {
				continue
			}

			if ProcessNewBlock(bt.Block, true, nil) { // todo realise <ProcessNewBlock>
				return nil, btcjson.RPCError{
					Code:    btcjson.RPCInternalError,
					Message: "ProcessNewBlock, block not accepted",
				}
			}
			height++
			ret = append(ret, bt.Block.GetHash().String())
		}
		_ = extraNonce

		return ret, nil*/ // TODO open
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
