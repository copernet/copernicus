package skiplist

import (
	"math/rand"
	"testing"

	"github.com/copernet/copernicus/util/algorithm/mapcontainer"
	"github.com/stretchr/testify/assert"
)

type intless int

func (il intless) Less(other mapcontainer.Lesser) bool {
	return il < other.(intless)
}

func TestHeight(t *testing.T) {
	sk := New(1 << 7)
	assert.Equal(t, len(sk.header.nexts), 7)
}

func testsort(t *testing.T, testdata []intless) {
	sk := New(256)
	assert.Equal(t, len(sk.header.nexts), 8)

	nodesMap := make(map[intless]*node)
	for _, td := range testdata {
		nodesMap[td] = sk.ReplaceOrInsert(intless(td)).(*node)
	}

	iter := sk.CreateIterator()

	nless, ok := iter.Next()
	if !ok {
		return
	}
	cnt := 1
	prev := nless.(intless)
	assert.Contains(t, nodesMap, prev)
	for {
		nless, ok := iter.Next()
		if !ok {
			break
		}
		cnt++
		cur := nless.(intless)

		assert.True(t, prev < cur,
			"%v not less than %v", prev, cur)
	}

	assert.Equal(t, cnt, len(nodesMap))
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
	size := rand.Intn(1 << 18)
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
