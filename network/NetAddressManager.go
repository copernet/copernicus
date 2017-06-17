package network

import (
	"sync"
	"net"
	"math/rand"
	"container/list"
	"copernicus/crypto"
	"encoding/binary"
	"github.com/siddontang/go/log"
)

const (
	NEW_BUCKET_COUNT         = 1024
	TRIED_BUCKET_COUNT       = 64
	NUM_MISSING_DAYS         = 30
	NUMRETIES                = 3
	MIN_BAD_DAYS             = 7
	MAX_FAILURES             = 10
	NEW_BUCKETS_PEER_ADDRESS = 8
	NEW_BUCKETS_PEER_GROUP   = 64
	NEW_BUCKET_SIZE          = 64
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

func (addressManager *NetAddressManager) updateAddress(netAddress, srcAddress *PeerAddress) {
	if !netAddress.IsRoutable() {
		return
	}
	addressString := netAddress.NetAddressKey()

	knownAddress := addressManager.find(netAddress)
	if knownAddress != nil {
		if netAddress.Timestamp.After(knownAddress.NetAddress.Timestamp) ||
			(knownAddress.NetAddress.ServicesFlag&netAddress.ServicesFlag) != netAddress.ServicesFlag {
			peerAddressCopy := *knownAddress.NetAddress
			peerAddressCopy.Timestamp = netAddress.Timestamp
			peerAddressCopy.AddService(netAddress.ServicesFlag)
			knownAddress.NetAddress = &peerAddressCopy
		}
		if knownAddress.tried {
			return
		}
		if knownAddress.refs == NEW_BUCKETS_PEER_ADDRESS {
			return
		}
		factor := int32(2 * knownAddress.refs)
		if addressManager.rand.Int31n(factor) != 0 {
			return
		}

	} else {
		netaddressCopy := *netAddress
		knownAddress = &KnownAddress{NetAddress: &netaddressCopy, SrcAddress: srcAddress}
		addressManager.addressIndex[addressString] = knownAddress
		addressManager.numNew++
	}
	bucket := addressManager.getNewBucket(netAddress, srcAddress)
	_, ok := addressManager.addressNew[bucket][addressString]
	if ok {
		return
	}
	if len(addressManager.addressNew[bucket]) > NEW_BUCKET_SIZE {
		log.Trace("new bucket is full ,expiring old")
		addressManager.expireNew(bucket)

	}
	knownAddress.refs++
	addressManager.addressNew[bucket][addressString] = knownAddress
	log.Trace("Added new address %s for a total of %d addresses", addressString, addressManager.numTried+addressManager.numNew)
}
func (addressManager *NetAddressManager) expireNew(bucket int) {
	var oldest *KnownAddress
	for k, v := range addressManager.addressNew[bucket] {
		if v.IsBad() {
			log.Trace("expiring bad address %v", k)
			delete(addressManager.addressNew[bucket], k)
			v.refs--
			if v.refs == 0 {
				addressManager.numNew--
				delete(addressManager.addressIndex, k)
			}

			continue
		}
		if oldest == nil {
			oldest = v
		} else if !v.NetAddress.Timestamp.After(oldest.NetAddress.Timestamp) {
			oldest = v
		}
	}
	if oldest != nil {
		key := oldest.NetAddress.NetAddressKey()
		log.Trace("expiring oldest address %v", key)
		delete(addressManager.addressNew[bucket], key)
		oldest.refs--
		if oldest.refs == 0 {
			addressManager.numNew--
			delete(addressManager.addressIndex, key)

		}

	}

}
func (addressManager *NetAddressManager)pickTried()
func (addressManger *NetAddressManager) getNewBucket(netAddr, srcAddr *PeerAddress) int {
	// bitcoind:
	// doublesha256(key + sourcegroup + int64(doublesha256(key + group + sourcegroup))%bucket_per_source_group) % num_new_buckets
	dataFirst := []byte{}
	dataFirst = append(dataFirst, addressManger.key[:]...)
	dataFirst = append(dataFirst, []byte(netAddr.GroupKey())...)
	dataFirst = append(dataFirst, []byte(srcAddr.GroupKey())...)
	hashFirst := crypto.DoubleSha256Bytes(dataFirst)
	hash64 := binary.LittleEndian.Uint64(hashFirst)
	hash64 %= NEW_BUCKETS_PEER_GROUP
	var hashbuf [8]byte
	binary.LittleEndian.PutUint64(hashbuf[:], hash64)
	dataSecond := []byte{}
	dataSecond = append(dataSecond, addressManger.key[:]...)
	dataSecond = append(dataSecond, srcAddr.GroupKey()...)
	dataSecond = append(dataSecond, hashbuf[:]...)
	hashSecond := crypto.DoubleSha256Bytes(dataSecond)
	return int(binary.LittleEndian.Uint64(hashSecond) % NEW_BUCKET_COUNT)

}
func (a *NetAddressManager) find(peerAddress *PeerAddress) *KnownAddress {
	return a.addressIndex[peerAddress.NetAddressKey()]
}
