package connect

import (
	"copernicus/protocol"
	"copernicus/network"
	"fmt"
	mrand "math/rand"
	"time"
	"strconv"
	"copernicus/msg"
	"copernicus/utils"
)

const (
	SECONDS_IN_3_DAYS int32 = 3 * 24 * 60 * 60
	SECONDS_IN_4_DAYS int32 = 4 * 24 * 60 * 60
)

type OnSeed func(addresses []*network.PeerAddress)

func SeedFromDNS(chainParams *msg.BitcoinParams, servicesFlag protocol.ServiceFlag,
	lookupFunc utils.LookupFunc, onSeed OnSeed) {
	for _, dnsSeed := range chainParams.DNSSeeds {
		var host string
		if !dnsSeed.HasFiltering || servicesFlag == protocol.SF_NODE_NETWORK_AS_FULL_NODE {
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
					time.Now().Add(-1 * time.Second* time.Duration(SECONDS_IN_3_DAYS+
						randSource.Int31n(SECONDS_IN_4_DAYS))),
					0, peer, uint16(intPort))
				
			}
			onSeed(addresses)
		}(host)
	}
}
