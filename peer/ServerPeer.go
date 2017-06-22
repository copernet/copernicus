package peer

import (
	"copernicus/connect"
	"copernicus/crypto"
	"sync"
	"copernicus/msg"
	"github.com/btcsuite/btcutil/bloom"
	"copernicus/algorithm"
)

type ServerPeer struct {
	feeFilter       int64
	*Peer
	connectRequest  *connect.ConnectRequest
	peerManager     *PeerManager
	persistent      bool
	continueHash    *crypto.Hash
	relayLock       sync.Mutex
	disableRelayTx  bool
	setAddress      bool
	requestQueue    [] *msg.InventoryVector
	requestedTxns   map[crypto.Hash]struct{}
	requestedBlocks map[crypto.Hash]struct{}
	filter          *bloom.Filter
	knownAddress    map[string]struct{}
	banScore        algorithm.DynamicBanScore
	quit            chan struct{}
	txProcessed     chan struct{}
	blockProcessed  chan struct{}
}

func NewServerPeer(peerManager *PeerManager, isPersistent bool) (*ServerPeer) {
	serverPeer := ServerPeer{
		peerManager:     peerManager,
		persistent:      isPersistent,
		requestedTxns:   make(map[crypto.Hash]struct{}),
		requestedBlocks: make(map[crypto.Hash]struct{}),
		filter:          bloom.LoadFilter(nil),
		knownAddress:    make(map[string]struct{}),
		quit:            make(chan struct{}),
		txProcessed:     make(chan struct{}, 1),
		blockProcessed:  make(chan struct{}, 1),
		
	}
	return &serverPeer
}

