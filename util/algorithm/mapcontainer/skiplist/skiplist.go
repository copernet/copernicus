package skiplist

import (
	"math"
	"math/rand"
	"time"

	"github.com/copernet/copernicus/util/algorithm/mapcontainer"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type node struct {
	mapcontainer.Lesser
	nexts []*node
}

type skiplist struct {
	header node
}

// NewSkipListWithHeight one of @estimatedNodeCount and @maxHeight is enough
func New(estimatedNodeCount int) *skiplist {
	return &skiplist{header: node{nexts: make([]*node,
		int64(math.Ceil(math.Log2(float64(estimatedNodeCount)))))}}
}

func isEqual(ls1 mapcontainer.Lesser, ls2 mapcontainer.Lesser) bool {
	return !ls1.Less(ls2) && !ls2.Less(ls1)
}

func (sl *skiplist) ReplaceOrInsert(elem mapcontainer.Lesser) mapcontainer.Lesser {
	insertToHeight := calcInsertionHeight(len(sl.header.nexts))
	newNode := &node{elem, make([]*node, insertToHeight)}
	prevs, _ := sl.removeIfAny(elem)

	for i := 0; i < len(newNode.nexts); i++ {
		newNode.nexts[i] = prevs[i].nexts[i]
		prevs[i].nexts[i] = newNode
	}
	return newNode
}

func (sl *skiplist) getMaxHeight() int {
	return len(sl.header.nexts)
}

func (sl *skiplist) searchPrev(target mapcontainer.Lesser) []*node {
	prevs := make([]*node, sl.getMaxHeight())

	for h := sl.getMaxHeight() - 1; h >= 0; h-- {
		prevs[h] = &sl.header
		for pv := prevs[h].nexts[h]; pv != nil && pv.Lesser.Less(target); pv = pv.nexts[h] {
			prevs[h] = pv
		}
	}
	return prevs
}

func (sl *skiplist) Search(target mapcontainer.Lesser) (mapcontainer.Lesser, bool) {
	prevs := sl.searchPrev(target)
	if prevs[0].nexts[0] != nil && isEqual(prevs[0].nexts[0].Lesser, target) {
		return prevs[0].nexts[0].Lesser, true
	}
	return nil, false
}
func (sl *skiplist) removeIfAny(target mapcontainer.Lesser) (prevs []*node, deleted *node) {
	prevs = sl.searchPrev(target)
	for i := 0; i < len(prevs) && prevs[i].nexts[i] != nil && isEqual(prevs[i].nexts[i].Lesser, target); i++ {
		deleted = prevs[i].nexts[i]
		prevs[i].nexts[i] = prevs[i].nexts[i].nexts[i]
	}
	return
}

func calcInsertionHeight(maxHeight int) int {
	res := 1
	for i := 0; i < maxHeight-1; i++ {
		upDown := rand.Intn(2)
		if upDown == 0 {
			break
		}
		res++
	}
	return res
}

func (sl *skiplist) Delete(target mapcontainer.Lesser) (deleted mapcontainer.Lesser, found bool) {
	_, deletedNode := sl.removeIfAny(target)
	if deletedNode != nil && isEqual(deletedNode.Lesser, target) {
		return deletedNode.Lesser, true
	}
	return nil, false
}

func (sl *skiplist) Min() mapcontainer.Lesser {
	return sl.header.nexts[0].Lesser
}

func (sl *skiplist) Ascend(iterHandler func(i mapcontainer.Lesser) bool) {
	next := sl.header.nexts[0]
	for next != nil {
		if !iterHandler(next.Lesser) {
			break
		}
		next = next.nexts[0]
	}
}
