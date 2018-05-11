package outpoint

import (
	"bytes"
	"os"
	"testing"

	"github.com/btcboost/copernicus/util"
)

var testOutPoint *OutPoint

func TestNewOutPoint(t *testing.T) {
	var preHash = util.Hash{
		0xc1, 0x60, 0x7e, 0x00, 0x31, 0xbc, 0xb1, 0x57,
		0xa3, 0xb2, 0xfd, 0x73, 0x0e, 0xcf, 0xac, 0xd1,
		0x6e, 0xda, 0x9d, 0x95, 0x7c, 0x5e, 0x03, 0xfa,
		0x34, 0x4e, 0x50, 0x21, 0xbb, 0x07, 0xcc, 0xbe,
	}

	testOutPoint = NewOutPoint(preHash, 1)
	if testOutPoint.Index != 1 {
		t.Errorf("NewOutPoint() assignment index data %d should be equal 1 ", testOutPoint.Index)
	}
	if !bytes.Equal(testOutPoint.Hash[:], preHash[:]) {
		t.Errorf("NewOutPoint() assignment hash data %v "+
			"should be equal origin hash data %v", testOutPoint.Hash, preHash)
	}

}

func TestOutPointWriteOutPoint(t *testing.T) {
	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	err = testOutPoint.WriteOutPoint(file)
	if err != nil {
		t.Error(err)
	}

	file.Seek(0, 0)
	txOutRead := &OutPoint{}
	txOutRead.Hash = util.Hash{}

	err = txOutRead.Unserialize(file)
	if err != nil {
		t.Error(err)
	}

	if txOutRead.Index != testOutPoint.Index {
		t.Errorf("Deserialize() return the index data %d "+
			"should be equal origin index data %d", txOutRead.Index, testOutPoint.Index)
	}

	if !bytes.Equal(txOutRead.Hash[:], testOutPoint.Hash[:]) {
		t.Errorf("Deserialize() return the hash data %v"+
			"should be equal origin hash data %v", txOutRead.Hash, testOutPoint.Hash)
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}

}
