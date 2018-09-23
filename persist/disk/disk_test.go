package disk

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
)

func TestMain(m *testing.M) {
	conf.Cfg = conf.InitConfig([]string{})
	persist.InitPersistGlobal()
	os.Exit(m.Run())
}

func TestWRBlockToDisk(t *testing.T) {
	//init block header
	blkHeader := block.NewBlockHeader()
	//bitcoin-cli getblock 000000003c6cbebb51b3733fe2804b5a348f9a6d56f98aaee237022e14f0d3bc
	//{
	//  "hash": "000000003c6cbebb51b3733fe2804b5a348f9a6d56f98aaee237022e14f0d3bc",
	//  "confirmations": 11,
	//  "size": 2953,
	//  "height": 1252932,
	//  "version": 536870912,
	//  "versionHex": "20000000",
	//  "merkleroot": "7e814211a7de289a490380c0c20353e0fd4e62bf55a05b38e1628e0ea0b4fd3d",
	//  "tx": [
	//    "102f1fd1a6ba89b107fa8a6dece6c665f7fc01115677947188765c16156f2065",
	//    "27a73674b250fe205d04f94974f23cf5e976ae54caeac6e67b6118bbca997177",
	//    "1098322d430c0aadd7754489ec67499d815589fd78ca85df343b262e1bc064dc",
	//    "f99adae100f6434503d8612220f0ff6b526b47fa614e833e377c5bf40eee6e43",
	//    "381b8bb48a4846ec599f5712bfd4996186e97659b3d1e114e58a2d88c516db1a",
	//    "ca92679d9fbfba0c17507e163b82aa6ce203ab5f5ac373c83b3ffa1f5b52c1ff",
	//    "96d140b5200ffd7346e8d7b48497b12ba6fe3f0eff1cc998835451a3850779ab",
	//    "ea83b578d9e0db0990ab5447973be4ba2819c0222b53b02ca968ac5b2facaf48",
	//    "14d761ffbbef6de081d8c403a9ead2b3f7d4e205eeb58162198a925b3cd38765",
	//    "6ce23ea2eac652d71a5991db53f423d0549a90fd0f025fe274bfa12cd77129b8",
	//    "8d89cbb203afac7957970c2d7cf9729264ce6a662de98f1d6f5117e98f21847a",
	//    "b345e9b6b0365ce29bb9a6f9f7a4ef9b6d249b13e25820b06d1da9c818ebbd86",
	//    "a58e29d51cbc0f317a760bf84af6cea025d274329549e6ac6740e621a6629ffc"
	//  ],
	//  "time": 1534822771,
	//  "mediantime": 1534818759,
	//  "nonce": 1391785674,
	//  "bits": "1d00ffff",
	//  "difficulty": 1,
	//  "chainwork": "000000000000000000000000000000000000000000000031fcf736422920c3a1",
	//  "previousblockhash": "00000000000001bcd6b635a1249dfbe76c0d001592a7219a36cd9bbd002c7238",
	//  "nextblockhash": "0000000000000335133799d40608458fed06711df06166a4f628159682840113"
	//}

	blkHeader.Time = uint32(1534822771)
	blkHeader.Version = 536870912
	blkHeader.Bits = 486604799
	preHash := util.HashFromString("00000000000001bcd6b635a1249dfbe76c0d001592a7219a36cd9bbd002c7238")
	//hash := util.HashFromString("0000000000000000002a07f0b3b2d62a876d85e25fb98915af76b44a02408cd4")
	merkleRoot := util.HashFromString("7e814211a7de289a490380c0c20353e0fd4e62bf55a05b38e1628e0ea0b4fd3d")
	blkHeader.HashPrevBlock = *preHash
	//blkHeader.Hash = *hash
	blkHeader.Nonce = 1391785674
	blkHeader.MerkleRoot = *merkleRoot

	//init block
	blk := block.NewBlock()
	blk.Header = *blkHeader
	blk.GetHash()
	pos := block.NewDiskBlockPos(11, 9)
	ret := WriteBlockToDisk(blk, pos)
	if !ret {
		t.Error("write block to disk failed, please check.")
	}

	blkIndex := blockindex.NewBlockIndex(blkHeader)
	blkIndex.File = 11
	blkIndex.DataPos = 9
	blkIndex.Status = 8
	blks, ok := ReadBlockFromDisk(blkIndex, &model.TestNetParams)

	if !ok {
		t.Error("check proof work failed.")
	}

	if !reflect.DeepEqual(blks.Header, blk.Header) {
		t.Errorf("the blks should equal blk\nblks:%v\nblk :%v", blks, blk)
	}
}

