package blockchain

import "time"

const (
	MAX_ALLOWED_OFFSET_SECS = 70 * 60
	SIMILAR_TIME_SECS       = 5 * 60
	MAX_MEDIAN_TIME_ENTRIES = 200
)

type IMedianTimeSource interface {
	AdjustedTime() time.Time
	AddTimeSample(id string, timeVal time.Time)
	Offset() time.Duration
}

