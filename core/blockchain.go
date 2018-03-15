package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/btcboost/copernicus/utils"
)

type BlockChain struct {
	Path        string
	Magic       [4]byte
	CurrentFile *os.File
	CurrentID   uint32 // todo what
	LastBlock   *Block
	Lock        sync.Mutex
}

func ParseBlockchain(path string, magic [4]byte) (bc *BlockChain, err error) {
	bc.Path = path
	bc.Magic = magic
	bc.CurrentID = 0
	f, err := os.Open(blkFileName(path, 0))
	if err != nil {
		return
	}
	bc.CurrentFile = f
	return
}
func (bc *BlockChain) NextBlock() (*Block, error) {
	rawBlock, err := bc.FetchNextBlock()
	if err != nil {
		newBlkFile, err2 := os.Open(blkFileName(bc.Path, bc.CurrentID+1))
		if err2 != nil {
			return nil, err2
		}
		bc.CurrentID++
		bc.CurrentFile.Close()
		bc.CurrentFile = newBlkFile
		rawBlock, err = bc.FetchNextBlock()
	}
	block, err := ParseBlock(rawBlock)

	return block, err
}

func (bc *BlockChain) SkipBlock() (err error) {
	_, err = bc.FetchNextBlock()
	if err != nil {
		newBlkFile, err2 := os.Open(blkFileName(bc.Path, bc.CurrentID+1))
		if err2 != nil {
			return err2
		}
		bc.CurrentID++
		bc.CurrentFile.Close()
		bc.CurrentFile = newBlkFile
		_, err = bc.FetchNextBlock()
	}
	return
}

func (bc *BlockChain) FetchNextBlock() (raw []byte, err error) {
	buf := [4]byte{}
	_, err = bc.CurrentFile.Read(buf[:])
	if err != nil {
		return
	}
	if !bytes.Equal(buf[:], bc.Magic[:]) {
		err = errors.New("Bad magic")
		return
	}
	_, err = bc.CurrentFile.Read(buf[:])
	if err != nil {
		return

	}
	blockSize := uint32(blkSize(buf[:]))
	raw = make([]byte, blockSize)
	_, err = bc.CurrentFile.Read(raw[:])

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

func (blockChain *BlockChain) FetchBlockByHash(hash *utils.Hash) *Block {
	return nil
}