func TestUndoWRToDisk(t *testing.T) {
	//block undo value is nil
	blkUndo := undo.NewBlockUndo(1)
	pos := block.NewDiskBlockPos(11, 12)
	hash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d012")
	err := UndoWriteToDisk(blkUndo, pos, *hash, wire.MainNet)
	if err != nil {
		t.Errorf("write to disk failed: %v\n", err)
	}

	bundo, ok := UndoReadFromDisk(pos, *hash)
	if !ok && reflect.DeepEqual(bundo, blkUndo) {
		t.Errorf("the wantVal not equal except value: %v, %v\n", blkUndo, bundo)
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
		t.Errorf("write to disk failed: %v\n", err1)
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
	ds := DiskUsage(conf.Cfg.DataDir)
	ok := CheckDiskSpace(math.MaxUint32)
	if !ok {
		t.Error("the disk space not enough use.")
	}
	if ds.Free < math.MaxUint32 {
		t.Error("check disk space failed, please check.")
	}
}

func TestFindBlockPos(t *testing.T) {
	pos := block.NewDiskBlockPos(10, 9)
	timeNow := time.Now().Unix()

	//fKnown:Whether to know the location of the file; if it is false, then the second is an empty
	//object of CDiskBlockPos; otherwise it is an object with data
	ok := FindBlockPos(pos, 12345, 100000, uint64(timeNow), false)
	if !ok {
		t.Error("when fKnown value is false, find block by pos failed.")
	}

	pos1 := block.NewDiskBlockPos(100, 100)
	ret := FindBlockPos(pos1, 12345, 100000, uint64(timeNow), false)
	if !ret {
		t.Error("when fKnown value is false, find block by pos failed.")
	}

	pos2 := block.NewDiskBlockPos(math.MaxInt32, math.MaxInt32)
	ok1 := FindBlockPos(pos2, 12345, 100000, uint64(timeNow), false)
	if !ok1 {
		t.Error("when fKnown value is true, find block by pos failed.")
	}

	pos3 := block.NewDiskBlockPos(1, 0)
	ok2 := FindBlockPos(pos3, 12345, 100000, uint64(timeNow), true)
	if !ok2 {
		t.Error("when fKnown value is true, find block by pos failed.")
	}

	pos4 := block.NewDiskBlockPos(8, 9)
	gPersist := persist.GetInstance()
	i := len(gPersist.GlobalBlockFileInfo)
	for i <= int(pos4.File) {
		i++
		gPersist.GlobalBlockFileInfo = append(gPersist.GlobalBlockFileInfo, block.NewBlockFileInfo())
	}
	ok3 := FindBlockPos(pos4, 12345, 100000, uint64(timeNow), true)
	if !ok3 {
		t.Error("when fKnown value is true, find block by pos failed.")
	}
}

func TestFindUndoPos(t *testing.T) {
	pos := block.NewDiskBlockPos(11, 12)
	gPersist := persist.GetInstance()
	i := len(gPersist.GlobalBlockFileInfo)
	for i <= int(pos.File) {
		i++
		gPersist.GlobalBlockFileInfo = append(gPersist.GlobalBlockFileInfo, block.NewBlockFileInfo())
	}
	err := FindUndoPos(11, pos, 12345)
	if err != nil {
		t.Errorf("find undo by pos failed: %v", err)
	}
}

func initBlockDB() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	bc := &blkdb.BlockTreeDBConfig{
		Do: &db.DBOption{
			FilePath:  filepath.Join(dir, "/copernicus/blocks/index"),
			CacheSize: 1 << 20,
		},
	}

	blkdb.InitBlockTreeDB(bc)
}

