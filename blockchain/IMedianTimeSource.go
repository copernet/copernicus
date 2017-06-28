package blockchain

import "time"

const (
	MaxAllowedOffsetSecs = 70 * 60
	SimilarTimeSecs      = 5 * 60
	MaxMedianTimeRntries = 200
)

type IMedianTimeSource interface {
	AdjustedTime() time.Time
	AddTimeSample(id string, timeVal time.Time)
	Offset() time.Duration
}
