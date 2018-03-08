package utils

import (
	"sort"
	"sync"
)

type Set struct {
	mVal map[float64]bool
	sync.Mutex
}

func NewSet() *Set {
	return &Set{mVal: make(map[float64]bool)}
}

func (s *Set) Add(item float64) {
	s.Lock()
	defer s.Unlock()
	s.mVal[item] = true
}

func (s *Set) Remove(item float64) {
	s.Lock()
	defer s.Unlock()
	delete(s.mVal, item)
}

func (s *Set) GetSortList() sort.Float64Slice {
	s.Lock()
	defer s.Unlock()
	list := sort.Float64Slice{}
	for item := range s.mVal {
		list = append(list, item)
	}
	sort.Sort(list)

	return list
}

type FeeFilterRounder struct {
	feeSet       Set
	insecureRand FastRandomContext
}

func NewFeeFilterRounder(minIncrementalFee FeeRate, fDeterministic bool) *FeeFilterRounder {
	feeFilterRounder := FeeFilterRounder{feeSet: *NewSet()}
	var minFeeLimit int64
	minIncFee := minIncrementalFee.GetFeePerK() / 2
	if minIncFee > 1 {
		minFeeLimit = minIncFee
	}

	minFeeLimit = 1
	feeFilterRounder.feeSet.Add(0)
	for bucketBoundary := float64(minFeeLimit); bucketBoundary <= float64(MAX_FEERATE); bucketBoundary *= FEE_SPACING {
		feeFilterRounder.feeSet.Add(bucketBoundary)
	}

	feeFilterRounder.insecureRand = *NewFastRandomContext(fDeterministic)
	return &feeFilterRounder
}

func (feeFilterRounder *FeeFilterRounder) Round(currentMinFee int64) int64 {
	list := feeFilterRounder.feeSet.GetSortList()
	index := list.Search(float64(currentMinFee))

	if index != 0 && feeFilterRounder.insecureRand.Rand32()%3 != 0 {
		index--
	}
	if index == len(list) && list[index] < float64(currentMinFee) {
		index--
	}
	return int64(list[index])
}
