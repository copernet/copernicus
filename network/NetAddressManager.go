package network

import (
	"container/list"
	crand "crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/protocol"
	"github.com/btcboost/copernicus/utils"

	beegoUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/core"
)

const (
	BucketCount           = 1024
	TriedBucketCount      = 64
	TriedBucketsPeerGroup = 8
	NumMissingDays        = 30
	NumReties             = 3
	MinBadDays            = 7
	MaxFailures           = 10
	BucketsPeerAddress    = 8
	NewBucketsPeerGroup   = 64
	NewBucketSize         = 64
	DumpAddressInterval   = time.Minute * 10
	SerialisationVersion  = 1
	NeedAddressThreshold  = 1000
	AddressPercent        = 23
	AddressMax            = 2500
	TriedBucketSize       = 256
)

var log = logs.NewLogger()

type NetAddressManager struct {
	lock           sync.Mutex
	peersFile      string
	lookupFunc     utils.LookupFunc
	rand           *rand.Rand
	key            [32]byte
	addressIndex   *beegoUtils.BeeMap
	addressNew     [BucketCount]*beegoUtils.BeeMap
	addressTried   [TriedBucketCount]*list.List
	started        int32
	shutdown       int32
	waitGroup      sync.WaitGroup
	quit           chan struct{}
	numTried       int
	numNew         int
	localAddresses map[string]*LocalAddress
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
		if knownAddress.refs == BucketsPeerAddress {
			return
		}
		factor := int32(2 * knownAddress.refs)
		if factor > 0 && addressManager.rand.Int31n(factor) != 0 {
			return
		}

	} else {
		netaddressCopy := *netAddress
		knownAddress = &KnownAddress{NetAddress: &netaddressCopy, SrcAddress: srcAddress}
		addressManager.addressIndex.Set(addressString, knownAddress)
		addressManager.numNew++
	}
	bucket := addressManager.getNewBucket(netAddress, srcAddress)
	ok := addressManager.addressNew[bucket].Check(addressString)
	if ok {
		return
	}
	if addressManager.addressNew[bucket].Count() > NewBucketSize {
		log.Trace("new bucket is full ,expiring old")
		addressManager.expireNew(bucket)

	}
	knownAddress.refs++
	addressManager.addressNew[bucket].Set(addressString, knownAddress)
	log.Trace("Added new address %s for addressManager total of %d addresses", addressString, addressManager.numTried+addressManager.numNew)
}
func (addressManager *NetAddressManager) expireNew(bucket int) {
	var oldest *KnownAddress
	for k, v := range addressManager.addressNew[bucket].Items() {
		knownAddrssValue := v.(*KnownAddress)
		if knownAddrssValue.IsBad() {
			log.Trace("expiring bad address %v", k)
			addressManager.addressNew[bucket].Delete(k)
			knownAddrssValue.refs--
			if knownAddrssValue.refs == 0 {
				addressManager.numNew--
				addressManager.addressIndex.Delete(k)
			}

			continue
		}
		if oldest == nil {
			oldest = knownAddrssValue
		} else if !knownAddrssValue.NetAddress.Timestamp.After(oldest.NetAddress.Timestamp) {
			oldest = knownAddrssValue
		}
	}
	if oldest != nil {
		key := oldest.NetAddress.NetAddressKey()
		log.Trace("expiring oldest address %v", key)
		addressManager.addressNew[bucket].Delete(key)
		oldest.refs--
		if oldest.refs == 0 {
			addressManager.numNew--
			addressManager.addressIndex.Delete(key)

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
	dataFirst := []byte{}
	dataFirst = append(dataFirst, addressManager.key[:]...)
	dataFirst = append(dataFirst, []byte(netAddress.NetAddressKey())...)
	hashFirst := core.DoubleSha256Bytes(dataFirst)
	hash64 := binary.LittleEndian.Uint64(hashFirst)
	hash64 %= TriedBucketsPeerGroup
	var hashBuf [8]byte
	binary.LittleEndian.PutUint64(hashBuf[:], hash64)
	dataSecond := []byte{}
	dataSecond = append(dataSecond, addressManager.key[:]...)
	dataSecond = append(dataSecond, netAddress.GroupKey()...)
	dataSecond = append(dataSecond, hashBuf[:]...)
	hashSecond := core.DoubleSha256Bytes(dataSecond)
	return int(binary.LittleEndian.Uint64(hashSecond) % TriedBucketCount)

}
func (addressManager *NetAddressManager) savePeers() {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	serializedAddressManager := new(SerializedAddressManager)
	serializedAddressManager.Version = SerialisationVersion
	copy(serializedAddressManager.Key[:], addressManager.key[:])
	serializedAddressManager.Addresses = make([]*SerializedKnownAddress, addressManager.addressIndex.Count())
	i := 0
	for k, v := range addressManager.addressIndex.Items() {
		serializedKnownAddress := new(SerializedKnownAddress)
		serializedKnownAddress.AddressString = k.(string)
		valueAddress := v.(*KnownAddress)
		serializedKnownAddress.TimeStamp = valueAddress.NetAddress.Timestamp.Unix()
		serializedKnownAddress.Source = valueAddress.SrcAddress.NetAddressKey()
		serializedKnownAddress.LastAttempt = valueAddress.LastAttempt.Unix()
		serializedKnownAddress.LastSuccess = valueAddress.lastSuccess.Unix()
		serializedAddressManager.Addresses[i] = serializedKnownAddress
		i++
	}
	for i := range addressManager.addressNew {
		serializedAddressManager.NewBuckets[i] = make([]string, addressManager.addressNew[i].Count())
		j := 0
		for k := range addressManager.addressNew[i].Items() {
			serializedAddressManager.NewBuckets[i][j] = k.(string)
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
		log.Error("Error opening file %s :%v", addressManager.peersFile, err)
		return
	}
	newEncoder := json.NewEncoder(w)
	defer w.Close()

	if err := newEncoder.Encode(&serializedAddressManager); err != nil {
		log.Error("Failed to encode file %s :%v", addressManager.peersFile, err)
	}
}
func (addressManager *NetAddressManager) addressHandler() {
	dumpAddressTicker := time.NewTicker(DumpAddressInterval)
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
	return addressManager.HostToNetAddress(host, uint16(port), protocol.SFNodeNetworkAsFullNode)
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
		log.Debug("%s", ip)
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
	return NewPeerAddressIPPort(servicesFlag, ip, port), nil
}
func (addressManager *NetAddressManager) getNewBucket(netAddr, srcAddr *PeerAddress) int {
	// bitcoind:
	// doublesha256(key + sourcegroup + int64(doublesha256(key + group + sourcegroup))%bucket_per_source_group) % num_new_buckets
	dataFirst := []byte{}
	dataFirst = append(dataFirst, addressManager.key[:]...)
	dataFirst = append(dataFirst, []byte(netAddr.GroupKey())...)
	dataFirst = append(dataFirst, []byte(srcAddr.GroupKey())...)
	hashFirst := core.DoubleSha256Bytes(dataFirst)
	hash64 := binary.LittleEndian.Uint64(hashFirst)
	hash64 %= NewBucketsPeerGroup
	var hashbuf [8]byte
	binary.LittleEndian.PutUint64(hashbuf[:], hash64)
	dataSecond := []byte{}
	dataSecond = append(dataSecond, addressManager.key[:]...)
	dataSecond = append(dataSecond, srcAddr.GroupKey()...)
	dataSecond = append(dataSecond, hashbuf[:]...)
	hashSecond := core.DoubleSha256Bytes(dataSecond)
	return int(binary.LittleEndian.Uint64(hashSecond) % BucketCount)

}
func (addressManager *NetAddressManager) Start() {
	if atomic.AddInt32(&addressManager.started, 1) != 1 {
		return
	}
	log.Trace("Starting address manager")
	addressManager.loadPeers()
	addressManager.waitGroup.Add(1)
	go addressManager.addressHandler()
}

func (addressManager *NetAddressManager) Stop() error {
	if atomic.AddInt32(&addressManager.shutdown, 1) != 1 {
		log.Warn("address manager is already in the process of shutting down ")
		return nil
	}
	log.Info("address manger shutting down")
	close(addressManager.quit)
	addressManager.waitGroup.Wait()
	return nil
}
func (addressManager *NetAddressManager) AddPeerAddresses(addresses []*PeerAddress, srcAddress *PeerAddress) {
	//addressManager.lock.Unlock()
	//defer addressManager.lock.Unlock()
	for _, peeraddress := range addresses {
		addressManager.updateAddress(peeraddress, srcAddress)
	}
}
func (addressManager *NetAddressManager) AddAddress(addr, srcAddr *PeerAddress) {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	addressManager.updateAddress(addr, srcAddr)
}
func (addressManager *NetAddressManager) AddAddressByIP(addressIP string) error {
	addr, porStr, err := net.SplitHostPort(addressIP)
	if err != nil {
		return err
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("invalid ip address %s", addr)
	}
	port, err := strconv.ParseUint(porStr, 10, 0)
	if err != nil {
		return fmt.Errorf("invalid port %s:%v", porStr, err)
	}
	peerAddress := NewPeerAddressIPPort(0, ip, uint16(port))
	addressManager.AddAddress(peerAddress, peerAddress)
	return nil
}
func (addressManager *NetAddressManager) Numaddresses() int {
	//addressManager.lock.Lock()
	//defer addressManager.lock.Unlock()
	return addressManager.addressIndex.Count()
}
func (addressManager *NetAddressManager) NeedMoreAddresses() bool {
	//addressManager.lock.Lock()
	//defer addressManager.lock.Unlock()
	return addressManager.Numaddresses() < NeedAddressThreshold
}
func (addressManager *NetAddressManager) find(peerAddress *PeerAddress) *KnownAddress {
	//addressManager.lock.Lock()
	//defer addressManager.lock.Unlock()
	value := addressManager.addressIndex.Get(peerAddress.NetAddressKey())
	if value == nil {
		return nil
	}
	return addressManager.addressIndex.Get(peerAddress.NetAddressKey()).(*KnownAddress)
}
func (addressManager *NetAddressManager) AddressCache() []*PeerAddress {
	//addressManager.lock.Lock()
	//defer addressManager.lock.Unlock()
	addressIndexLen := addressManager.addressIndex.Count()
	if addressIndexLen == 0 {
		return nil
	}
	allAddress := make([]*PeerAddress, 0, addressIndexLen)
	//for _, v := range addressManager.addressIndex {
	//	allAddress = append(allAddress, v.NetAddress)
	//}
	numAddresses := addressIndexLen * AddressPercent / 100
	if numAddresses > AddressMax {
		numAddresses = AddressMax
	}
	for i := 0; i < numAddresses; i++ {
		j := rand.Intn(addressIndexLen-i) + i
		allAddress[i], allAddress[j] = allAddress[j], allAddress[i]

	}
	return allAddress[0:numAddresses]

}
func (addressManager *NetAddressManager) reset() {
	addressManager.addressIndex = beegoUtils.NewBeeMap()
	io.ReadFull(crand.Reader, addressManager.key[:])
	for i := range addressManager.addressNew {
		addressManager.addressNew[i] = beegoUtils.NewBeeMap()
	}
	for i := range addressManager.addressTried {
		addressManager.addressTried[i] = list.New()
	}

}
func (addressManager *NetAddressManager) GetAddress() *KnownAddress {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	if addressManager.Numaddresses() == 0 {
		return nil
	}
	if addressManager.numTried > 0 && (addressManager.numNew == 0 || addressManager.rand.Intn(2) == 0) {
		large := 1 << 30
		factor := 1.0
		for {
			bucket := addressManager.rand.Intn(len(addressManager.addressTried))
			if addressManager.addressTried[bucket].Len() == 0 {
				continue
			}
			e := addressManager.addressTried[bucket].Front()
			for i := addressManager.rand.Int63n(int64(addressManager.addressTried[bucket].Len())); i > 0; i-- {
				e = e.Next()
			}
			knownAddress := e.Value.(*KnownAddress)
			randVal := addressManager.rand.Intn(large)
			if float64(randVal) < (factor * knownAddress.Chance() * float64(large)) {
				log.Trace("selected %v from tried bucket ", knownAddress.NetAddress.NetAddressKey())
			}
		}
	} else {
		large := 1 << 30
		factor := 1.0
		for {
			bucket := addressManager.rand.Intn(len(addressManager.addressNew))
			if addressManager.addressNew[bucket].Count() == 0 {
				continue
			}
			var knownAddress *KnownAddress
			nth := addressManager.rand.Intn(addressManager.addressNew[bucket].Count())
			for _, value := range addressManager.addressNew[bucket].Items() {
				if nth == 0 {
					knownAddress = value.(*KnownAddress)
				}
				nth--
			}
			randval := addressManager.rand.Intn(large)
			if float64(randval) < (factor * knownAddress.Chance() * float64(large)) {
				log.Trace("selected %v from new bucket", knownAddress.NetAddress.NetAddressKey())
				return knownAddress
			}
			factor *= 1.2

		}

	}

}
func (addressManager *NetAddressManager) Attempt(peerAddress *PeerAddress) {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	knownAddress := addressManager.find(peerAddress)
	if knownAddress == nil {
		return
	}
	knownAddress.attempts++
	knownAddress.LastAttempt = time.Now()
}
func (addressManager *NetAddressManager) Connected(peerAddress *PeerAddress) {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	knownAddress := addressManager.find(peerAddress)
	if knownAddress == nil {
		return
	}
	now := time.Now()
	if now.After(knownAddress.NetAddress.Timestamp.Add(time.Minute * 20)) {
		peerAddressCopy := knownAddress.NetAddress
		peerAddressCopy.Timestamp = time.Now()
		knownAddress.NetAddress = peerAddressCopy
	}
}

func (addressManager *NetAddressManager) MarkGood(address *PeerAddress) {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	knownAddress := addressManager.find(address)
	if knownAddress == nil {
		return
	}
	now := time.Now()
	knownAddress.lastSuccess = now
	knownAddress.LastAttempt = now
	knownAddress.attempts = 0
	if knownAddress.tried {
		return
	}
	addressKey := address.NetAddressKey()
	oldBucket := 1
	for i := range addressManager.addressNew {
		ok := addressManager.addressNew[i].Check(addressKey)
		if ok {
			addressManager.addressNew[i].Delete(addressKey)
			knownAddress.refs--
			if oldBucket == -1 {
				oldBucket = i
			}
		}
	}
	addressManager.numNew--
	if oldBucket == -1 {
		return
	}
	bucket := addressManager.getTriedBucket(knownAddress.NetAddress)
	if addressManager.addressTried[bucket].Len() < TriedBucketSize {
		knownAddress.tried = true
		addressManager.addressTried[bucket].PushBack(knownAddress)
		addressManager.numTried++
		return
	}
	entry := addressManager.pickTried(bucket)
	rmKnownAddress := entry.Value.(*KnownAddress)
	newBucket := addressManager.getNewBucket(rmKnownAddress.NetAddress, rmKnownAddress.SrcAddress)
	if addressManager.addressNew[newBucket].Count() >= NewBucketSize {
		newBucket = oldBucket
	}
	knownAddress.tried = true
	entry.Value = knownAddress
	rmKnownAddress.tried = false
	rmKnownAddress.refs++
	addressManager.numNew++

	rmKey := rmKnownAddress.NetAddress.NetAddressKey()
	log.Trace("Replacing %s with %s in tried", rmKey, addressKey)
	addressManager.addressNew[newBucket].Set(rmKey, rmKnownAddress)

}

func (addressManager *NetAddressManager) AddLocalAddress(peerAddress *PeerAddress, priority AddressPriority) error {
	if !peerAddress.IsRoutable() {
		return fmt.Errorf("addrss :%s is not routable ", peerAddress.IP)
	}
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	addressKey := peerAddress.NetAddressKey()
	localAddress, ok := addressManager.localAddresses[addressKey]
	if !ok || localAddress.score < priority {
		if ok {
			localAddress.score = priority + 1
		} else {
			addressManager.localAddresses[addressKey] = &LocalAddress{
				PeerAddress: peerAddress,
				score:       priority,
			}
		}
	}
	return nil
}

func (addressManager *NetAddressManager) GetBestLocalAddress(remoteAddress *PeerAddress) *PeerAddress {
	addressManager.lock.Lock()
	defer addressManager.lock.Unlock()
	var bestReachability Reachability
	var bestScore AddressPriority
	var bestAddress *PeerAddress
	for _, localAddress := range addressManager.localAddresses {
		reachability := GetReachabilityFrom(localAddress.PeerAddress, remoteAddress)
		if reachability > bestReachability || (reachability == bestReachability && localAddress.score > bestScore) {
			bestReachability = reachability
			bestScore = localAddress.score
			bestAddress = localAddress.PeerAddress

		}
	}
	if bestAddress != nil {
		log.Debug("suggesting address %s:%d for %s:%d", bestAddress.IP, bestAddress.Port, remoteAddress.IP, remoteAddress.Port)

	} else {
		log.Debug("No worthy address for %s:%d", remoteAddress.IP, remoteAddress.Port)
		var ip net.IP
		if !remoteAddress.IsIPv4() && !remoteAddress.IsOnionCatTor() {
			ip = net.IPv6zero
		} else {
			ip = net.IPv4zero
		}
		bestAddress = NewPeerAddressIPPort(protocol.SFNodeNetworkAsFullNode, ip, 0)

	}
	return bestAddress
}
func NewNetAddressManager(dataDir string, lookupFunc utils.LookupFunc) *NetAddressManager {
	addressManager := NetAddressManager{
		peersFile:      filepath.Join(dataDir, "p2p.json"),
		lookupFunc:     lookupFunc,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
		quit:           make(chan struct{}),
		localAddresses: make(map[string]*LocalAddress),
	}
	addressManager.reset()
	return &addressManager
}
