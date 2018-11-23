package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/logic/lmerkleroot"
	"github.com/copernet/copernicus/logic/lwallet"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/bitcointime"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/versionbits"
	"github.com/copernet/copernicus/net/server"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/service/mining"
	"github.com/copernet/copernicus/util"
	"gopkg.in/fatih/set.v0"
	"math/big"
)

var miningHandlers = map[string]commandHandler{
	"getnetworkhashps":  handleGetNetWorkhashPS,
	"getmininginfo":     handleGetMiningInfo,
	"getblocktemplate":  handleGetblocktemplate,
	"submitblock":       handleSubmitBlock,
	"generatetoaddress": handleGenerateToAddress,
	"generate":          handleGenerate,
	//"estimatefee":       handleEstimateFee,
}

func GetNetworkHashPS(lookup int32, height int32) float64 {
	index := chain.GetInstance().Tip()
	if height > 0 && height < chain.GetInstance().Height() {
		index = chain.GetInstance().GetIndex(height)
	}

	if index == nil || index.Height == 0 {
		return 0
	}

	if lookup <= 0 {
		lookup = index.Height%int32(model.ActiveNetParams.DifficultyAdjustmentInterval()) + 1
	}
	if lookup > index.Height {
		lookup = index.Height
	}

	b := index
	minTime := b.GetBlockTime()
	maxTime := minTime
	for i := int32(0); i < lookup; i++ {
		b = b.Prev
		blockTime := b.GetBlockTime()
		minTime = util.MinU32(blockTime, minTime)
		maxTime = util.MaxU32(blockTime, maxTime)
	}

	if minTime == maxTime {
		return 0
	}

	workDiff := new(big.Float).SetInt(new(big.Int).Sub(&index.ChainWork, &b.ChainWork))
	timeDiff := new(big.Float).SetInt64(int64(maxTime - minTime))
	hashesPerSec, _ := new(big.Float).Quo(workDiff, timeDiff).Float64()

	return hashesPerSec
}

func handleGetNetWorkhashPS(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetNetworkHashPSCmd)

	lookup := *c.Blocks
	height := *c.Height

	hashesPerSec := GetNetworkHashPS(lookup, height)

	return hashesPerSec, nil
}

func handleGetMiningInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	const defaultLookup = 120
	const defaultHeight = -1

	index := chain.GetInstance().Tip()
	result := &btcjson.GetMiningInfoResult{
		Blocks:                  index.Height,
		CurrentBlockSize:        mining.GetLastBlockSize(),
		CurrentBlockTx:          mining.GetLastBlockTx(),
		Difficulty:              getDifficulty(index),
		BlockPriorityPercentage: tx.DefaultBlockPriorityPercentage, // NOT support
		Errors:                  "",                                // NOT support
		NetworkHashPS:           GetNetworkHashPS(defaultLookup, defaultHeight),
		PooledTx:                uint64(mempool.GetInstance().Size()),
		Chain:                   chain.GetInstance().GetParams().Name,
	}
	return result, nil
}

// global variable in package rpc
var (
	transactionsUpdatedLast uint64
	indexPrev               *blockindex.BlockIndex
	start                   int64
	blocktemplate           *mining.BlockTemplate
)

// See https://en.bitcoin.it/wiki/BIP_0022 and
// https://en.bitcoin.it/wiki/BIP_0023 for more details.
func handleGetblocktemplate(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockTemplateCmd)
	request := c.Request

	// Set the default mode and override it if supplied.
	mode := "template"
	if request != nil && request.Mode != "" {
		mode = request.Mode
	}

	switch mode {
	case "template":
		return handleGetBlockTemplateRequest(s, request, closeChan)
	case "proposal":
		return handleGetBlockTemplateProposal(request)
	}

	return nil, &btcjson.RPCError{
		Code:    btcjson.ErrRPCInvalidParameter,
		Message: "Invalid mode",
	}
}

