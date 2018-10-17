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

	Whitelists []string `long:"whitelist" description:"whitelist"`
}

func InitArgs(args []string) (*Opts, error) {
	opts := new(Opts)
	_, err := flags.ParseArgs(opts, args)
	if err != nil {
		if flasgErr, ok := err.(*flags.Error); ok && flasgErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		return nil, err
	}

	return opts, nil
}

func (opts *Opts) String() string {
	return fmt.Sprintf("datadir:%s regtest:%v testnet:%v", opts.DataDir, opts.RegTest, opts.TestNet)
}
