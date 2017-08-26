package model

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestBlkFileName(t *testing.T) {

	t.Log("assemble FilePath : ", blkFileName("yyx", 0))
}

func TestParseBlockchain(t *testing.T) {
	path := os.Getenv("GOPATH")
	path += "/src/github.com/btcboost/copernicus/model"
	creatFile(path, uint32(0), t)

	var magic = [4]byte{1, 2, 3, 4}
	testBlcokChain, err := ParseBlockchain(path, magic)
	if err != nil {
		t.Error(err)
		t.Log(testBlcokChain)
		return
	}
	t.Log(testBlcokChain)
	defer testBlcokChain.CurrentFile.Close()

}

func creatFile(path string, id uint32, t *testing.T) {

	file, err := os.Create(blkFileName(path, id))
	if err != nil {
		t.Error(err)
		return
	}
	defer file.Close()
}

func CreatNextFile(block *BlockChain, t *testing.T) {
	creatFile(block.Path, block.CurrentID+1, t)
}

func WriteContentInFile(block *BlockChain) error {

	blockTmp, err := ParseBlock(rawByte[:])
	if err != nil {
		return err
	}
	block.LastBlock = blockTmp

	_, err = block.CurrentFile.Write(block.Magic[:])
	if err != nil {
		return err
	}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, block.LastBlock.Size)
	_, err = block.CurrentFile.Write(buf)
	if err != nil {
		return err
	}

	_, err = block.CurrentFile.Write(block.LastBlock.Raw)

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

func TestBlockChain_FetchNextBlock(t *testing.T) {

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
		t.Errorf(" FetchNextBlock() return raw Not equal origin raw data")
		return
	}
}

func TestBlockChain_SkipTo(t *testing.T) {
	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}
	testBlcokChain.CurrentFile.Close()
	CreatNextFile(testBlcokChain, t)

	err = testBlcokChain.SkipTo(1, 0)
	if err != nil {
		t.Error(err)
		return
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

func TestBlockChain_NextBlock(t *testing.T) {
	WriteNextFile()

	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		return
	}
	defer testBlcokChain.CurrentFile.Close()

	_, err = testBlcokChain.NextBlock()
	if err != nil {
		t.Error(err)
		return
	}

}

func TestBlockChain_BestBlockHash(t *testing.T) {

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
	t.Logf("hash : %v, height : %d\n", hash, height)
}

func TestBlockChain_SkipBlock(t *testing.T) {
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
		return
	}
	err = os.Remove(blkFileName(path, testBlcokChain.CurrentID+1))
	if err != nil {
		t.Error(err)
		return
	}

}
