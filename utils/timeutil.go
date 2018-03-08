package utils

import (
	"sync/atomic"
	"time"
)

var mockTime int64
var timeOffset int64

func GetMockTime() int64 {
	if mockTime > 0 {
		return mockTime
	}
	return int64(time.Now().Second())
}

func SetMockTime(time int64) {
	mockTime = time
}

func GetMillisTimee() int64 {
	return time.Now().Unix()
}

func GetMicrosTime() int64 {
	return time.Now().UnixNano()
}

func GetTimeInSeconds() int64 {
	return int64(time.Now().Second())
}

func GetMockTimeInMicros() int64 {
	if mockTime > 0 {
		return mockTime * 1000 * 1000
	}
	return GetMicrosTime()
}

func GetAdjustedTime() int64 {
	return GetMockTime() + GetTimeOffset()
}

func GetTimeOffset() int64 {
	return atomic.LoadInt64(&timeOffset)
}
