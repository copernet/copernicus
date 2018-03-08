package utils

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	//MIN_FEERATE Minimum and Maximum values for tracking feerates
	MIN_FEERATE  int64   = 10
	MAX_FEERATE  int64   = 1e7
	INF_FEERATE  int64   = MAX_MONEY
	INF_PRIORITY float64 = 1e9 * float64(MAX_MONEY)

	//FEE_SPACING We have to lump transactions into buckets based on feerate, but we want to be
	// able to give accurate estimates over a large range of potential feerates.
	// Therefore it makes sense to exponentially space the buckets
	FEE_SPACING float64 = 1.1
)

const (
	/*MAX_BLOCK_CONFIRMS Track confirm delays up to 25 blocks, can't estimate beyond that */
	MAX_BLOCK_CONFIRMS uint = 25

	/*DEFAULT_DECAY Decay of .998 is a half-life of 346 blocks or about 2.4 days */
	DEFAULT_DECAY float64 = .998

	/*MIN_SUCCESS_PCT Require greater than 95% of X feerate transactions to be confirmed within Y
	 * blocks for X to be big enough */
	MIN_SUCCESS_PCT float64 = .95

	/*SUFFICIENT_FEETXS Require an avg of 1 tx in the combined feerate bucket per block to have stat
	 * significance */
	SUFFICIENT_FEETXS float64 = 1
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

func (feeRate *FeeRate) GetFeePerK() int64 {
	return feeRate.GetFee(1000)
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
