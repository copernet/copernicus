package protocol

import (
	"strings"
	"strconv"
)

type ServiceFlag uint64

func (f ServiceFlag) ToString() string {
	var orderedSFString = []ServiceFlag{
		SF_NODE_NETWORK_AS_FULL_NODE,
		SF_NODE_GET_UTXO,
		SF_NODE_BLOOM_FILTER,
	}
	var sfStrings = map[ServiceFlag]string{
		SF_NODE_NETWORK_AS_FULL_NODE: "SFNodeNetwork",
		SF_NODE_BLOOM_FILTER:         "SFNodeBloom",
		SF_NODE_GET_UTXO:             "SFNodeGetUTXO",
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
