package util

import (
	"testing"
	"time"
)

func TestGetTime(t *testing.T) {
	mockTime = 0
	nowTime := time.Now().Unix()
	actualTime := GetTime()
	if nowTime != actualTime {
		t.Errorf("nowTime:%d should equal actualTime:%d", nowTime, actualTime)
	}

	//reset value
	mockTime = 1539746375
	actualTime = GetTime()
	nowTime = time.Now().Unix()
	if nowTime == actualTime {
		t.Errorf("the nowTime:%d should not equal actualTime:%d", nowTime, actualTime)
	}
}

func TestGetMicrosTime(t *testing.T) {
	GetMicrosTime()
}

func TestGetAdjustedTime(t *testing.T) {
	GetAdjustedTime()
}

func TestGetMockTimeInMicros(t *testing.T) {
	mockTime = 100
	actualTime := GetMockTimeInMicros()
	if actualTime != mockTime*1000*1000 {
		t.Errorf("the condition the condition should is false, actualTime is:%d", actualTime)
	}
	mockTime = 0
	GetMockTimeInMicros()
}

func TestGetTimeOffset(t *testing.T) {
	for i := 0; i <= 100000; i++ {
		timeOffset = int64(i)
		times := GetTimeOffset()
		if times != int64(i) {
			t.Errorf("the condition should is false, please check.")
		}
	}
}

func TestSetMockTime(t *testing.T) {
	tmpMockTime := int64(1539746375)
	SetMockTime(tmpMockTime)
	actualMockTime := GetTime()
	if tmpMockTime != actualMockTime {
		t.Errorf("tmpMockTime:%d should equal actualMockTime:%d", tmpMockTime, actualMockTime)
	}
}
