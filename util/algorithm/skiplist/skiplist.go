package skiplist

import (
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type Lesser interface {
	Less(other Lesser) bool
}

type MapContainer interface {
	Insert(Lesser) bool
	Search(Lesser)
}

type MapContainerIterator interface {
	Next() (Lesser, bool)
}

type node struct {
	Lesser
	nexts []*node
}

type skiplist struct {
	//headers []*node

	header node
}

type skiplistIterator struct {
	cur *node
}

func (sli *skiplistIterator) Next() (elem Lesser, found bool) {
	if sli.cur.nexts[0] != nil {
		sli.cur = sli.cur.nexts[0]
		return sli.cur.Lesser, true
	}
	return nil, false
}

// NewSkipListWithHeight one of @estimatedNodeCount and @maxHeight is enough
func New(estimatedNodeCount int) *skiplist {
	return &skiplist{header: node{nexts: make([]*node,
		int64(math.Ceil(math.Log2(float64(estimatedNodeCount)))))}}
}

func isEqual(ls1 Lesser, ls2 Lesser) bool {
	return !ls1.Less(ls2) && !ls2.Less(ls1)
}

func (sl *skiplist) CreateIterator() MapContainerIterator {
	return &skiplistIterator{&sl.header}
}

func (sl *skiplist) InsertOrReplace(elem Lesser) Lesser {
	insertToHeight := calcInsertionHeight(len(sl.header.nexts))

	newNode := &node{elem, make([]*node, insertToHeight)}

	prevs := sl.removeIfAny(elem)
	// do insertion
	for i := 0; i < len(newNode.nexts); i++ {
		newNode.nexts[i] = prevs[i].nexts[i]
		prevs[i].nexts[i] = newNode
	}
	return newNode
}

var none Lesser

func (sl *skiplist) searchPrev(level int, target Lesser) []*node {
	prevs := make([]*node, level+1)
	for i := 0; i < len(prevs); i++ {
		prevs[i] = &sl.header
	}

	for h := level; h >= 0; h-- {
		for pv := prevs[h].nexts[h]; pv != nil && pv.Lesser.Less(target); pv = pv.nexts[h] {
			prevs[h] = pv
		}
	}
	return prevs
}

func (sl *skiplist) Search(target Lesser) (Lesser, bool) {
	prevs := sl.searchPrev(len(sl.header.nexts)-1, target)
	if prevs[0].nexts[0] != nil && isEqual(prevs[0].nexts[0].Lesser, target) {
		return prevs[0].nexts[0].Lesser, true
	}
	return nil, false
}
func (sl *skiplist) removeIfAny(target Lesser) []*node {
	prevs := sl.searchPrev(len(sl.header.nexts)-1, target)
	for i := 0; i < len(prevs) && prevs[i] != nil && prevs[i].nexts[i] != nil && isEqual(prevs[i].nexts[i].Lesser, target); i++ {
		prevs[i].nexts[i] = prevs[i].nexts[i].nexts[i]
	}
	return prevs
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