func handleGetBlockTemplateRequest(s *Server, request *btcjson.TemplateRequest, closeChan <-chan struct{}) (interface{}, error) {
	maxVersionVb := int64(-1)
	setClientRules := set.New()
	log.Debug("getblocktemplate %#v", request)

	if lchain.IsInitialBlockDownload() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCClientInInitialDownload,
			Message: "Bitcoin is downloading blocks...",
		}
	}

	//if request != nil && request.LongPollID != "" {
	// Wait to respond until either the best block changes, OR a minute has
	// passed and there are more transactions
	//var hashWatchedChain utils.Hash
	//checktxtime := time.Now()
	//transactionsUpdatedLastLP := 0
	// todo complete
	//}

	persist.CsMain.Lock() //lock chain tip for CreateNewBlock
	defer persist.CsMain.Unlock()

	if indexPrev != chain.GetInstance().Tip() ||
		mempool.GetInstance().TransactionsUpdated != transactionsUpdatedLast &&
			util.GetTime()-start > 5 {

		// Clear pindexPrev so future calls make a new block, despite any
		// failures from here on
		indexPrev = nil
		// Store the pindexBest used before CreateNewBlock, to avoid races
		transactionsUpdatedLast = mempool.GetInstance().TransactionsUpdated
		indexPrevNew := chain.GetInstance().Tip()
		start = util.GetTime()

		// Create new block
		ba := mining.NewBlockAssembler(model.ActiveNetParams, s.timeSource)
		scriptPubKey := script.NewScriptRaw([]byte{opcodes.OP_TRUE})
		blocktemplate = ba.CreateNewBlock(scriptPubKey, mining.BasicScriptSig())
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

	res, err := blockTemplateResult(blocktemplate, setClientRules, uint32(maxVersionVb), transactionsUpdatedLast)
	log.Debug("getblocktemplate response bits: %d, height: %d, time: %d, expires: %d， prehash: %s, noncerange: %s",
		res.Bits, res.Height, res.CurTime, res.Expires, res.PreviousHash, res.NonceRange)
	return res, err
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
		err := tx.Serialize(dataBuf)
		if err != nil {
			log.Error("mining:serialize tx failed: %v", err)
			return nil, err
		}
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

	mutable := make([]string, 3, 4)
	mutable[0] = "time"
	mutable[1] = "transactions"
	mutable[2] = "prevblock"

	coinbaseValue := bt.Block.Txs[0].GetTxOut(0).GetValue()
	target := pow.CompactToBig(bt.Block.Header.Bits)
	maxSigOps, _ := consensus.GetMaxBlockSigOpsCount(consensus.DefaultMaxBlockSize)
	return &btcjson.GetBlockTemplateResult{
		Capabilities:  []string{"proposal"},
		Version:       bt.Block.Header.Version,
		PreviousHash:  bt.Block.Header.HashPrevBlock.String(),
		Transactions:  transactions,
		CoinbaseAux:   &btcjson.GetBlockTemplateResultAux{Flags: mining.CoinbaseFlag},
		CoinbaseValue: (*int64)(&coinbaseValue),
		LongPollID:    chain.GetInstance().Tip().GetBlockHash().String() + fmt.Sprintf("%d", transactionsUpdatedLast),
		Target:        fmt.Sprintf("%064x", &target),
		MinTime:       indexPrev.GetMedianTimePast() + 1,
		Mutable:       mutable,
		NonceRange:    "00000000ffffffff",
		// FIXME: Allow for mining block greater than 1M.
		SigOpLimit: int64(maxSigOps),
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
		if bindex.IsValid(blockindex.BlockValidScripts) {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrUnDefined,
				Message: "duplicate",
			}
		}

		if bindex.IsInvalid() {
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
		return "inconclusive-not-best-prevblk", nil
	}

	err = mining.TestBlockValidity(&bk, indexPrev, false, true)
	err = errcode.GetBip22Result(err)
	//err = lblock.CheckBlock(&bk, true, true)
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
				strRejectReason = "rejected"
			}

			return strRejectReason, nil
		}
	}

	// Should be impossible
	return "valid?", nil
}

// handleSubmitBlock implements the submitblock command.
func handleSubmitBlock(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SubmitBlockCmd)

	// Unserialize the submitted block.
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
		log.Error("Block decode failed: %s", err.Error())
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block decode failed: " + err.Error(),
		}
	}

	if len(bk.Txs) == 0 || !bk.Txs[0].IsCoinBase() {
		log.Error("Block does not start with a coinbase, block hash: %s", bk.GetHash().String())
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block does not start with a coinbase",
		}
	}

	hash := bk.GetHash()
	_, err = server.ProcessForRPC(bk)
	if err != nil {
		log.Error("rejected: %s, blk=%+v txs=%+v", err.Error(), bk, bk.Txs)
		return fmt.Sprintf("rejected: %s", err.Error()), nil
	}
	log.Debug("Accepted block %s via submitblock", &hash)
	return nil, nil
}