func initUtxoDB() {
	path, err := ioutil.TempDir("", "coindbtest")
	if err != nil {
		panic(fmt.Sprintf("generate temp db path failed: %s\n", err))
	}

	dbo := db.DBOption{
		FilePath:       path,
		CacheSize:      1 << 20,
		Wipe:           false,
		ForceCompactdb: false,
	}

	uc := &utxo.UtxoConfig{
		&dbo,
	}
	utxo.InitUtxoLruTip(uc)
}

func TestFlushStateToDisk(t *testing.T) {
	initBlockDB()
	initUtxoDB()
	chain.InitGlobalChain()

	necm := utxo.NewEmptyCoinsMap()
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	hash2 := util.HashFromString("000000003c6cbebb51b3733fe2804b5a348f9a6d56f98aaee237022e14f0d3bc")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}
	script1 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout1 := txout.NewTxOut(3, script1)
	coin1 := utxo.NewCoin(txout1, 1, false)

	necm.AddCoin(&outpoint1, coin1, false)
	guci := utxo.GetUtxoCacheInstance()
	err := guci.UpdateCoins(necm, hash1)
	if err != nil {
		t.Errorf("update coin failed: %v\n", err)
	}
	h, err := guci.GetBestBlock()
	if err != nil {
		t.Errorf("get best block failed: %v\n", err)
	}
	log.Info("the best block hash value is: %v\n", h)

	gPersist := persist.GetInstance()
	gdfi := gPersist.GlobalDirtyFileInfo
	gdfi = make(map[int32]bool)
	gdfi[0] = true
	gdfi[1] = false
	gdfi[2] = false

	blkHeader := block.NewBlockHeader()
	idx := blockindex.NewBlockIndex(blkHeader)

	blkHeader1 := block.NewBlockHeader()
	blkHeader.Time = uint32(1534822771)
	blkHeader.Version = 536870912
	blkHeader.Bits = 486604799
	preHash := util.HashFromString("00000000000001bcd6b635a1249dfbe76c0d001592a7219a36cd9bbd002c7238")
	merkleRoot := util.HashFromString("7e814211a7de289a490380c0c20353e0fd4e62bf55a05b38e1628e0ea0b4fd3d")
	blkHeader.HashPrevBlock = *preHash
	blkHeader.Nonce = 1391785674
	blkHeader.MerkleRoot = *merkleRoot
	//init block index
	blkidx := blockindex.NewBlockIndex(blkHeader1)

	gdbi := gPersist.GlobalDirtyBlockIndex
	gdbi = make(map[util.Hash]*blockindex.BlockIndex)
	gdbi[*hash1] = idx
	gdbi[*hash2] = blkidx

	gPersist.GlobalLastBlockFile = 1

	for _, mode := range []FlushStateMode{FlushStateNone, FlushStateIfNeeded, FlushStatePeriodic, FlushStateAlways} {
		err := FlushStateToDisk(mode, 100)
		if err != nil {
			t.Errorf("flush state mode to disk failed:%v", err)
		}
	}
}

func checkFileSize(f *os.File, size int64) bool {
	fs, err := f.Stat()
	if err != nil {
		return false
	}

	return fs.Size() == size
}

func allocateFileRangeWithNewFile(t *testing.T, size int64) {
	f, err := ioutil.TempFile("", "AllocateFileRange.*.txt")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		defer os.Remove(f.Name())
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()

	AllocateFileRange(f, 0, uint32(size))
	if !checkFileSize(f, size) {
		t.Errorf("Allocate file from %d to %d failed", 0, size)
	}

	AllocateFileRange(f, uint32(size), uint32(size))
	if !checkFileSize(f, 2*size) {
		t.Errorf("Allocate file from %d to %d failed", size, 2*size)
	}

	AllocateFileRange(f, uint32(2*size)-1, uint32(size))
	if !checkFileSize(f, 2*size-1+size) {
		t.Errorf("Allocate file from %d to %d failed", 2*size-1, 2*size-1+size)
	}
}

func TestAllocateFileRange(t *testing.T) {
	sizes := []int64{1, 99, 34, 65536, 88888}
	for _, size := range sizes {
		allocateFileRangeWithNewFile(t, size)
	}
}
