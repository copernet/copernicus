package bitcoinutil

import (
	"github.com/btcboost/copernicus/net/wire"
)

const (
	MainNet  wire.BitcoinNet = 0xd9b4bef9
	TestNet  wire.BitcoinNet = 0xdab5bffa
	TestNet3 wire.BitcoinNet = 0x0709110b
	SimNet   wire.BitcoinNet = 0x12141c16
)

//func (n BitcoinNet) ToString() string {
//	var bnStrings = map[BitcoinNet]string{
//		MainNet:  "MainNet",
//		TestNet:  "TestNet",
//		TestNet3: "TestNet3",
//		SimNet:   "SimNet",
//	}
//	result, exist := bnStrings[n]
//	if exist {
//		return result
//	}
//	return fmt.Sprintf("Unknown BitcoinNet %d", uint32(n))
//}

type DNSSeed struct {
	Host         string
	HasFiltering bool
}
