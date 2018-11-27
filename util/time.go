package util

import (
	"time"
)

var mockTime int64

func GetTime() int64 {
	if mockTime > 0 {
		return mockTime
	}
	return time.Now().Unix()
}

func SetMockTime(time int64) {
	mockTime = time
}

//func GetMillisTime() int64 {
//	return time.Now().Unix()
//}

func GetMicrosTime() int64 {
	return time.Now().UnixNano()
}

//func GetTimeInSeconds() int64 {
//	return int64(time.Now().Second())
//}

func GetMockTimeInMicros() int64 {
	if mockTime > 0 {
		return mockTime * 1000 * 1000
	}
	return GetMicrosTime()
}

func GetAdjustedTime() int64 {
	return GetTime() + GetTimeOffset()
}

func GetTimeOffset() int64 {
	return int64(GetTimeSource().Offset().Seconds())
}
