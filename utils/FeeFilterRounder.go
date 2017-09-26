package utils

import (
	"sort"
	"sync"
)

type ListFloat []float64

func (list ListFloat) Len() int {
	return len(list)
}

func (list ListFloat) Less(i, j int) bool {
	return list[i] < list[j]
}

func (list ListFloat) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

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

func (s *Set) GetSortList() ListFloat {
	s.Lock()
	defer s.Unlock()
	list := ListFloat{}
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

func NewFeeFilterRounder(minIncrementalFee FeeRate) *FeeFilterRounder {
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

	return &feeFilterRounder
}

func (feeFilterRounder *FeeFilterRounder) Round(currentMinFee int64) int64 {
	list := feeFilterRounder.feeSet.GetSortList()

	var index int
	for i, v := range list {
		if v >= float64(currentMinFee) {
			index = i
		}
	}
	if index != 0 && feeFilterRounder.insecureRand.Rand32()%3 != 0 {
		index--
	}
	if index == len(list) && list[index] < float64(currentMinFee) {
		index--
	}
	return int64(list[index])
}
