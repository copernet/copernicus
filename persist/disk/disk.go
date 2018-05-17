package disk

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/conf"

	"os"
	blogs "github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/log"

	"fmt"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/pow"
	"github.com/btcboost/copernicus/model/blockindex"
	"bytes"
)

func OpenBlockFile(pos *block.DiskBlockPos, fReadOnly bool) *os.File {
	return OpenDiskFile(*pos, "blk", fReadOnly)
}

func OpenUndoFile(pos block.DiskBlockPos, fReadOnly bool) *os.File {
	return OpenDiskFile(pos, "rev", fReadOnly)
}

func OpenDiskFile(pos block.DiskBlockPos, prefix string, fReadOnly bool) *os.File {
	if pos.IsNull() {
		return nil
	}
	path := GetBlockPosParentFilename()
	os.MkdirAll(path, os.ModePerm)

	file, err := os.Open(path + "rb+")
	if file == nil && !fReadOnly || err != nil {
		file, err = os.Open(path + "wb+")
		if err == nil {
			panic("open wb+ file failed ")
		}
	}
	if file == nil {
		blogs.Info("Unable to open file %s\n", path)
		return nil
	}
	if pos.Pos > 0 {
		if _, err := file.Seek(0, 1); err != nil {
			blogs.Info("Unable to seek to position %u of %s\n", pos.Pos, path)
			file.Close()
			return nil
		}
	}

	return file
}

func GetBlockPosFilename(pos block.DiskBlockPos, prefix string) string {
	return conf.GetDataPath() + "/blocks/" + fmt.Sprintf("%s%05d.dat", prefix, pos.File)
}

func GetBlockPosParentFilename() string {
	return conf.GetDataPath() + "/blocks/"
}


func ReadBlockFromDiskByPos(pos block.DiskBlockPos, param *consensus.BitcoinParams) (*block.Block,bool) {
	block.SetNull()

	// Open history file to read
	file := OpenBlockFile(&pos, true)
	if file == nil {
		blogs.Error("ReadBlockFromDisk: OpenBlockFile failed for %s", pos.ToString())
		return nil, false
	}

	// Read block
	blk := block.NewBlock()
	if err := blk.Unserialize(file); err != nil {
		blogs.Error("%s: Deserialize or I/O error - %s at %s", log.TraceLog(), err.Error(), pos.ToString())
	}

	// Check the header
	pow := pow.Pow{}
	if !pow.CheckProofOfWork(blk.GetHash(), blk.Header.Bits, param) {
		blogs.Error(fmt.Sprintf("ReadBlockFromDisk: Errors in block header at %s", pos.ToString()))
		return nil, false
	}
	return blk, true
}


func ReadBlockFromDisk(pindex *blockindex.BlockIndex, param *consensus.BitcoinParams) (*block.Block, bool) {
	blk, ret := ReadBlockFromDiskByPos(pindex.GetBlockPos(), param)
	if !ret{
		return nil, false
	}
	hash := pindex.GetBlockHash()
	pos := pindex.GetBlockPos()
	if bytes.Equal(blk.GetHash()[:], hash[:]) {
		blogs.Error(fmt.Sprintf("ReadBlockFromDisk(CBlock&, CBlockIndex*): GetHash()"+
			"doesn't match index for %s at %s", pindex.ToString(), pos.ToString()))
		return blk, false
	}
	return blk, true
}