package conf

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"os"
	"strings"
)

type Opts struct {
	DataDir string `long:"datadir" description:"specified program data dir"`

	// //Set -discover=0 in regtest framework
	// Discover int  `long:"discover" default:"1" description:"Discover own IP addresses (default: 1 when listening and no -externalip or -proxy) "`
	RegTest bool `long:"regtest" description:"initiate regtest"`
	TestNet bool `long:"testnet" description:"initiate testnet"`
}

func InitArgs(args []string) (*Opts, error) {
	e := checkArgs(args)
	if e != nil {
		return nil, e
	}
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

func checkArgs(args []string) error {
	if len(args) < 1 {
		return nil
	}

	for _, v := range args[1:] {
		if len(v) == 1 || !strings.HasPrefix(v, "-") || v == "--" {
			return errors.New("args error")
		}
	}
	return nil
}

func (opts *Opts) String() string {
	return fmt.Sprintf("datadir:%s regtest:%v testnet:%v", opts.DataDir, opts.RegTest, opts.TestNet)
}
