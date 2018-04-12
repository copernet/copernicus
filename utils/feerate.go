package utils

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	// MinFeeRate minimum and Maximum values for tracking feeRates
	MinFeeRate  int64   = 10
	MaxFeeRate  int64   = 1e7
	InfFeeRate  int64   = MaxMoney
	InfPriority float64 = 1e9 * float64(MaxMoney)

	//FeeSpacing we have to lump transactions into buckets based on feeRate, but we want to be
	// able to give accurate estimates over a large range of potential feeRates.
	// Therefore it makes sense to exponentially space the buckets
	FeeSpacing float64 = 1.1
)

const (
	// MaxBlockConfirms track confirm delays up to 25 blocks, can't estimate beyond that
	MaxBlockConfirms uint = 25

	// DefaultDecay decay of .998 is a half-life of 346 blocks or about 2.4 days
	DefaultDecay float64 = .998

	// MinSuccessPct require greater than 95% of X feeRate transactions to be confirmed within Y
	// blocks for X to be big enough
	MinSuccessPct float64 = .95

	// SufficientFeeTxs Require an avg of 1 tx in the combined feeRate bucket per block to have stat
	// significance
	SufficientFeeTxs float64 = 1
)

// FeeRate : Fee rate in satoshis per kilobyte: Amount / kB
type FeeRate struct {
	SataoshisPerK int64
}

// GetFee : Return the fee in satoshis for the given size in bytes.
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

// GetFeePerK : Return the fee in satoshis for a size of 1000 bytes
func (feeRate *FeeRate) GetFeePerK() int64 {
	return feeRate.GetFee(1000)
}

func (feeRate *FeeRate) String() string {
	return fmt.Sprintf("%d.%08d %s/kb",
		feeRate.SataoshisPerK/COIN,
		feeRate.SataoshisPerK%COIN,
		CurrencyUnit)
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

func (feeRate *FeeRate) Less(b FeeRate) bool {
	return feeRate.SataoshisPerK < b.SataoshisPerK
}

func NewFeeRate(amount int64) FeeRate {
	feeRate := FeeRate{SataoshisPerK: amount}
	return feeRate
}

func NewFeeRateWithSize(feePaid int64, bytes int64) FeeRate {
	if bytes > math.MaxInt64 {
		panic("bytes is  greater than MaxInt64")
	}
	if bytes > 0 {
		return NewFeeRate(feePaid * 1000 / bytes)
	}
	return NewFeeRate(0)
}
