package protocol

import "copernicus/network"

type Params struct {
	Name        string
	BitcoinNet  BitcoinNet
	DefaultPort string
	DNSSeeds []network.DNSSeed

}
