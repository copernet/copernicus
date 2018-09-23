package conf

import (
	"fmt"
	"github.com/jessevdk/go-flags"
)

type Opts struct {
	DataDir  string `long:"datadir" description:"specified program data dir"`
	Discover int    `long:"discover" description:"Discover own IP addresses (default: 1 when listening and no -externalip or -proxy)"`
}

func InitArgs(args []string) *Opts {
	opts := new(Opts)
	_, err := flags.ParseArgs(opts, args)
	if err != nil {
		panic(err)
	}
	return opts
}

func (opts *Opts) String() string {
	return fmt.Sprintf("datadir:%s ,Discover:%d", opts.DataDir, opts.Discover)
}
