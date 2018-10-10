package outpoint

import (
	"bytes"
	"os"
	"testing"

	"fmt"
	"github.com/copernet/copernicus/util"
)

var testOutPoint *OutPoint
var nullOutPoint *OutPoint
var preHash util.Hash

func initTestOutPoint() {
	preHash = util.Hash{
		0xc1, 0x60, 0x7e, 0x00, 0x31, 0xbc, 0xb1, 0x57,
		0xa3, 0xb2, 0xfd, 0x73, 0x0e, 0xcf, 0xac, 0xd1,
		0x6e, 0xda, 0x9d, 0x95, 0x7c, 0x5e, 0x03, 0xfa,
		0x34, 0x4e, 0x50, 0x21, 0xbb, 0x07, 0xcc, 0xbe,
	}

	testOutPoint = NewOutPoint(preHash, 1)

	nullOutPoint = NewDefaultOutPoint()
}

func TestNewOutPoint(t *testing.T) {
	initTestOutPoint()

	if testOutPoint.Index != 1 {
		t.Errorf("NewOutPoint() assignment index data %d should be equal 1 ", testOutPoint.Index)
	}
	if !bytes.Equal(testOutPoint.Hash[:], preHash[:]) {
		t.Errorf("NewOutPoint() assignment hash data %v "+
			"should be equal origin hash data %v", testOutPoint.Hash, preHash)
	}
	if !nullOutPoint.IsNull() {
		t.Errorf("OutPoint IsNull test failed")
	}
	var nullOutPoint1 *OutPoint
	if !nullOutPoint1.IsNull() {
		t.Errorf("OutPoint IsNull test failed")
	}
}

func TestOutPointEncodeAndDecode(t *testing.T) {
	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	initTestOutPoint()
	err = testOutPoint.Encode(file)
	if err != nil {
		t.Error(err)
	}

	file.Seek(0, 0)
	txOutRead := &OutPoint{}
	txOutRead.Hash = util.Hash{}

	err = txOutRead.Decode(file)
	if err != nil {
		t.Error(err)
	}

	if txOutRead.Index != testOutPoint.Index {
		t.Errorf("Unserialize() return the index data %d "+
			"should be equal origin index data %d", txOutRead.Index, testOutPoint.Index)
	}

	if !bytes.Equal(txOutRead.Hash[:], testOutPoint.Hash[:]) {
		t.Errorf("Unserialize() return the hash data %v"+
			"should be equal origin hash data %v", txOutRead.Hash, testOutPoint.Hash)
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}

	encodeSize := testOutPoint.EncodeSize()
	if encodeSize != 36 {
		t.Errorf("EncodeSize: %d err which should be 36", encodeSize)
	}
}

func TestOutPointSerializeAndUnserialize(t *testing.T) {
	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	initTestOutPoint()
	err = testOutPoint.Serialize(file)
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
		t.Errorf("Unserialize() return the index data %d "+
			"should be equal origin index data %d", txOutRead.Index, testOutPoint.Index)
	}

	if !bytes.Equal(txOutRead.Hash[:], testOutPoint.Hash[:]) {
		t.Errorf("Unserialize() return the hash data %v"+
			"should be equal origin hash data %v", txOutRead.Hash, testOutPoint.Hash)
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}

	serializeSize := testOutPoint.SerializeSize()
	mustSerializeSize := 32 + uint32(len(util.EncodeVarLenInt(uint64(testOutPoint.Index))))
	if serializeSize != mustSerializeSize {
		t.Errorf("SerializeSize: %d err which should be: %d", serializeSize, mustSerializeSize)
	}
}

func TestOutPoint_String(t *testing.T) {
	initTestOutPoint()
	strOutPoint := testOutPoint.String()
	strHash := preHash.String()
	strMust := fmt.Sprintf("OutPoint (hash:%s index: %d)", strHash, testOutPoint.Index)
	if strOutPoint != strMust {
		t.Errorf("OutPoint String should be %s", strMust)
	}
}
