package model

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/btcboost/copernicus/utils"
	"os"
	"sync"
)

type BlockChain struct {
	Path        string
	Magic       [4]byte
	CurrentFile *os.File
	CurrentID   uint32
	LastBlock   *Block
	Lock        sync.Mutex
}

func ParseBlockchain(path string, magic [4]byte) (blockchain *BlockChain, err error) {
	blockchain = new(BlockChain)
	blockchain.Path = path
	blockchain.Magic = magic
	blockchain.CurrentID = 0
	f, err := os.Open(blkFileName(path, 0))
	if err != nil {
		return
	}
	blockchain.CurrentFile = f
	return
}
func (blockChain *BlockChain) NextBlock() (block *Block, err error) {

	rawBlock, err := blockChain.FetchNextBlock()
	if err != nil {
		newBlkFile, err2 := os.Open(blkFileName(blockChain.Path, blockChain.CurrentID+1))
		if err2 != nil {
			return nil, err2
		}
		blockChain.CurrentID++
		blockChain.CurrentFile.Close()
		blockChain.CurrentFile = newBlkFile
		rawBlock, err = blockChain.FetchNextBlock()
	}
	block, err = ParseBlock(rawBlock)
	return

}
func (blockChain *BlockChain) SkipBlock() (err error) {

	_, err = blockChain.FetchNextBlock()
	if err != nil {
		newBlkFile, err2 := os.Open(blkFileName(blockChain.Path, blockChain.CurrentID+1))
		if err2 != nil {
			return err2
		}
		blockChain.CurrentID++
		blockChain.CurrentFile.Close()
		blockChain.CurrentFile = newBlkFile
		_, err = blockChain.FetchNextBlock()
	}
	return
}

func (blockChain *BlockChain) FetchNextBlock() (raw []byte, err error) {

	buf := [4]byte{}
	_, err = blockChain.CurrentFile.Read(buf[:])
	if err != nil {
		return
	}
	if !bytes.Equal(buf[:], blockChain.Magic[:]) {
		err = errors.New("Bas magic")
		return
	}
	_, err = blockChain.CurrentFile.Read(buf[:])
	if err != nil {
		return

	}
	blockSize := uint32(blkSize(buf[:]))
	raw = make([]byte, blockSize)
	_, err = blockChain.CurrentFile.Read(raw[:])

	return
}

func (blockChain *BlockChain) SkipTo(blkID uint32, offset int64) (err error) {

	blockChain.CurrentID = blkID
	f, err := os.Open(blkFileName(blockChain.Path, blkID))
	if err != nil {
		return
	}

	if blockChain.CurrentFile != nil {
		blockChain.CurrentFile.Close()
	}
	blockChain.CurrentFile = f
	_, err = blockChain.CurrentFile.Seek(offset, 0)
	return
}

func blkFileName(path string, id uint32) string {
	return fmt.Sprintf("%s/blk%05.dat", path, id)
}

func blkSize(buf []byte) (size uint64) {

	for i := 0; i < len(buf); i++ {
		size |= (uint64(buf[i]) << uint(i*8))
	}
	return
}
func (blockChain *BlockChain) BestBlockHash() (utils.Hash, int32, error) {
	blockChain.Lock.Lock()
	defer blockChain.Lock.Unlock()
	return blockChain.LastBlock.Hash, blockChain.LastBlock.Height, nil

}
