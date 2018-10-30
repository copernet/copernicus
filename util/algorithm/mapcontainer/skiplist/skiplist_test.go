package skiplist

import (
	"math/rand"
	"testing"

	"github.com/copernet/copernicus/util/algorithm/mapcontainer"
	"github.com/stretchr/testify/assert"
	"sort"
)

type intless int

func (il intless) Less(other mapcontainer.Lesser) bool {
	return il < other.(intless)
}

func TestHeight(t *testing.T) {
	mapper := New(1 << 7)
	sk := mapper.(*skiplist)
	assert.Equal(t, len(sk.header.nexts), 7)

	mapper = New(3)
	sk = mapper.(*skiplist)
	assert.Equal(t, len(sk.header.nexts), 2)

	mapper = New(1)
	sk = mapper.(*skiplist)
	assert.Equal(t, len(sk.header.nexts), 1)

	mapper = New(6)
	sk = mapper.(*skiplist)
	assert.Equal(t, len(sk.header.nexts), 3)
}

func testsort(t *testing.T, testdata []intless) {
	sk := New(len(testdata))

	_, found := sk.Min()
	assert.False(t, found)
	_, found = sk.Max()
	assert.False(t, found)

	nodesMap := make(map[intless]*node)
	for i, td := range testdata {
		nodesMap[td] = sk.ReplaceOrInsert(intless(td)).(*node)
		minp, maxp := findMinMax(testdata[:i+1])

		minc, found := sk.Min()
		assert.True(t, found)
		assert.Equal(t, minp, minc)

		maxc, found := sk.Max()
		assert.True(t, found)
		assert.Equal(t, maxp, maxc)
	}
	prev := intless(-1)
	cnt := 0
	sk.Ascend(func(item mapcontainer.Lesser) bool {
		cur := item.(intless)
		assert.Contains(t, nodesMap, cur)
		assert.True(t, prev < cur, "%v not less than %v", prev, cur)
		prev = cur
		cnt++
		return true
	})
	assert.Equal(t, cnt, len(nodesMap))
	assert.Equal(t, cnt, sk.Len())
}

func TestSorted1(t *testing.T) {
	testsort(t, []intless{20, 10}[:])
}

func TestSorted2(t *testing.T) {
	testsort(t, []intless{10, 20}[:])
}

func TestSorted_sort(t *testing.T) {
	testsort(t, []intless{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}[:])
}

func TestSorted_reversed(t *testing.T) {
	testsort(t, []intless{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}[:])
}

func TestSorted_mixed(t *testing.T) {
	testsort(t, []intless{2, 0, 7, 6, 4, 3, 5, 1, 9, 8, 10})
}

func TestSorted_duplicated(t *testing.T) {
	testsort(t, []intless{1, 2, 2, 3, 1, 3, 1, 4, 6, 3, 5, 34, 5, 5, 467, 6, 56, 7, 5, 734, 56, 234, 5, 235, 23, 45, 35, 43, 45, 3, 45, 345})
}

func TestSorted_randomdata(t *testing.T) {
	size := rand.Intn(1 << 13)
	testdata := make([]intless, 0, size)
	for i := 0; i < size; i++ {
		testdata = append(testdata, intless(rand.Intn(1000)))
	}
	testsort(t, testdata)
}

func TestSearch(t *testing.T) {
	sk := New(1 << 16)
	cap := 1 << 16
	testdata := make([]intless, 0, cap)
	for i := 0; i < cap; i++ {
		d := intless(rand.Intn(1000))
		testdata = append(testdata, d)
		sk.ReplaceOrInsert(d)
	}

	for i := 0; i < len(testdata); i++ {
		nl, found := sk.Search(testdata[i])
		assert.True(t, found)
		assert.Equal(t, testdata[i], nl)
	}
	_, found := sk.Search(intless(1001))
	assert.False(t, found)
}

func findMinMax(data []intless) (minp intless, maxp intless) {
	minp = data[0]
	maxp = data[0]
	for _, item := range data {
		if item.Less(minp) {
			minp = item
		}
		if maxp.Less(item) {
			maxp = item
		}
	}
	return
}

func TestDelete(t *testing.T) {
	insertP := perm(1000)
	sk := New(len(insertP))

	keySet := make(map[mapcontainer.Lesser]struct{})
	for _, item := range insertP {
		sk.ReplaceOrInsert(item)
		keySet[item] = struct{}{}
	}
	for i := range keySet {
		_, found := sk.Delete(i)
		assert.True(t, found)

		_, found = sk.Search(i)
		assert.False(t, found)
	}

	_, found := sk.Delete(intless(33333))
	assert.False(t, found)
}

func TestAscend(t *testing.T) {
	insertP := perm(1000)
	cnt := 0
	sk := New(len(insertP))

	for _, item := range insertP {
		sk.ReplaceOrInsert(item)
	}
	sk.Ascend(func(item mapcontainer.Lesser) bool {
		if cnt == 456 {
			return false
		}
		cnt++
		return true
	})
	assert.Equal(t, 456, cnt)
}

func TestDeleteMax(t *testing.T) {
	insertP := perm(1000)
	intP := make([]int, 0, len(insertP))
	intMap := make(map[int]struct{})

	sk := New(len(insertP))
	// _, found := sk.DeleteMax()
	// assert.False(t, found)

	for _, item := range insertP {
		if _, ok := intMap[int(item.(intless))]; !ok {
			intP = append(intP, int(item.(intless)))
			intMap[int(item.(intless))] = struct{}{}
		}

		sk.ReplaceOrInsert(item)
	}

	sort.Ints(intP)
	for i := len(intP) - 1; i >= 0; i-- {
		less, found := sk.DeleteMax()
		assert.True(t, found)
		assert.Equal(t, intless(intP[i]), less)
	}
	_, found := sk.DeleteMax()
	assert.False(t, found)
}

func perm(n int) (out []mapcontainer.Lesser) {
	for _, v := range rand.Perm(n) {
		out = append(out, intless(v))
	}
	return
}
func BenchmarkInsert1(b *testing.B) {
	b.StopTimer()
	insertP := perm(10000)
	b.StartTimer()

	i := 0
	for i < b.N {
		sk := New(10000)
		for _, item := range insertP {
			sk.ReplaceOrInsert(item)
			i++
			if i >= b.N {
				return
			}
		}
	}
}

func BenchmarkSequenceInsert(b *testing.B) {
	shift := uint(16)
	size := 1 << shift
	sk := New(1 << 30)
	//assert.Equal(b, len(sk.header.nexts), int(shift))
	for i := 0; i < size; i++ {
		sk.ReplaceOrInsert(intless(rand.Intn(10000)))
	}
}

func BenchmarkReverseInsert(b *testing.B) {
	shift := uint(16)
	size := 1 << shift
	sk := New(1 << 30)
	//assert.Equal(b, len(sk.header.nexts), int(shift))
	for i := size - 1; i >= 0; i-- {
		sk.ReplaceOrInsert(intless(rand.Intn(10000)))
	}
}
