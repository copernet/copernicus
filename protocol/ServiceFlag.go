package protocol

import (
	"strings"
	"strconv"
)

type ServiceFlag uint64

func (f ServiceFlag) ToString() string {
	var orderedSFString = []ServiceFlag{
		SFNodeNetworkAsFullNode,
		SFNodeGetUtxo,
		SFNodeBloomFilter,
	}
	var sfStrings = map[ServiceFlag]string{
		SFNodeNetworkAsFullNode: "SFNodeNetwork",
		SFNodeBloomFilter:       "SFNodeBloom",
		SFNodeGetUtxo:           "SFNodeGetUTXO",
	}
	if f == 0 {
		return "0x0"
	}
	s := ""
	for _, flag := range orderedSFString {
		if f&flag == flag {
			s += sfStrings[flag] + "|"
			f -= flag
		}
	}
	s = strings.TrimRight(s, "|")
	if f != 0 {
		s += "|0x" + strconv.FormatUint(uint64(f), 16)
	}
	s = strings.TrimLeft(s, "|")
	return s
}
