package model

import (
	"testing"

	"bytes"
	"github.com/btcboost/copernicus/utils"
	"os"
)

var testOutPoint *OutPoint

func TestNewOutPoint(t *testing.T) {
	var preHash = utils.Hash{
		0xc1, 0x60, 0x7e, 0x00, 0x31, 0xbc, 0xb1, 0x57,
		0xa3, 0xb2, 0xfd, 0x73, 0x0e, 0xcf, 0xac, 0xd1,
		0x6e, 0xda, 0x9d, 0x95, 0x7c, 0x5e, 0x03, 0xfa,
		0x34, 0x4e, 0x50, 0x21, 0xbb, 0x07, 0xcc, 0xbe,
	}

	testOutPoint = NewOutPoint(&preHash, 1)
	if testOutPoint.Index != 1 {
		t.Error("The index should be 1 instead of ", testOutPoint.Index)
	}
	if !bytes.Equal(testOutPoint.Hash[:], preHash[:]) {
		t.Error("The two slice should be equal")
	}

}

func TestOutPoint_WriteOutPoint(t *testing.T) {
	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	err = testOutPoint.WriteOutPoint(file, 10, 1)
	if err != nil {
		t.Error(err)
	}

	file.Seek(0, 0)
	txOutRead := &OutPoint{}
	txOutRead.Hash = new(utils.Hash)

	err = txOutRead.ReadOutPoint(file, 1)
	if err != nil {
		t.Error(err)
	}

	if txOutRead.Index != testOutPoint.Index {
		t.Error("The two index should be equal")
	}

	if !bytes.Equal(txOutRead.Hash[:], testOutPoint.Hash[:]) {
		t.Error("The two slice should be equal")
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}

}
