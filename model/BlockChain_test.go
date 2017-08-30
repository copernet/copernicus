package model

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestParseBlockchain(t *testing.T) {
	path := os.Getenv("GOPATH")
	path += "/src/github.com/btcboost/copernicus/model"
	err := creatFile(path, uint32(0))
	if err != nil {
		t.Error(err)
	}

	var magic = [4]byte{1, 2, 3, 4}
	testBlcokChain, err := ParseBlockchain(path, magic)
	if err != nil {
		t.Error(err)
	}
	defer testBlcokChain.CurrentFile.Close()
	if testBlcokChain.CurrentFile == nil {
		t.Error("The file Not Open")
	}
	if !bytes.Equal(testBlcokChain.Magic[:], magic[:]) {
		t.Errorf("ParseBlockchain() assignment magic data %v"+
			"should be equal the origin magic data %v", testBlcokChain.Magic, magic)
	}
}

func creatFile(path string, id uint32) error {

	file, err := os.Create(blkFileName(path, id))
	defer file.Close()
	return err
}

func CreatNextFile(block *BlockChain) {
	creatFile(block.Path, block.CurrentID+1)
}

func WriteContentInFile(blockChain *BlockChain) error {

	blockTmp, err := ParseBlock(rawByte[:])
	if err != nil {
		return err
	}
	blockChain.LastBlock = blockTmp

	_, err = blockChain.CurrentFile.Write(blockChain.Magic[:])
	if err != nil {
		return err
	}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, blockChain.LastBlock.Size)
	_, err = blockChain.CurrentFile.Write(buf)
	if err != nil {
		return err
	}

	_, err = blockChain.CurrentFile.Write(blockChain.LastBlock.Raw)

	return err
}

func creatBlockChiain() (*BlockChain, error) {
	path := os.Getenv("GOPATH")
	path += "/src/github.com/btcboost/copernicus/model"
	var magic = [4]byte{1, 2, 3, 4}
	testBlcokChain, err := ParseBlockchain(path, magic)
	if err != nil {
		return nil, err
	}
	return testBlcokChain, nil
}

func TestBlockChainFetchNextBlock(t *testing.T) {

	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}
	testBlcokChain.CurrentFile.Close()

	testBlcokChain.CurrentFile, err = os.OpenFile(blkFileName(testBlcokChain.Path, testBlcokChain.CurrentID), os.O_RDWR, 0666)
	if err != nil {
		t.Error(err)
		return
	}
	defer testBlcokChain.CurrentFile.Close()

	err = WriteContentInFile(testBlcokChain)
	if err != nil {
		return
	}

	_, err = testBlcokChain.CurrentFile.Seek(0, 0)
	if err != nil {
		t.Error(err)
		return
	}

	raw, err := testBlcokChain.FetchNextBlock()
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(raw, rawByte[:]) {
		t.Errorf(" FetchNextBlock() return raw data %v "+
			"should be equal origin raw data : %v", raw, rawByte)
	}
}

func TestBlockChainSkipTo(t *testing.T) {
	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}
	testBlcokChain.CurrentFile.Close()
	CreatNextFile(testBlcokChain)

	err = testBlcokChain.SkipTo(1, 0)
	if err != nil {
		t.Error(err)
	}
}

func WriteNextFile() error {

	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		return err
	}
	testBlcokChain.CurrentFile.Close()

	testBlcokChain.CurrentFile, err = os.OpenFile(blkFileName(testBlcokChain.Path, testBlcokChain.CurrentID+1), os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer testBlcokChain.CurrentFile.Close()

	err = WriteContentInFile(testBlcokChain)

	return err
}

func TestBlockChainNextBlock(t *testing.T) {
	err := WriteNextFile()
	if err != nil {
		t.Error(err)
	}

	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		return
	}
	defer testBlcokChain.CurrentFile.Close()

	block, err := testBlcokChain.NextBlock()
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(block.Raw[:], rawByte[:]) {
		t.Errorf("NextBlock return the raw data %v"+
			"should be equal origin raw data %v", block.Raw, rawByte)
	}

}

func TestBlockChainBestBlockHash(t *testing.T) {

	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}

	blockTmp, err := ParseBlock(rawByte[:])
	if err != nil {
		t.Error(err)
		return
	}
	testBlcokChain.LastBlock = blockTmp

	hash, height, err := testBlcokChain.BestBlockHash()
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(hash[:], testBlcokChain.LastBlock.Hash[:]) {
		t.Errorf("BestBlockHash() return the hash %v "+
			"should be eqaul origin hash data %v", hash, testBlcokChain.LastBlock.Hash)
	}
	if testBlcokChain.LastBlock.Height != height {
		t.Errorf("BestBlockHash() return the height %d"+
			"should be equal origin height %d", height, testBlcokChain.LastBlock.Height)
	}

}

func TestBlockChainSkipBlock(t *testing.T) {
	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}

	err = testBlcokChain.SkipBlock()
	if err != nil {
		t.Error(err)
		return
	}

	path := os.Getenv("GOPATH")
	path += "/src/github.com/btcboost/copernicus/model"
	err = os.Remove(blkFileName(path, testBlcokChain.CurrentID))
	if err != nil {
		t.Error(err)
	}
	err = os.Remove(blkFileName(path, testBlcokChain.CurrentID+1))
	if err != nil {
		t.Error(err)
	}

}
