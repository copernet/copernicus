package conf

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

type Opts struct {
	DataDir string `long:"datadir" description:"specified program data dir"`
	Reindex bool   `long:"reindex" description:"reindex"`

	// //Set -discover=0 in regtest framework
	// Discover int  `long:"discover" default:"1" description:"Discover own IP addresses (default: 1 when listening and no -externalip or -proxy) "`
	RegTest bool `long:"regtest" description:"initiate regtest"`
	TestNet bool `long:"testnet" description:"initiate testnet"`

	UtxoHashStartHeigh int32 `long:"utxohashstartheight" default:"-1" description:"Which height begin logging out the utxos hash at"`
	UtxoHashEndHeigh   int32 `long:"utxohashendheight" default:"-1" description:"Which height finish logging out the utxos hash at"`

	Whitelists         []string `long:"whitelist" description:"whitelist"`
	Excessiveblocksize uint64   `long:"excessiveblocksize" default:"32000000" description:"excessive block size"`

	ReplayProtectionActivationTime int64  `long:"replayprotectionactivationtime" default:"-1"`
	MonolithActivationTime         int64  `long:"monolithactivationtime" default:"-1"`
	StopAtHeight                   int32  `long:"stopatheight" default:"-1"`
	PromiscuousMempoolFlags        string `long:"promiscuousmempoolflags"`
	Limitancestorcount             int    `long:"limitancestorcount" default:"50000"`
	BlockVersion                   int32  `long:"blockversion" default:"-1" description:"regtest block version"`
	MaxMempool                     int64  `long:"maxmempool" default:"300000000"`
	SpendZeroConfChange            uint8  `long:"spendzeroconfchange" default:"1"`
}

func InitArgs(args []string) (*Opts, error) {
	opts := new(Opts)

	_, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).ParseArgs(args)
	if err == nil {

		if !opts.RegTest {
			return strictParseArgs(err, args)
		}
	}

	return opts, err
}

func strictParseArgs(err error, args []string) (*Opts, error) {
	opts := new(Opts)
	_, err = flags.NewParser(opts, flags.Default).ParseArgs(args)
	if flasgErr, ok := err.(*flags.Error); ok && flasgErr.Type == flags.ErrHelp {
		os.Exit(0)
	}
	return opts, err
}

func (opts *Opts) String() string {
	return fmt.Sprintf("datadir:%s regtest:%v testnet:%v", opts.DataDir, opts.RegTest, opts.TestNet)
}
