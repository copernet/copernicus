package manager

import (
	"copernicus/peer"
	"copernicus/model"
	"copernicus/crypto"
	"sync"
	"container/list"
)

type BlockManager struct {
	server          *peer.PeerManager //todo mutual reference
	started         int32
	shutdown        int32
	chain           *model.Blockchain
	rejectedTxns    map[crypto.Hash]struct{}
	requestedTxns   map[crypto.Hash]struct{}
	requestedBlocks map[crypto.Hash]struct{}
	progressLogger  *BlockProgressLogger //todo do't need?
	syncPeer        *peer.ServerPeer     //todo mutual reference
	
	messageChan      chan interface{}
	waitGroup        sync.WaitGroup
	quit             chan struct{}
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	netCheckPoint    *model.Checkpoint
}
