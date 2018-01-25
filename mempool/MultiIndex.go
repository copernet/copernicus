package mempool

import (
	"sort"

	"github.com/btcboost/copernicus/utils"
)

const (
	DESCENDANTSCORE = iota
	MININGSCORE
	ANCESTORSCORE
	TIMESORT
)

//MultiIndex the struct for support mempool store node, to implement MultiIndex sort
type MultiIndex struct {
	poolNode              map[utils.Hash]*TxMempoolEntry //unique
	nodeKey               []*TxMempoolEntry
	byDescendantScoreSort []*TxMempoolEntry //ordered_non_unique;
	byEntryTimeSort       []*TxMempoolEntry //ordered_non_unique;
	byScoreSort           []*TxMempoolEntry //ordered_unique;
	byAncestorFeeSort     []*TxMempoolEntry //ordered_non_unique;
}

func NewMultiIndex() *MultiIndex {
	multi := MultiIndex{}
	multi.poolNode = make(map[utils.Hash]*TxMempoolEntry)
	multi.byDescendantScoreSort = make([]*TxMempoolEntry, 0)
	multi.byEntryTimeSort = make([]*TxMempoolEntry, 0)
	multi.byScoreSort = make([]*TxMempoolEntry, 0)
	multi.byAncestorFeeSort = make([]*TxMempoolEntry, 0)
	multi.nodeKey = make([]*TxMempoolEntry, 0)

	return &multi
}

//AddElement add the element to the multiIndex; the element must meet multiIndex's keys various criterions;
func (multiIndex *MultiIndex) AddElement(hash utils.Hash, txEntry *TxMempoolEntry) {
	if _, has := multiIndex.poolNode[hash]; has {
		return
	}
	multiIndex.poolNode[hash] = txEntry
	multiIndex.nodeKey = append(multiIndex.nodeKey, txEntry)
}

//DelEntryByHash : delete the key correspond value In multiIndex;
func (multiIndex *MultiIndex) DelEntryByHash(hash utils.Hash) {
	if _, ok := multiIndex.poolNode[hash]; ok {
		delete(multiIndex.poolNode, hash)
		for i, v := range multiIndex.nodeKey {
			oriHash := v.TxRef.Hash
			if (&oriHash).IsEqual(&hash) {
				multiIndex.nodeKey = append(multiIndex.nodeKey[:i], multiIndex.nodeKey[i+1:]...)
				break
			}
		}
	}
}

//GetEntryByHash : return the key correspond value In multiIndex;
//And modify The return value will be Influence the multiIndex;
func (multiIndex *MultiIndex) GetEntryByHash(hash utils.Hash) *TxMempoolEntry {
	if v, ok := multiIndex.poolNode[hash]; ok {
		return v
	}
	return nil
}

func (multiIndex *MultiIndex) Size() int {
	return len(multiIndex.poolNode)
}

//GetByDescendantScoreSort : return the sort slice by descendantScore
func (multiIndex *MultiIndex) GetByDescendantScoreSort() []*TxMempoolEntry {
	multiIndex.updateSort(DESCENDANTSCORE)
	return multiIndex.byDescendantScoreSort
}

func (multiIndex *MultiIndex) GetByDescendantScoreSortBegin() interface{} {
	multiIndex.updateSort(DESCENDANTSCORE)
	if len(multiIndex.byDescendantScoreSort) > 0 {
		return multiIndex.byDescendantScoreSort[0]
	}
	return nil
}

func (multiIndex *MultiIndex) updateSort(flag int) {
	switch flag {
	case DESCENDANTSCORE:
		multiIndex.byDescendantScoreSort = make([]*TxMempoolEntry, len(multiIndex.nodeKey))
		copy(multiIndex.byDescendantScoreSort, multiIndex.nodeKey)
		sort.SliceStable(multiIndex.byDescendantScoreSort, func(i, j int) bool {
			return CompareTxMemPoolEntryByDescendantScore(multiIndex.byDescendantScoreSort[i], multiIndex.byDescendantScoreSort[j])
		})
	case ANCESTORSCORE:
		multiIndex.byAncestorFeeSort = make([]*TxMempoolEntry, len(multiIndex.nodeKey))
		copy(multiIndex.byAncestorFeeSort, multiIndex.nodeKey)
		sort.SliceStable(multiIndex.byAncestorFeeSort, func(i, j int) bool {
			return CompareTxMemPoolEntryByAncestorFee(multiIndex.byAncestorFeeSort[i], multiIndex.byAncestorFeeSort[j])
		})
	case MININGSCORE:
		multiIndex.byScoreSort = make([]*TxMempoolEntry, len(multiIndex.nodeKey))
		copy(multiIndex.byScoreSort, multiIndex.nodeKey)
		sort.SliceStable(multiIndex.byScoreSort, func(i, j int) bool {
			return CompareTxMempoolEntryByScore(multiIndex.byScoreSort[i], multiIndex.byScoreSort[j])
		})
	case TIMESORT:
		multiIndex.byEntryTimeSort = make([]*TxMempoolEntry, len(multiIndex.nodeKey))
		copy(multiIndex.byEntryTimeSort, multiIndex.nodeKey)
		sort.SliceStable(multiIndex.byEntryTimeSort, func(i, j int) bool {
			return CompareTxMemPoolEntryByEntryTime(multiIndex.byEntryTimeSort[i], multiIndex.byEntryTimeSort[j])
		})

	}
}

func (multiIndex *MultiIndex) GetbyEntryTimeSort() []*TxMempoolEntry {
	multiIndex.updateSort(TIMESORT)
	return multiIndex.byEntryTimeSort
}

func (multiIndex *MultiIndex) GetbyEntryTimeSortBegin() interface{} {
	multiIndex.updateSort(TIMESORT)
	if len(multiIndex.byEntryTimeSort) > 0 {
		return multiIndex.byEntryTimeSort[len(multiIndex.byEntryTimeSort)-1]
	}
	return nil
}

func (multiIndex *MultiIndex) GetbyAncestorFeeSort() []*TxMempoolEntry {
	multiIndex.updateSort(ANCESTORSCORE)
	return multiIndex.byAncestorFeeSort
}

func (multiIndex *MultiIndex) GetbyAncestorFeeSortBegin() interface{} {
	multiIndex.updateSort(ANCESTORSCORE)
	if len(multiIndex.byAncestorFeeSort) > 0 {
		return multiIndex.byAncestorFeeSort[len(multiIndex.byAncestorFeeSort)-1]
	}
	return nil
}

func (multiIndex *MultiIndex) GetbyScoreSort() []*TxMempoolEntry {
	multiIndex.updateSort(MININGSCORE)
	return multiIndex.byScoreSort
}

func (multiIndex *MultiIndex) GetbyScoreSortBegin() interface{} {
	multiIndex.updateSort(MININGSCORE)
	if len(multiIndex.byScoreSort) > 0 {
		return multiIndex.byScoreSort[len(multiIndex.byScoreSort)-1]
	}
	return nil
}
