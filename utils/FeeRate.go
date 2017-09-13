package utils

type FeeRate struct {
	SataoshiaPerK int64
}

func NewFeeRate(amount int64) *FeeRate {
	feeRate := FeeRate{SataoshiaPerK: amount}
	return &feeRate

}
