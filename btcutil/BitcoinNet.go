package btcutil

import "fmt"

// BitcoinNet represents which bitcoin network a msg belongs to.
type BitcoinNet uint32

const (
	MAIN_NET   BitcoinNet = 0xd9b4bef9
	TEST_NET   BitcoinNet = 0xdab5bffa
	TEST_NET_3 BitcoinNet = 0x0709110b
	SIM_NET    BitcoinNet = 0x12141c16
)

func (n BitcoinNet) ToString() string {
	var bnStrings = map[BitcoinNet]string{
		MAIN_NET:   "MainNet",
		TEST_NET:   "TestNet",
		TEST_NET_3: "TestNet3",
		SIM_NET:    "SimNet",
	}
	result, exist := bnStrings[n]
	if exist {
		return result
	}
	return fmt.Sprintf("Unkonwn BitcoinNet %d", uint32(n))
}
