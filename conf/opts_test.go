package conf

import "testing"

var args = []string{
	"--datadir=/test",
	"--regtest",
	"--testnet",
}

var empty []string

func TestInitArgs(t *testing.T) {

	opts, err := InitArgs(args)
	if err != nil {
		t.Error(err.Error())
	}

	// if opts.Discover != 1 {
	// 	t.Errorf("format error  discover is %d", opts.Discover)
	// }
	if opts.DataDir != "/test" {
		t.Errorf("format error ")
	}

}

func TestOpts_String(t *testing.T) {
	opts, err := InitArgs(args)
	if err != nil {
		t.Error(err.Error())
	}
	str := opts.String()
	if str != "datadir:/test regtest:true testnet:true" {
		t.Errorf("opts to string is error :%s", str)
	}
	opts, err = InitArgs(empty)
	if err != nil {
		t.Error(err.Error())
	}
	str = opts.String()
	if str != "datadir: regtest:false testnet:false" {
		t.Errorf("opts to string is error :%s", str)
	}
}
