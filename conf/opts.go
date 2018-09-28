package conf

import (
	"fmt"
	"github.com/jessevdk/go-flags"
)

type Opts struct {
	DataDir string `long:"datadir" description:"specified program data dir"`

	// //Set -discover=0 in regtest framework
	// Discover int  `long:"discover" default:"1" description:"Discover own IP addresses (default: 1 when listening and no -externalip or -proxy) "`
	RegTest bool `long:"regtest" description:"initiate regtest"`
	TestNet bool `long:"testnet" description:"initiate testnet"`
}

func InitArgs(args []string) (*Opts, error) {
	opts := new(Opts)
	_, err := flags.ParseArgs(opts, args)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func (opts *Opts) String() string {
	return fmt.Sprintf("datadir:%s regtest:%v testnet:%v", opts.DataDir, opts.RegTest, opts.TestNet)
}
