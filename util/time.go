package util

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
)

const similarTimeSecs = 5 * 60

var maxMedianTimeRetries = 200

var mockTime int64

var globalMedianTimeSource *MedianTime

// int64Sorter implements sort.Interface to allow a slice of 64-bit integers to
// be sorted.
type int64Sorter []int64

// Len returns the number of 64-bit integers in the slice.  It is part of the
// sort.Interface implementation.
func (s int64Sorter) Len() int {
	return len(s)
}

// Swap swaps the 64-bit integers at the passed indices.  It is part of the
// sort.Interface implementation.
func (s int64Sorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether the 64-bit integer with index i should sort before the
// 64-bit integer with index j.  It is part of the sort.Interface
// implementation.
func (s int64Sorter) Less(i, j int) bool {
	return s[i] < s[j]
}

type MedianTime struct {
	mtx                sync.Mutex
	knowIDs            map[string]struct{}
	offsets            []int64
	offsetSec          int64
	invalidTimeChecked bool
}

func (medianTime *MedianTime) AddTimeSample(sourceID string, timeVal time.Time) {
	medianTime.mtx.Lock()
	defer medianTime.mtx.Unlock()
	if _, exists := medianTime.knowIDs[sourceID]; exists {
		return
	}
	medianTime.knowIDs[sourceID] = struct{}{}

	now := time.Unix(GetTimeSec(), 0)
	offsetSec := int64(timeVal.Sub(now).Seconds())
	numOffsets := len(medianTime.offsets)
	if numOffsets == maxMedianTimeRetries && maxMedianTimeRetries > 0 {
		medianTime.offsets = medianTime.offsets[1:]
		numOffsets--
	}
	medianTime.offsets = append(medianTime.offsets, offsetSec)
	numOffsets++
	sortedOffsets := make([]int64, numOffsets)
	copy(sortedOffsets, medianTime.offsets)
	int64Sorter := int64Sorter(sortedOffsets)
	sort.Sort(int64Sorter)
	log.Debug("added time sample of %v (total:%v)", offsetSec, numOffsets)

	// There is a known issue here (see issue #4521):
	//
	// - The structure vTimeOffsets contains up to 200 elements, after which any
	// new element added to it will not increase its size, replacing the oldest
	// element.
	//
	// - The condition to update nTimeOffset includes checking whether the
	// number of elements in vTimeOffsets is odd, which will never happen after
	// there are 200 elements.
	//
	// But in this case the 'bug' is protective against some attacks, and may
	// actually explain why we've never seen attacks which manipulate the clock
	// offset.
	//
	// So we should hold off on fixing this and clean it up as part of a timing
	// cleanup that strengthens it in a number of other ways.
	//
	if numOffsets < 5 || numOffsets&0x01 != 1 {
		return
	}

	median := sortedOffsets[numOffsets/2]
	if uint64(math.Abs(float64(median))) <= conf.Cfg.P2PNet.MaxTimeAdjustment {
		atomic.StoreInt64(&medianTime.offsetSec, median)
	} else {
		atomic.StoreInt64(&medianTime.offsetSec, 0)

		if !medianTime.invalidTimeChecked {
			medianTime.invalidTimeChecked = true
			var removeHasCloseTime bool
			for _, offset := range sortedOffsets {
				if math.Abs(float64(offset)) < similarTimeSecs {
					removeHasCloseTime = true
					break
				}
			}
			if !removeHasCloseTime {
				log.Warn("Please check your date and time are correct!")
			}
		}
	}
}

func (medianTime *MedianTime) getOffsetSec() int64 {
	return atomic.LoadInt64(&medianTime.offsetSec)
}

func newMedianTime() *MedianTime {
	return &MedianTime{
		knowIDs: make(map[string]struct{}),
		offsets: make([]int64, 0, maxMedianTimeRetries),
	}
}

func GetMedianTimeSource() *MedianTime {
	if globalMedianTimeSource == nil {
		globalMedianTimeSource = newMedianTime()
	}
	return globalMedianTimeSource
}

func SetMockTime(time int64) {
	atomic.StoreInt64(&mockTime, time)
}

func GetTimeSec() int64 {
	mockTimeSec := atomic.LoadInt64(&mockTime)
	if mockTimeSec > 0 {
		return mockTimeSec
	}
	return time.Now().Unix()
}

func GetTimeMicroSec() int64 {
	mockTimeSec := atomic.LoadInt64(&mockTime)
	if mockTimeSec > 0 {
		return mockTimeSec * 1000 * 1000
	}
	return time.Now().UnixNano() / 1000
}

func GetAdjustedTimeSec() int64 {
	return GetTimeSec() + GetTimeOffsetSec()
}

func GetTimeOffsetSec() int64 {
	return GetMedianTimeSource().getOffsetSec()
}
