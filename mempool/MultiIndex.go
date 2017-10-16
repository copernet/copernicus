package mempool

import (
	"fmt"

	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/utils"
)

//MultiIndex the struct for support mempool store node, to implement MultiIndex sort
type MultiIndex struct {
	poolNode              map[utils.Hash]*TxMempoolEntry //unique
	byDescendantScoreSort *algorithm.CacheMap            //ordered_non_unique; keys : TxMempoolEntry; m : map[byDescendantScore]([]Hash)
	byEntryTimeSort       *algorithm.CacheMap            //ordered_non_unique; keys : TxMempoolEntry; m : map[byEntryTime]([]Hash)
	byScoreSort           *algorithm.CacheMap            //ordered_unique; 	   keys : TxMempoolEntry; m : map[byScore]Hash
	byAncestorFeeSort     *algorithm.CacheMap            //ordered_non_unique; keys : TxMempoolEntry; m : map[byAncestorFee]([]Hash)
}

func NewMultiIndex() *MultiIndex {
	multi := MultiIndex{}
	multi.poolNode = make(map[utils.Hash]*TxMempoolEntry)
	multi.byDescendantScoreSort = algorithm.NewCacheMap(CompareTxMemPoolEntryByDescendantScore)
	multi.byEntryTimeSort = algorithm.NewCacheMap(CompareTxMemPoolEntryByEntryTime)
	multi.byScoreSort = algorithm.NewCacheMap(CompareTxMempoolEntryByScore)
	multi.byAncestorFeeSort = algorithm.NewCacheMap(CompareTxMemPoolEntryByAncestorFee)

	return &multi
}

//AddElement add the element to the multiIndex; the element must meet multiIndex's keys various criterions;
func (multiIndex *MultiIndex) AddElement(hash utils.Hash, txEntry *TxMempoolEntry) {
	if _, has := multiIndex.poolNode[hash]; has {
		return
	}
	multiIndex.poolNode[hash] = txEntry
	multiIndex.byScoreSort.Add(txEntry, hash)
	multiIndex.byDescendantScoreSort.Add(txEntry, hash)
	for i, v := range multiIndex.byDescendantScoreSort.GetAllKeys() {
		txEntry := v.(*TxMempoolEntry)
		fmt.Printf("index : %v, hash : %v\n ", i, txEntry.TxRef.Hash.ToString())
	}
	fmt.Println("--------------------------------------------------------------")
	multiIndex.byEntryTimeSort.Add(txEntry, hash)
	multiIndex.byAncestorFeeSort.Add(txEntry, hash)
}

//GetEntryByHash : return the key correspond value In multiIndex;
//And modify The return value will be Influence the multiIndex;
func (multiIndex *MultiIndex) GetEntryByHash(hash utils.Hash) *TxMempoolEntry {
	if v, ok := multiIndex.poolNode[hash]; ok {
		return v
	}
	return nil
}

//DelEntryByHash : delete the key correspond value In multiIndex;
func (multiIndex *MultiIndex) DelEntryByHash(hash utils.Hash) {
	if v, ok := multiIndex.poolNode[hash]; ok {
		delete(multiIndex.poolNode, hash)
		multiIndex.byAncestorFeeSort.Del(v)
		multiIndex.byEntryTimeSort.Del(v)
		multiIndex.byScoreSort.Del(v)
		multiIndex.byDescendantScoreSort.Del(v)
	}
}

//GetByDescendantScoreSort : return the sort slice by cendantScore
func (multiIndex *MultiIndex) GetByDescendantScoreSort() []interface{} {
	keys := multiIndex.byDescendantScoreSort.GetAllKeys()
	retKey := make([]interface{}, len(keys))
	copy(retKey, keys)
	return retKey
}

func (multiIndex *MultiIndex) GetByDescendantScoreSortBegin() interface{} {
	keys := multiIndex.byDescendantScoreSort.GetAllKeys()
	if len(keys) > 0 {
		return keys[0]
	}
	return nil
}

func (multiIndex *MultiIndex) GetbyScoreSort() []interface{} {
	keys := multiIndex.byScoreSort.GetAllKeys()
	retKey := make([]interface{}, len(keys))
	copy(retKey, keys)
	return retKey
}

func (multiIndex *MultiIndex) GetbyScoreSortBegin() interface{} {
	keys := multiIndex.byScoreSort.GetAllKeys()
	if len(keys) > 0 {
		return keys[0]
	}
	return nil
}

func (multiIndex *MultiIndex) GetbyEntryTimeSort() []interface{} {
	keys := multiIndex.byEntryTimeSort.GetAllKeys()
	retKey := make([]interface{}, len(keys))
	copy(retKey, keys)
	return retKey
}

func (multiIndex *MultiIndex) GetbyEntryTimeSortBegin() interface{} {
	keys := multiIndex.byEntryTimeSort.GetAllKeys()
	if len(keys) > 0 {
		return keys[0]
	}
	return nil
}

func (multiIndex *MultiIndex) GetbyAncestorFeeSort() []interface{} {
	keys := multiIndex.byAncestorFeeSort.GetAllKeys()
	retKey := make([]interface{}, len(keys))
	copy(retKey, keys)
	return retKey
}

func (multiIndex *MultiIndex) GetbyAncestorFeeSortBegin() interface{} {
	keys := multiIndex.byAncestorFeeSort.GetAllKeys()
	if len(keys) > 0 {
		return keys[0]
	}
	return nil
}

func (multiIndex *MultiIndex) Size() int {
	return len(multiIndex.poolNode)
}
