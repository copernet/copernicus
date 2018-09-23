package conf

import "testing"

var args = []string{
	"--datadir=/test",
	"--discover", "1",
}

func TestInitArgs(t *testing.T) {

	opts := InitArgs(args)

	if opts.Discover != 1 {
		t.Errorf("format error  discover is %d", opts.Discover)
	}
	if opts.DataDir != "/test" {
		t.Errorf("format error ")
	}

}

func TestOpts_String(t *testing.T) {
	opts := InitArgs(args)
	str := opts.String()
	if str != "datadir:/test ,Discover:1" {
		t.Errorf("opts to string is error :%s", str)
	}
}
