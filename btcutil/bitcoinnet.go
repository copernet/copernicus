package btcutil

import "fmt"

// BitcoinNet represents which bitcoin network a msg belongs to.
type BitcoinNet uint32

const (
	MainNet  BitcoinNet = 0xd9b4bef9
	TestNet  BitcoinNet = 0xdab5bffa
	TestNet3 BitcoinNet = 0x0709110b
	SimNet   BitcoinNet = 0x12141c16
)

func (n BitcoinNet) ToString() string {
	var bnStrings = map[BitcoinNet]string{
		MainNet:  "MainNet",
		TestNet:  "TestNet",
		TestNet3: "TestNet3",
		SimNet:   "SimNet",
	}
	result, exist := bnStrings[n]
	if exist {
		return result
	}
	return fmt.Sprintf("Unknown BitcoinNet %d", uint32(n))
}
