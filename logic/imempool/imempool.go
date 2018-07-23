package imempool

/*
import (
	mempool2 "github.com/copernet/copernicus/logic/mempool"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	core2 "github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/utxo"
)

type imempool interface {
	HasSpentOut(point *outpoint.OutPoint) bool
	LimitMempoolSize() []*outpoint.OutPoint
	RemoveUnFinalTx(*chain.Chain, *CoinsViewCache, int, int)
	RemoveTxSelf([]*tx.Tx)
	RemoveTxRecursive(*tx.Tx, mempool2.PoolRemovalReason)
	Check(*CoinsViewCache, int)
	GetCoin(point *outpoint.OutPoint) Coin
	GetRootTx() map[util.Hash]mempool.TxEntry
	GetAllTxEntry() map[util.Hash]*mempool.TxEntry
	FindTx(util.Hash) *core2.Tx
	Size() int
}
*/
