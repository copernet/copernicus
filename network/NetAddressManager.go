package network

import (
	"sync"
	"net"
	"math/rand"
	"container/list"
	"copernicus/msg"
)

const (
	NEW_BUCKET_COUNT   = 1024
	TRIED_BUCKET_COUNT = 64
	NUM_MISSING_DAYS   = 30
	NUMRETIES          = 3
	MIN_BAD_DAYS       = 7
	MAX_FAILURES       = 10
)

type NetAddressManager struct {
	lock         sync.Mutex
	peersFile    string
	lookupFunc   func(string) ([]net.IP, error)
	rand         *rand.Rand
	key          [32]byte
	addressIndex map[string]*KnownAddress
	addressNew   [NEW_BUCKET_COUNT]map[string]*KnownAddress
	addressTried [TRIED_BUCKET_COUNT]*list.List
	started      int32
	shutdown     int32
	waitGroup    sync.WaitGroup
	quit         chan struct{}
	numTried     int
	numNew       int
	lamtx        sync.Mutex
	localAddress map[string]*LocalAddress
}

func (addressManager *NetAddressManager) updateAddress(netAddress, sourceAddress *msg.PeerAddress) {

}
