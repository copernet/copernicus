package core

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestParseBlockchain(t *testing.T) {
	path := os.TempDir()
	err := creatFile(path, uint32(0))
	if err != nil {
		t.Error(err)
		return
	}

	var magic = [4]byte{1, 0, 0, 0}
	testBlcokChain, err := ParseBlockchain(path, magic)
	if err != nil {
		t.Error(err)
		return
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

func creatNextFile(block *BlockChain) error {
	err := creatFile(block.Path, block.CurrentID+1)
	return err
}

func WriteContentInFile(blockChain *BlockChain) error {
	blockTmp, err := ParseBlock(blockHead[:])
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
	path := os.TempDir()
	var magic = [4]byte{1, 0, 0, 0}
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

	newBlockHead, err := testBlcokChain.FetchNextBlock()
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(newBlockHead, blockHead[:]) {
		t.Errorf(" FetchNextBlock() return txRaw data %v "+
			"should be equal origin txRaw data : %v", newBlockHead, blockHead)
	}
}

func TestBlockChainSkipTo(t *testing.T) {
	testBlockChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}
	testBlockChain.CurrentFile.Close()
	err = creatNextFile(testBlockChain)
	if err != nil {
		t.Error(err)
		return
	}

	err = testBlockChain.SkipTo(1, 0)
	if err != nil {
		t.Error(err)
	}
}

func WriteNextFile() error {
	testBlockChain, err := creatBlockChiain()
	if err != nil {
		return err
	}
	testBlockChain.CurrentFile.Close()

	testBlockChain.CurrentFile, err = os.OpenFile(blkFileName(testBlockChain.Path, testBlockChain.CurrentID+1), os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer testBlockChain.CurrentFile.Close()

	err = WriteContentInFile(testBlockChain)
	return err
}

func TestBlockChainNextBlock(t *testing.T) {
	err := WriteNextFile()
	if err != nil {
		t.Error(err)
		return
	}

	testBlockChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}
	defer testBlockChain.CurrentFile.Close()

	block, err := testBlockChain.NextBlock()
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal(block.Raw[:], blockHead[:]) {
		t.Errorf("NextBlock return the txRaw data %v"+
			"should be equal origin txRaw data %v", block.Raw, blockHead)
	}
}

func TestBlockChainBestBlockHash(t *testing.T) {
	testBlcokChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}

	blockTmp, err := ParseBlock(blockHead[:])
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
	testBlockChain, err := creatBlockChiain()
	if err != nil {
		t.Error(err)
		return
	}

	err = testBlockChain.SkipBlock()
	if err != nil {
		t.Error(err)
		return
	}

	path := os.TempDir()
	err = os.Remove(blkFileName(path, testBlockChain.CurrentID))
	if err != nil {
		t.Error(err)
	}
	err = os.Remove(blkFileName(path, testBlockChain.CurrentID+1))
	if err != nil {
		t.Error(err)
	}
}
