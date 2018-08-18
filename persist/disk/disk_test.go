package disk

import (
	"time"
	"testing"

	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/net/wire"
	"reflect"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"math"
	"syscall"
	"github.com/copernet/copernicus/conf"
)

func TestWRBlockToDisk(t *testing.T) {
	//init block header
	blkHeader := block.NewBlockHeader()
	blkHeader.Time = uint32(time.Now().Unix())
	blkHeader.Hash = blkHeader.GetHash()
	blkHeader.Version = 0
	blkHeader.Bits = 0
	preHash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d011")
	hash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d012")
	merkleRoot := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d013")
	blkHeader.HashPrevBlock = *preHash
	blkHeader.Hash = *hash
	blkHeader.Nonce = 0
	blkHeader.MerkleRoot = *merkleRoot

	//init block
	blk := block.NewBlock()
	blk.Header = *blkHeader
	blk.Checked = false
	pos := block.NewDiskBlockPos(10, 9)
	ret := WriteBlockToDisk(blk, pos)
	if !ret {
		t.Error("write block to disk failed, please check.")
	}

	//fixme:CheckProofOfWork failed
	//blkIndex := blockindex.NewBlockIndex(blkHeader)
	//blkIndex.File = 10
	//blkIndex.DataPos = 9
	//blks, ok := ReadBlockFromDisk(blkIndex, &chainparams.TestNetParams)
	//if !reflect.DeepEqual(blks, blk) && !ok {
	//	t.Errorf("the blks should equal blk\nblks:%v\nblk:%v", blks, blk)
	//}
}

func TestUndoWRToDisk(t *testing.T) {
	//block undo value is nil
	blkUndo := undo.NewBlockUndo(1)
	pos := block.NewDiskBlockPos(11, 12)
	hash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d012")
	err := UndoWriteToDisk(blkUndo, pos, *hash, wire.MainNet)
	if err != nil {
		t.Error("write failed.")
	}

	bundo, ok := UndoReadFromDisk(pos, *hash)
	if !ok && reflect.DeepEqual(bundo, blkUndo) {
		t.Error("read undo block failed.")
	}

	//block undo add txundo info
	blkUndo1 := undo.NewBlockUndo(1)
	txundo := undo.NewTxUndo()
	//init coin
	script1 := script.NewEmptyScript()
	txout1 := txout.NewTxOut(2, script1)
	c := utxo.NewCoin(txout1, 10, false)
	txundo.SetUndoCoins([]*utxo.Coin{c})
	blkUndo1.AddTxUndo(txundo)
	pos1 := block.NewDiskBlockPos(11, 12)
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d012")
	err1 := UndoWriteToDisk(blkUndo1, pos1, *hash1, wire.MainNet)
	if err1 != nil {
		t.Error("write failed.")
	}

	bundo1, ok1 := UndoReadFromDisk(pos, *hash)
	if !ok1 && reflect.DeepEqual(bundo1, blkUndo1) {
		t.Error("read undo block failed.")
	}
}
type DiskStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
}

// disk usage of path/disk
func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.All = fs.Blocks * uint64(fs.Bsize)
	disk.Free = fs.Bfree * uint64(fs.Bsize)
	disk.Used = disk.All - disk.Free
	return
}

func TestCheckDiskSpace(t *testing.T) {
	ds:=DiskUsage(conf.Cfg.DataDir)
	ok := CheckDiskSpace(math.MaxUint32)
	if !ok {
		t.Error("the disk space not enough use.")
	}
	if ds.Free < math.MaxUint32 {
		t.Error("check disk space failed, please check.")
	}
}



