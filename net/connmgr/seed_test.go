package connmgr

import (
	"errors"
	"net"
	"testing"

	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/net/wire"
)

func okLookIP(host string) ([]net.IP, error) {
	return []net.IP{
		net.ParseIP("138.68.19.3"),
		net.ParseIP("2001:19f0:4400:4afd:5400:ff:fe77:4358"),
		net.ParseIP("74.208.94.136"),
	}, nil
}

func failLookIP(host string) ([]net.IP, error) {
	return nil, errors.New("not found ip")
}

func TestSeedFromDNS(t *testing.T) {

	bp := &model.BitcoinParams{
		DNSSeeds: []model.DNSSeed{
			{Host: "seed.bitcoinabc.org", HasFiltering: true},
			{Host: "seed-abc.bitcoinforks.org", HasFiltering: false},
			{Host: "btccash-seeder.bitcoinunlimited.info", HasFiltering: true},
			{Host: "seed.bitprim.org", HasFiltering: false},
			{Host: "seed.deadalnix.me", HasFiltering: true},
			{Host: "seeder.criptolayer.net", HasFiltering: false},
		},
	}
	tests := []struct {
		param    *model.BitcoinParams
		flag     wire.ServiceFlag
		lookupFn LookupFunc
		checkFn  func(addrs []*wire.NetAddress)
	}{
		{bp, wire.SFNodeNetwork, LookupFunc(okLookIP), func(addrs []*wire.NetAddress) {
			if len(addrs) != 3 {
				t.Errorf("expect got 3 addrs at test %d", 1)
			}
		},
		},
		{bp, wire.SFNodeGetUTXO, LookupFunc(failLookIP), func(addrs []*wire.NetAddress) {
			if len(addrs) != 0 {
				t.Errorf("expect no addrs at test %d", 2)
			}
		},
		},
		{bp, wire.SFNodeGetUTXO | wire.SFNodeNetwork, LookupFunc(okLookIP), func(addrs []*wire.NetAddress) {
			if len(addrs) != 3 {
				t.Errorf("expect got 3 addrs at test %d", 3)
			}
		},
		},
	}

	for _, test := range tests {
		SeedFromDNS(test.param, test.flag, test.lookupFn, test.checkFn)
	}
}