func handleGenerateToAddress(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GenerateToAddressCmd)

	coinbaseScript, rpcErr := getStandardScriptPubKey(c.Address, nil)
	if rpcErr != nil {
		return nil, rpcErr
	}

	return generateBlocks(coinbaseScript, int(c.NumBlocks), *c.MaxTries, s.timeSource)
}

// handleGenerate handles generate commands.
func handleGenerate(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GenerateCmd)

	if !lwallet.IsWalletEnable() {
		return nil, walletDisableRPCError
	}

	// Respond with an error if the client is requesting 0 blocks to be generated.
	if c.NumBlocks == 0 {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInternal.Code,
			Message: "Please request a nonzero number of blocks to generate.",
		}
	}

	addr, err := lwallet.GetMiningAddress()
	if err != nil {
		log.Info("GetMiningAddress error:%s", err.Error())
		return nil, btcjson.ErrRPCInternal
	}

	coinbaseScript, rpcErr := getStandardScriptPubKey(addr, nil)
	if rpcErr != nil || coinbaseScript == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "No coinbase script available (mining requires a wallet)",
		}
	}

	return generateBlocks(coinbaseScript, int(c.NumBlocks), *c.MaxTries, s.timeSource)
}

const nInnerLoopCount = 0x100000

func generateBlocks(scriptPubKey *script.Script, generate int, maxTries uint64, ts *bitcointime.MedianTime) (interface{}, error) {
	heightStart := chain.GetInstance().Height()
	heightEnd := heightStart + int32(generate)
	height := heightStart
	params := model.ActiveNetParams

	ret := make([]string, 0)
	var extraNonce uint
	for height < heightEnd {
		ba := mining.NewBlockAssembler(params, ts)

		bt := createBlockForCPUMining(ba, scriptPubKey, extraNonce)
		if bt == nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCInternalError,
				Message: "Could not create new block",
			}
		}

		bt.Block.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(bt.Block.Txs, nil)

		powCheck := pow.Pow{}
		bits := bt.Block.Header.Bits
		for maxTries > 0 && bt.Block.Header.Nonce < nInnerLoopCount {
			maxTries--
			bt.Block.Header.Nonce++
			hash := bt.Block.GetHash()
			if powCheck.CheckProofOfWork(&hash, bits, params) {
				break
			}
		}

		if maxTries == 0 {
			break
		}

		if bt.Block.Header.Nonce == nInnerLoopCount {
			extraNonce++
			continue
		}

		if _, err := server.ProcessForRPC(bt.Block); err != nil {
			log.Error("generateBlocks: ProcessNewBlock got an error:", err)
			return nil, btcjson.RPCError{
				Code:    btcjson.RPCInternalError,
				Message: "ProcessNewBlock, block not accepted",
			}
		}

		height++
		extraNonce = 0

		blkHash := bt.Block.GetHash()
		ret = append(ret, blkHash.String())

		// TODO: simple implementation just for testing
		if lwallet.IsWalletEnable() {
			lwallet.AddToWallet(bt.Block.Txs[0], bt.Block.GetHash(), nil)
		}
	}

	return ret, nil
}

func createBlockForCPUMining(ba *mining.BlockAssembler, scriptPK *script.Script, extraNonce uint) *mining.BlockTemplate {
	persist.CsMain.Lock()
	defer persist.CsMain.Unlock()
	return ba.CreateNewBlock(scriptPK, mining.CoinbaseScriptSig(extraNonce))
}

// handleEstimateFee handles estimatefee commands.
//func handleEstimateFee(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//	c := cmd.(*btcjson.EstimateFeeCmd)
//
//	if s.cfg.FeeEstimator == nil {
//		return nil, errors.New("Fee estimation disabled")
//	}
//
//	if c.NumBlocks <= 0 {
//		return -1.0, errors.New("Parameter NumBlocks must be positive")
//	}
//
//	feeRate, err := s.cfg.FeeEstimator.EstimateFee(uint32(c.NumBlocks))
//
//	if err != nil {
//		return -1.0, err
//	}
//
//	// Convert to satoshis per kb.
//	return float64(feeRate), nil
//}

func registerMiningRPCCommands() {
	for name, handler := range miningHandlers {
		appendCommand(name, handler)
	}
}
