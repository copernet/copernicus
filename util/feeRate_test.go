package util

import (
	"testing"
	"fmt"
	"bytes"
)

func TestFeeRate(t *testing.T) {
	amountValue := int64(1000)
	feeR := NewFeeRate(amountValue)
	if feeR.SataoshisPerK != amountValue {
		t.Errorf("the SataoshisPerK:%d should equal amountValue:%d", feeR.SataoshisPerK, amountValue)
	}

	str := feeR.String()
	fmt.Println(str)

	bytes := 1000
	fee := feeR.GetFee(bytes)
	if fee != 1000 {
		t.Errorf("calculation fee:%d failed.", fee)
	}

	bytes = 10
	feeR.SataoshisPerK = 10
	fee = feeR.GetFee(bytes)
	if fee != 1 {
		t.Errorf("calculation fee:%d failed.", fee)
	}

	bytes = 20
	feeR.SataoshisPerK = -10
	fee = feeR.GetFee(bytes)
	if fee != -1 {
		t.Errorf("calculation fee:%d failed.", fee)
	}

	size := feeR.SerializeSize()
	if size != 8 {
		t.Errorf("SerializeSize failed, size is:%d", size)
	}

	feeR.SataoshisPerK = 100
	value := feeR.GetFeePerK()
	if value != 100 {
		t.Errorf("GetFeePerK failed, value is:%d", value)
	}

	tmpFeeR := NewFeeRateWithSize(10, 1000)

	if ok := feeR.Less(*tmpFeeR); ok {
		t.Errorf("the feeR.SataoshisPerK:%d < tmpFeeR.SataoshisPerK:%d", feeR.SataoshisPerK, tmpFeeR.SataoshisPerK)
	}

	feeRateSize := NewFeeRateWithSize(10, 0)
	if feeRateSize.SataoshisPerK != 0 {
		t.Errorf("SataoshisPerK value:%d should equal 0", feeRateSize.SataoshisPerK)
	}
}

func TestSerialize(t *testing.T) {
	feeR := NewFeeRate(10)
	buf := bytes.NewBuffer(nil)
	err := feeR.Serialize(buf)
	if err != nil {
		t.Errorf(err.Error())
	}

	fee, err := Unserialize(buf)
	if err != nil {
		t.Errorf(err.Error())
	}
	if fee.SataoshisPerK != 10 {
		t.Errorf("the fee.SataoshisPerK:%d should equal 10", fee.SataoshisPerK)
	}
}
