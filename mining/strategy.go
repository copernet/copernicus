package mining

import (
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/utils"
	"github.com/google/btree"
	"github.com/spf13/viper"
)

type sortType int

const (
	sortByFee sortType = 1 << iota
	sortByFeeRate
)

const defaultSortStrategy = sortByFeeRate

var strategy sortType

var strategies = map[string]sortType{
	"ancestorfee":     sortByFee,
	"ancestorfeerate": sortByFeeRate,
}

// EntryFeeSort TxEntry sorted by feeWithAncestors
type EntryFeeSort mempool.TxEntry

func (e EntryFeeSort) Less(than btree.Item) bool {
	t := than.(EntryFeeSort)
	if e.SumFeeWithAncestors == t.SumFeeWithAncestors {
		return e.Tx.Hash.Cmp(&t.Tx.Hash) > 0
	}
	return e.SumFeeWithAncestors > than.(EntryFeeSort).SumFeeWithAncestors
}

func sortedByFeeWithAncestors() *btree.BTree {
	b := btree.New(32)
	for _, txentry := range blockchain.GMemPool.PoolData {
		b.ReplaceOrInsert(EntryFeeSort(*txentry))
	}
	return b
}

// EntryAncestorFeeRateSort TxEntry sorted by feeRateWithAncestors
type EntryAncestorFeeRateSort mempool.TxEntry

func (r EntryAncestorFeeRateSort) Less(than btree.Item) bool {
	t := than.(EntryAncestorFeeRateSort)
	b1 := utils.NewFeeRateWithSize((r).SumFeeWithAncestors, r.SumSizeWitAncestors).SataoshisPerK
	b2 := utils.NewFeeRateWithSize(t.SumFeeWithAncestors, t.SumSizeWitAncestors).SataoshisPerK
	if b1 == b2 {
		return r.Tx.Hash.Cmp(&t.Tx.Hash) > 0
	}
	return b1 > b2
}

func sortedByFeeRateWithAncestors() *btree.BTree {
	b := btree.New(32)
	for _, txentry := range blockchain.GMemPool.PoolData {
		b.ReplaceOrInsert(EntryAncestorFeeRateSort(*txentry))
	}
	return b
}

func init() {
	sortParam := viper.GetString("strategy")
	ret, ok := strategies[sortParam]
	if !ok {
		logs.Info("the specified strategy< %s > is not exist, so use default strategy< %s >", sortParam, defaultSortStrategy)
		strategy = defaultSortStrategy
	}
	strategy = ret
}
