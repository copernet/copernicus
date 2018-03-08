package p2p

import (
	"container/list"
	"sync"
	"sync/atomic"

	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type BlockManager struct {
	server          *PeerManager //todo mutual reference
	started         int32
	shutdown        int32
	Chain           *model.BlockChain
	rejectedTxns    map[utils.Hash]struct{}
	requestedTxns   map[utils.Hash]struct{}
	requestedBlocks map[utils.Hash]struct{}
	progressLogger  *BlockProgressLogger //todo do't need?
	syncPeer        *ServerPeer          //todo mutual reference

	messageChan      chan interface{}
	waitGroup        sync.WaitGroup
	quit             chan struct{}
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	netCheckPoint    *model.Checkpoint
}

func (blockManager *BlockManager) NewPeer(serverPeer *ServerPeer) {
	if atomic.LoadInt32(&blockManager.shutdown) != 0 {
		return
	}
	blockManager.messageChan <- &NewPeerMessage{serverPeer: serverPeer}

}
func (blockManager *BlockManager) Start() {
	//todo
}
func (blockManager *BlockManager) Stop() {

}
