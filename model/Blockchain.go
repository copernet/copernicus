package model

import (
	"os"
	"fmt"
	"bytes"
	"errors"
	"sync"
	"github.com/btccom/copernicus/utils"
)

type Blockchain struct {
	Path        string
	Magic       [4]byte
	CurrentFile *os.File
	CurrentId   uint32
	LastBlock   *Block
	Lock        sync.Mutex
}

func ParseBlockchain(path string, magic [4] byte) (blockchain *Blockchain, err error) {
	blockchain = new(Blockchain)
	blockchain.Path = path
	blockchain.Magic = magic
	blockchain.CurrentId = 0
	f, err := os.Open(blkFileName(path, 0))
	if err != nil {
		return
	}
	blockchain.CurrentFile = f
	return
}
func (blockchain *Blockchain) NextBlock() (block *Block, err error) {
	
	rawBlock, err := blockchain.FetchNextBlock()
	if err != nil {
		newBlkFile, err2 := os.Open(blkFileName(blockchain.Path, blockchain.CurrentId+1))
		if err2 != nil {
			return nil, err2
		}
		blockchain.CurrentId++
		blockchain.CurrentFile.Close()
		blockchain.CurrentFile = newBlkFile
		rawBlock, err = blockchain.FetchNextBlock()
	}
	block, err = ParseBlock(rawBlock)
	return
	
}
func (blockchain *Blockchain) SkipBlock() (err error) {
	
	_, err = blockchain.FetchNextBlock()
	if err != nil {
		newBlkFile, err2 := os.Open(blkFileName(blockchain.Path, blockchain.CurrentId+1))
		if err2 != nil {
			return err2
		}
		blockchain.CurrentId++
		blockchain.CurrentFile.Close()
		blockchain.CurrentFile = newBlkFile
		_, err = blockchain.FetchNextBlock()
	}
	return
}

func (blockchain *Blockchain) FetchNextBlock() (raw []byte, err error) {
	
	buf := [4]byte{}
	_, err = blockchain.CurrentFile.Read(buf[:])
	if err != nil {
		return
	}
	if !bytes.Equal(buf[:], blockchain.Magic[:]) {
		err = errors.New("Bas magic")
		return
	}
	_, err = blockchain.CurrentFile.Read(buf[:])
	if err != nil {
		return
		
	}
	blockSize := uint32(blkSize(buf[:]))
	raw = make([]byte, blockSize)
	_, err = blockchain.CurrentFile.Read(raw[:])
	_, err = blockchain.CurrentFile.Read(raw[:])
	return
}

func (blockchain *Blockchain) SkipTo(blkId uint32, offset int64) (err error) {
	
	blockchain.CurrentId = blkId
	f, err := os.Open(blkFileName(blockchain.Path, blkId))
	if err != nil {
		return
	}
	blockchain.CurrentFile = f
	_, err = blockchain.CurrentFile.Seek(offset, 0)
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
func (blockChain *Blockchain) BestBlockHash() (utils.Hash, int32, error) {
	blockChain.Lock.Lock()
	defer blockChain.Lock.Unlock()
	return blockChain.LastBlock.Hash, blockChain.LastBlock.Height, nil
	
}
