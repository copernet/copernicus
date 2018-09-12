package rpc

import (
	"sort"
	"strings"
	"sync"

	"github.com/copernet/copernicus/rpc/btcjson"
)

// helpCacher provides a concurrent safe type that provides help and usage for
// the RPC server commands and caches the results for future calls.
type helpCacher struct {
	sync.Mutex
	usage      string
	methodHelp map[string]string
}

var methodHelp = map[string]string{
	"getexcessiveblock": getexcessiveblockDesc,
	"setexcessiveblock": setexcessiveblockDesc,

	"getblockchaininfo":     getblockchaininfoDesc,
	"getbestblockhash":      getbestblockhashDesc,
	"getblockcount":         getblockcountDesc,
	"getblock":              getblockDesc,
	"getblockhash":          getblockhashDesc,
	"getblockheader":        getblockheader,
	"getchaintips":          getchaintipsDesc,
	"getchaintxstats":       getchaintxstatsDesc,
	"getdifficulty":         getdifficultyDesc,
	"getmempoolancestors":   getmempoolancestorsDesc,
	"getmempooldescendants": getmempooldescendantsDesc,
	"getmempoolentry":       getmempoolentryDesc,
	"getmempoolinfo":        getmempoolinfoDesc,
	"getrawmempool":         getrawmempoolDesc,
	"gettxout":              gettxoutDesc,
	"gettxoutsetinfo":       gettxoutsetinfoDesc,
	"pruneblockchain":       pruneblockchainDesc,
	"verifychain":           verifychainDesc,
	"preciousblock":         preciousblockDesc,

	"getnetworkhashps":  getnetworkhashpsDesc,
	"getmininginfo":     getmininginfoDesc,
	"getblocktemplate":  getblocktemplateDesc,
	"submitblock":       submitblockDesc,
	"generate":          generateDesc,
	"generatetoaddress": generatetoaddressDesc,

	"getconnectioncount": getconnectioncountDesc,
	"ping":               pingDesc,
	"getpeerinfo":        getpeerinfoDesc,
	"addnode":            addnodeDesc,
	"disconnectnode":     disconnectnodeDesc,
	"getaddednodeinfo":   getaddednodeinfoDesc,
	"getnettotals":       getnettotalsDesc,
	"getnetworkinfo":     getnetworkinfoDesc,
	"setban":             setbanDesc,
	"listbanned":         listbannedDesc,
	"clearbanned":        clearbannedDesc,
	"setnetworkactive":   setnetworkactiveDesc,

	"getrawtransaction":    getrawtransactionDesc,
	"createrawtransaction": createrawtransactionDesc,
	"decoderawtransaction": decoderawtransactionDesc,
	"decodescript":         decodescriptDesc,
	"sendrawtransaction":   sendrawtransactionDesc,
	"signrawtransaction":   signrawtransactionDesc,
	"gettxoutproof":        gettxoutproofDesc,
	"verifytxoutproof":     verifytxoutproofDesc,

	"getinfo":         getinfoDesc,
	"validateaddress": validateaddressDesc,
	"createmultisig":  createmultisigDesc,
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

	return help, nil
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
	usageTexts := make([]string, 0, len(methodHelp))
	for k := range methodHelp {

		usage, err := btcjson.MethodUsageText(k)
		if err != nil {
			return "", err
		}
		usageTexts = append(usageTexts, usage)
	}

	sort.Sort(sort.StringSlice(usageTexts))
	c.usage = strings.Join(usageTexts, "\n")
	return c.usage, nil
}

// newHelpCacher returns a new instance of a help cacher which provides help and
// usage for the RPC server commands and caches the results for future calls.
func newHelpCacher() *helpCacher {
	return &helpCacher{
		methodHelp: methodHelp,
	}
}
