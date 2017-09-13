package utils

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type FeeRate struct {
	SataoshisPerK int64
}

func (feeRate *FeeRate) GetFee(bytes int) int64 {
	if bytes > math.MaxInt64 {
		panic("bytes is  greater than MaxInt64")
	}
	size := int64(bytes)
	fee := feeRate.SataoshisPerK * size / 1000
	if fee == 0 && size != 0 {
		if feeRate.SataoshisPerK > 0 {
			fee = 1
		}
		if feeRate.SataoshisPerK < 0 {
			fee = -1
		}
	}
	return fee
}

func (feeRate *FeeRate) String() string {
	return fmt.Sprintf("%d.%08d %s/kb",
		feeRate.SataoshisPerK/COIN,
		feeRate.SataoshisPerK%COIN,
		CURRENCY_UNIT)
}
func NewFeeRate(amount int64) *FeeRate {
	feeRate := FeeRate{SataoshisPerK: amount}
	return &feeRate

}

func (feeRate *FeeRate) SerializeSize() int {
	return 8
}

func (feeRate *FeeRate) Serialize(writer io.Writer) error {
	return binary.Write(writer, binary.LittleEndian, feeRate.SataoshisPerK)

}

func Deserialize(reader io.Reader) (*FeeRate, error) {
	feeRate := new(FeeRate)
	var sataoshiaPerK int64
	err := binary.Read(reader, binary.LittleEndian, &sataoshiaPerK)
	if err != nil {
		return feeRate, err
	}
	feeRate.SataoshisPerK = sataoshiaPerK
	return feeRate, nil

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
