package conn

import (
	"fmt"
	"github.com/btccom/copernicus/msg"
	"github.com/btccom/copernicus/network"
	"github.com/btccom/copernicus/protocol"
	"github.com/btccom/copernicus/utils"
	mrand "math/rand"
	"strconv"
	"time"
)

const (
	SecondsIn3Days int32 = 3 * 24 * 60 * 60
	SecondsIn4Days int32 = 4 * 24 * 60 * 60
)

type OnSeed func(addresses []*network.PeerAddress)

func SeedFromDNS(chainParams *msg.BitcoinParams, servicesFlag protocol.ServiceFlag,
	lookupFunc utils.LookupFunc, onSeed OnSeed) {
	for _, dnsSeed := range chainParams.DNSSeeds {
		var host string
		if !dnsSeed.HasFiltering || servicesFlag == protocol.SFNodeNetworkAsFullNode {
			host = dnsSeed.Host
		} else {
			host = fmt.Sprintf("x%x.%s", uint64(servicesFlag), dnsSeed.Host)
		}
		go func(host string) {
			randSource := mrand.New(mrand.NewSource(time.Now().UnixNano()))
			seedPeers, err := lookupFunc(host)
			if err != nil {
				log.Warn("DNS discovery failed on sedd %s :v%", host, err)
				return
			}
			numPeers := len(seedPeers)
			if numPeers == 0 {
				return
			}
			addresses := make([]*network.PeerAddress, len(seedPeers))
			intPort, _ := strconv.Atoi(chainParams.DefaultPort)
			for i, peer := range seedPeers {
				addresses[i] = network.NewPeerAddressTimestamp(
					// bitcoind seeds with addresses from
					// a time randomly selected between 3
					// and 7 days ago.
					time.Now().Add(-1*time.Second*time.Duration(SecondsIn3Days+
						randSource.Int31n(SecondsIn4Days))),
					0, peer, uint16(intPort))

			}
			onSeed(addresses)
		}(host)
	}
}
