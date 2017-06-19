package network

import (
	"sync"
	"net"
	"math/rand"
	"container/list"
	"copernicus/crypto"
	"encoding/binary"
	"github.com/siddontang/go/log"
	"time"
	"os"
	"encoding/json"
	"strconv"
	"copernicus/protocol"
	"encoding/base32"
	"strings"
	"fmt"
)

const (
	NEW_BUCKET_COUNT         = 1024
	TRIED_BUCKET_COUNT       = 64
	TRIED_BUCKETS_PEER_GROUP = 8
	NUM_MISSING_DAYS         = 30
	NUMRETIES                = 3
	MIN_BAD_DAYS             = 7
	MAX_FAILURES             = 10
	NEW_BUCKETS_PEER_ADDRESS = 8
	NEW_BUCKETS_PEER_GROUP   = 64
	NEW_BUCKET_SIZE          = 64
	DUMP_ADDRESS_INTERVAL    = time.Minute * 10
	SERIALISATION_VERSION    = 1
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
func (addressManager *NetAddressManager) pickTried(bucket int) *list.Element {
	var oldest *KnownAddress
	var oldestElem *list.Element
	for e := addressManager.addressTried[bucket].Front(); e != nil; e.Next() {
		knownAddress := e.Value.(*KnownAddress)
		if oldest == nil || oldest.NetAddress.Timestamp.After(knownAddress.NetAddress.Timestamp) {
			oldestElem = e
			oldest = knownAddress
		}
	}
	return oldestElem
}
func (addressManager *NetAddressManager) getTriedBucket(netAddress *PeerAddress) int {
	dataFrist := []byte{}
	dataFrist = append(dataFrist, addressManager.key[:]...)
	dataFrist = append(dataFrist, []byte(netAddress.NetAddressKey())...)
	hashFrist := crypto.DoubleSha256Bytes(dataFrist)
	hash64 := binary.LittleEndian.Uint64(hashFrist)
	hash64 %= TRIED_BUCKETS_PEER_GROUP
	var hashBuf [8]byte
	binary.LittleEndian.PutUint64(hashBuf[:], hash64)
	dataSecond := []byte{}
	dataSecond = append(dataSecond, addressManager.key[:]...)
	dataSecond = append(dataSecond, netAddress.GroupKey()...)
	dataSecond = append(dataSecond, hashBuf[:]...)
	hashSecond := crypto.DoubleSha256Bytes(dataSecond)
	return int(binary.LittleEndian.Uint64(hashSecond) % TRIED_BUCKET_COUNT)

}
func (addressManager *NetAddressManager) savePeers() {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	serializedAddressManager := new(SerializedAddressManager)
	serializedAddressManager.Version = SERIALISATION_VERSION
	copy(serializedAddressManager.Key[:], addressManager.key[:])
	serializedAddressManager.Addresses = make([]*SerializedKnownAddress, len(addressManager.addressIndex))
	i := 0
	for k, v := range addressManager.addressIndex {
		serializedKnownAddress := new(SerializedKnownAddress)
		serializedKnownAddress.AddressString = k
		serializedKnownAddress.TimeStamp = v.NetAddress.Timestamp.Unix()
		serializedKnownAddress.Source = v.SrcAddress.NetAddressKey()
		serializedKnownAddress.LastAttempt = v.LastAttempt.Unix()
		serializedKnownAddress.LastSuccess = v.lastSuccess.Unix()
		serializedAddressManager.Addresses[i] = serializedKnownAddress
		i++
	}
	for i := range addressManager.addressNew {
		serializedAddressManager.NewBuckets[i] = make([]string, len(addressManager.addressNew[i]))
		j := 0
		for k := range addressManager.addressNew[i] {
			serializedAddressManager.NewBuckets[i][j] = k
			j++
		}
	}
	for i := range addressManager.addressTried {
		serializedAddressManager.TriedBuckets[i] = make([]string, addressManager.addressTried[i].Len())
		j := 0
		for e := addressManager.addressTried[i].Front(); e != nil; e = e.Next() {
			knownAddress := e.Value.(*KnownAddress)
			serializedAddressManager.TriedBuckets[i][j] = knownAddress.NetAddress.NetAddressKey()
			j++
		}
	}
	w, err := os.Create(addressManager.peersFile)
	if err != nil {
		log.Errorf("Error opening file %s :%v", addressManager.peersFile, err)
		return
	}
	newEncoder := json.NewEncoder(w)
	defer w.Close()

	if err := newEncoder.Encode(&serializedAddressManager); err != nil {
		log.Errorf("Failed to encode file %s :%v", addressManager.peersFile, err)
	}
}
func (addressManager *NetAddressManager) addressHandler() {
	dumpAddressTicker := time.NewTicker(DUMP_ADDRESS_INTERVAL)
	defer dumpAddressTicker.Stop()
out:
	for {
		select {
		case <-dumpAddressTicker.C:
			addressManager.savePeers()

		case <-addressManager.quit:
			break out
		}
	}
	addressManager.savePeers()
	addressManager.waitGroup.Done()
	log.Trace("address handler done ")
}
func (addressManager *NetAddressManager) loadPeers() {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()

}

func (addressManager *NetAddressManager) DeserializeNetAddress(addressString string) (*PeerAddress, error) {
	host, portStr, err := net.SplitHostPort(addressString)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}
	return addressManager.HostToNetAddress(host, uint16(port), protocol.SF_NODE_NETWORK_AS_FULL_NODE)
}
func (addressManager *NetAddressManager) HostToNetAddress(host string, port uint16, servicesFlag protocol.ServiceFlag) (*PeerAddress, error) {
	var ip net.IP
	if len(host) == 22 && host[16:] == ".onion" {
		data, err := base32.StdEncoding.DecodeString(strings.ToUpper(host[:16]))
		if err != nil {
			return nil, err
		}
		prefix := []byte{0xfd, 0x87, 0xd8, 0x7e, 0xeb, 0x43}
		ip := net.IP(append(prefix, data...))
	} else if ip = net.ParseIP(host); ip == nil {
		ips, err := addressManager.lookupFunc(host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses found for %s", host)
		}
		ip = ips[0]
	}
	return InitPeerAddressIPPort(servicesFlag, ip, port), nil
}
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
