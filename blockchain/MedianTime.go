package blockchain

import (
	"sync"
	"time"
	"sort"
	"copernicus/algorithm"
	
	"math"
	"github.com/astaxie/beego/logs"
)

var log = logs.NewLogger()

type MedianTime struct {
	lock               sync.Mutex
	knowIDs            map[string]struct{}
	offsets            [] int64
	offsetsSecs        int64
	invalidTimeChecked bool
}

var _ IMedianTimeSource = (*MedianTime)(nil)

func (medianTime *MedianTime) AdjustedTime() time.Time {
	medianTime.lock.Lock()
	defer medianTime.lock.Unlock()
	now := time.Unix(time.Now().Unix(), 0)
	return now.Add(time.Duration(medianTime.offsetsSecs) * time.Second)
}
func (medianTime *MedianTime) AddTimeSample(sourceID string, timeVal time.Time) {
	medianTime.lock.Lock()
	defer medianTime.lock.Unlock()
	if _, exists := medianTime.knowIDs[sourceID]; exists {
		return
	}
	medianTime.knowIDs[sourceID] = struct{}{}
	
	now := time.Unix(time.Now().Unix(), 0)
	offsetSecs := int64(timeVal.Sub(now).Seconds())
	numOffsets := len(medianTime.offsets)
	if numOffsets == MAX_MEDIAN_TIME_ENTRIES && MAX_MEDIAN_TIME_ENTRIES > 0 {
		medianTime.offsets = medianTime.offsets[1:]
		numOffsets--
	}
	medianTime.offsets = append(medianTime.offsets, offsetSecs)
	numOffsets++
	sortedOffsets := make([]int64, numOffsets)
	copy(sortedOffsets, medianTime.offsets)
	int64Sorter := algorithm.Int64Sorter(sortedOffsets)
	sort.Sort(int64Sorter)
	offsetDuration := time.Duration(offsetSecs) * time.Second
	log.Debug("added time sample of %v (total:%v)", offsetDuration, numOffsets)
	
	if numOffsets < 5 || numOffsets&0x01 != 1 {
		return
	}
	median := sortedOffsets[numOffsets/2]
	if math.Abs(float64(median)) < MAX_ALLOWED_OFFSET_SECS {
		medianTime.offsetsSecs = median
	} else {
		medianTime.offsetsSecs = 0
		if !medianTime.invalidTimeChecked {
			medianTime.invalidTimeChecked = true
			var removeHasCloseTime bool
			for _, offset := range sortedOffsets {
				if math.Abs(float64(offset)) < SIMILAR_TIME_SECS {
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
func (medianTime *MedianTime) Offset() time.Duration {
	medianTime.lock.Lock()
	defer medianTime.lock.Unlock()
	return time.Duration(medianTime.offsetsSecs) * time.Second
}
func NewMedianTime() IMedianTimeSource {
	medianTime := MedianTime{
		knowIDs: make(map[string]struct{}),
		offsets: make([]int64, 0, MAX_MEDIAN_TIME_ENTRIES),
	}
	return &medianTime
}
