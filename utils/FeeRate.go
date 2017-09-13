package utils

import (
	"fmt"
	"math"
)

type FeeRate struct {
	SataoshiaPerK int64
}

func (feeRate *FeeRate) GetFee(bytes int) int64 {
	if bytes > math.MaxInt64 {
		panic("bytes is  greater than MaxInt64")
	}
	size := int64(bytes)
	fee := feeRate.SataoshiaPerK * size / 1000
	if fee == 0 && size != 0 {
		if feeRate.SataoshiaPerK > 0 {
			fee = 1
		}
		if feeRate.SataoshiaPerK < 0 {
			fee = -1
		}
	}
	return fee
}

func (feeRate *FeeRate) String() string {
	return fmt.Sprintf("%d.%08d %s/kb",
		feeRate.SataoshiaPerK/COIN,
		feeRate.SataoshiaPerK%COIN,
		CURRENCY_UNIT)
}
func NewFeeRate(amount int64) *FeeRate {
	feeRate := FeeRate{SataoshiaPerK: amount}
	return &feeRate

}

func NewFeeRateWithSize(feePaid int64, bytes int) *FeeRate {
	if bytes > math.MaxInt64 {
		panic("bytes is  greater than MaxInt64")
	}
	size := int64(bytes)
	if size > 0 {
		return NewFeeRate(feePaid * 1000 / size)
	}
	return NewFeeRate(0)
}
