package mempool

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

type CoinsViewMemPool struct {
	Base  utxo.CoinsView
	Mpool *TxMempool
}

func (m *CoinsViewMemPool) GetCoin(point *core.OutPoint, coin *utxo.Coin) bool {
	// If an entry in the mempool exists, always return that one, as it's
	// guaranteed to never conflict with the underlying cache, and it cannot
	// have pruned entries (as it contains full) transactions. First checking
	// the underlying cache risks returning a pruned entry instead.
	if ptx, ok := m.Mpool.PoolData[point.Hash]; ok {
		if int(point.Index) < len(ptx.Tx.Outs) {
			*coin = *utxo.NewCoin(ptx.Tx.Outs[point.Index], HeightMemPool, false)
			return true
		}
		return false
	}

	return m.Base.GetCoin(point, coin) && !coin.IsSpent()
}
func (m *CoinsViewMemPool) HaveCoin(point *core.OutPoint) bool {
	return m.Mpool.ExistsOutPoint(point) || m.Base.HaveCoin(point)
}

func (m *CoinsViewMemPool) GetBestBlock() utils.Hash {
	return m.Base.GetBestBlock()
}

func (m *CoinsViewMemPool) BatchWrite(coinsMap utxo.CacheCoins, hash *utils.Hash) bool {
	return m.Base.BatchWrite(coinsMap, hash)
}

func (m *CoinsViewMemPool) EstimateSize() uint64 {
	return m.Base.EstimateSize()
}

func NewCoinsViewMemPool(base utxo.CoinsView, mempool *TxMempool) *CoinsViewMemPool {
	return &CoinsViewMemPool{
		Base:  base,
		Mpool: mempool,
	}
}
