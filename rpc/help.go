package rpc

import (
	"sort"
	"strings"
	"sync"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/rpc/btcjson"
)

// helpCacher provides a concurrent safe type that provides help and usage for
// the RPC server commands and caches the results for future calls.
type helpCacher struct {
	sync.Mutex
	usage      string
	methodHelp map[string]helpDescInfo
}

type helpDescInfo struct {
	category    string
	description string
}

const (
	DebugCmd           = ""
	BlockChainCmd      = "BlockChain"
	ControlCmd         = "Control"
	GeneratingCmd      = "Generating"
	MiningCmd          = "Mining"
	NetworkCmd         = "Network"
	RawTransactionsCmd = "RawTransactions"
	UtilCmd            = "Util"
	WalletCmd          = "Wallet"
)

var allMethodHelp = map[string]helpDescInfo{
	"getblockchaininfo":     {BlockChainCmd, getblockchaininfoDesc},
	"getbestblockhash":      {BlockChainCmd, getbestblockhashDesc},
	"getblockcount":         {BlockChainCmd, getblockcountDesc},
	"getblock":              {BlockChainCmd, getblockDesc},
	"getblockhash":          {BlockChainCmd, getblockhashDesc},
	"getblockheader":        {BlockChainCmd, getblockheader},
	"getchaintips":          {BlockChainCmd, getchaintipsDesc},
	"getchaintxstats":       {BlockChainCmd, getchaintxstatsDesc},
	"getdifficulty":         {BlockChainCmd, getdifficultyDesc},
	"getmempoolancestors":   {BlockChainCmd, getmempoolancestorsDesc},
	"getmempooldescendants": {BlockChainCmd, getmempooldescendantsDesc},
	"getmempoolentry":       {BlockChainCmd, getmempoolentryDesc},
	"getmempoolinfo":        {BlockChainCmd, getmempoolinfoDesc},
	"getrawmempool":         {BlockChainCmd, getrawmempoolDesc},
	"gettxout":              {BlockChainCmd, gettxoutDesc},
	"gettxoutsetinfo":       {BlockChainCmd, gettxoutsetinfoDesc},
	"pruneblockchain":       {BlockChainCmd, pruneblockchainDesc},
	"verifychain":           {BlockChainCmd, verifychainDesc},
	"preciousblock":         {BlockChainCmd, preciousblockDesc},

	"getnetworkhashps": {MiningCmd, getnetworkhashpsDesc},
	"getmininginfo":    {MiningCmd, getmininginfoDesc},
	"getblocktemplate": {MiningCmd, getblocktemplateDesc},
	"submitblock":      {MiningCmd, submitblockDesc},

	"generate":          {GeneratingCmd, generateDesc},
	"generatetoaddress": {GeneratingCmd, generatetoaddressDesc},

	"getconnectioncount": {NetworkCmd, getconnectioncountDesc},
	"ping":               {NetworkCmd, pingDesc},
	"getpeerinfo":        {NetworkCmd, getpeerinfoDesc},
	"addnode":            {NetworkCmd, addnodeDesc},
	"disconnectnode":     {NetworkCmd, disconnectnodeDesc},
	"getaddednodeinfo":   {NetworkCmd, getaddednodeinfoDesc},
	"getnettotals":       {NetworkCmd, getnettotalsDesc},
	"getnetworkinfo":     {NetworkCmd, getnetworkinfoDesc},
	"setban":             {NetworkCmd, setbanDesc},
	"listbanned":         {NetworkCmd, listbannedDesc},
	"clearbanned":        {NetworkCmd, clearbannedDesc},
	"setnetworkactive":   {NetworkCmd, setnetworkactiveDesc},

	"getrawtransaction":    {RawTransactionsCmd, getrawtransactionDesc},
	"createrawtransaction": {RawTransactionsCmd, createrawtransactionDesc},
	"decoderawtransaction": {RawTransactionsCmd, decoderawtransactionDesc},
	"decodescript":         {RawTransactionsCmd, decodescriptDesc},
	"sendrawtransaction":   {RawTransactionsCmd, sendrawtransactionDesc},
	"signrawtransaction":   {RawTransactionsCmd, signrawtransactionDesc},
	"gettxoutproof":        {RawTransactionsCmd, gettxoutproofDesc},
	"verifytxoutproof":     {RawTransactionsCmd, verifytxoutproofDesc},

	"getinfo": {ControlCmd, getinfoDesc},
	"help":    {ControlCmd, helpDesc},
	"stop":    {ControlCmd, stopDesc},

	"validateaddress": {UtilCmd, validateaddressDesc},
	"createmultisig":  {UtilCmd, createmultisigDesc},

	"getexcessiveblock":  {DebugCmd, getexcessiveblockDesc},
	"setexcessiveblock":  {DebugCmd, setexcessiveblockDesc},
	"waitforblockheight": {DebugCmd, waitforblockheightDesc},
	"echo":               {DebugCmd, echoDesc},

	"getnewaddress":  {WalletCmd, getnewaddressDesc},
	"listunspent":    {WalletCmd, listunspentDesc},
	"settxfee":       {WalletCmd, settxfeeDesc},
	"sendtoaddress":  {WalletCmd, sendtoaddressDesc},
	"getbalance":     {WalletCmd, getbalanceDesc},
	"gettransaction": {WalletCmd, gettransactionDesc},
}

// rpcMethodHelp returns an RPC help string for the provided method.
//
// This function is safe for concurrent access.
func (c *helpCacher) rpcMethodHelp(method string) (string, error) {
	c.Lock()
	defer c.Unlock()
	help, exists := c.methodHelp[method]

	if !exists {
		return "", nil
	}

	return help.description, nil
}

// rpcUsage returns one-line usage for all support RPC commands.
//
// This function is safe for concurrent access.
func (c *helpCacher) rpcUsage(includeWebsockets bool) (string, error) {
	c.Lock()
	defer c.Unlock()

	// Return the cached usage if it is available.
	if c.usage != "" {
		return c.usage, nil
	}

	// Generate a list of one-line usage for every command.
	usageTexts := make(map[string]*[]string)
	for method, info := range allMethodHelp {
		// Debug command is not shown in help
		if info.category == DebugCmd {
			continue
		}
		if !conf.Cfg.Wallet.Enable && info.category == WalletCmd {
			continue
		}
		if _, ok := usageTexts[info.category]; !ok {
			category := make([]string, 0)
			usageTexts[info.category] = &category
		}
		usage, err := btcjson.MethodUsageText(method)
		if err != nil {
			return "", err
		}
		*usageTexts[info.category] = append(*usageTexts[info.category], usage)
	}

	categories := make([]string, 0, len(usageTexts))
	for category := range usageTexts {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		sort.Strings(*usageTexts[category])
		c.usage += "--- " + category + " ---\n" +
			strings.Join(*usageTexts[category], "\n") + "\n\n"
	}
	return c.usage, nil
}

// newHelpCacher returns a new instance of a help cacher which provides help and
// usage for the RPC server commands and caches the results for future calls.
func newHelpCacher() *helpCacher {
	return &helpCacher{
		methodHelp: allMethodHelp,
	}
}
