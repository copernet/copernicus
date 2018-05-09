package mempool

import (
	"sync"
	"github.com/google/btree"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/outpoint"
)

var Gpool *TxMempool

// TxMempool is safe for concurrent write And read access.
type TxMempool struct {
	sync.RWMutex
	// current mempool best feerate for one transaction.
	fee util.FeeRate
	// poolData store the tx in the mempool
	poolData map[util.Hash]*TxEntry
	//NextTx key is txPrevout, value is tx.
	nextTx map[outpoint.OutPoint]*TxEntry
	//RootTx contain all root transaction in mempool.
	rootTx                  map[util.Hash]*TxEntry
	txByAncestorFeeRateSort btree.BTree
	timeSortData            btree.BTree
	cacheInnerUsage         int64
	checkFrequency          float64
	// sum of all mempool tx's size.
	totalTxSize uint64
	//transactionsUpdated mempool update transaction total number when create mempool late.
	transactionsUpdated uint64
}


func NewTxMempool() *TxMempool {
	t := &TxMempool{}
	t.fee = util.FeeRate{SataoshisPerK: 1}
	t.nextTx = make(map[outpoint.OutPoint]*TxEntry)
	t.poolData = make(map[util.Hash]*TxEntry)
	t.timeSortData = *btree.New(32)
	t.rootTx = make(map[util.Hash]*TxEntry)
	t.txByAncestorFeeRateSort = *btree.New(32)
	return t
}

func InitMempool() {
	Gpool = NewTxMempool()
}

type LockPoints struct {
	// Height and Time will be set to the blockChain height and median time past values that
	// would be necessary to satisfy all relative lockTime constraints (BIP68)
	// of this tx given our view of block chain history
	Height int
	Time   int64
	// MaxInputBlock as long as the current chain descends from the highest height block
	// containing one of the inputs used in the calculation, then the cached
	// values are still valid even after a reOrg.
	MaxInputBlock *blockindex
}

func NewLockPoints() *LockPoints {
	lockPoints := LockPoints{}
	return &lockPoints
}





