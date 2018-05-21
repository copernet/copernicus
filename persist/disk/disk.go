package disk

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/conf"
	"sync"
	"os"
	blogs "github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/log"

	"fmt"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/pow"
	"github.com/btcboost/copernicus/model/blockindex"
	"bytes"
	"copernicus/policy"
	"math"
	"sync/atomic"
	"time"
	"github.com/btcboost/copernicus/model/utxo"
	"syscall"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/tx"
)
type FlushStateMode int

const (
	FlushStateNone FlushStateMode = iota
	FlushStateIfNeeded
	FlushStatePeriodic
	FlushStateAlways
)

var GRequestShutdown   = new(atomic.Value)
func StartShutdown() {
	GRequestShutdown.Store(true)
}

func AbortNodes(reason, userMessage string) bool {
	log.Info("*** %s\n", reason)

	//todo:
	if len(userMessage) == 0 {
		panic("Error: A fatal internal error occurred, see debug.log for details")
	} else {

	}
	StartShutdown()
	return false
}
func AbortNode(state *block.ValidationState, reason, userMessage string) bool {
	AbortNodes(reason, userMessage)
	return state.Error(reason)
}



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


func FlushStateToDisk(state *block.ValidationState, mode FlushStateMode, nManualPruneHeight int) (ret bool) {
	ret = true
	// TODO: LOCK2(cs_main, cs_LastBlockFile);
	// var sc sync.RWMutex
	// sc.Lock()
	// defer sc.Unlock()
	//
	//var setFilesToPrune *set.Set
	//fFlushForPrune := false

	defer func() {
		if r := recover(); r != nil {
			ret = AbortNode(state, "System error while flushing:", "")
		}
	}()
	fFlushForPrune := false
	// todo for prune
	//if GPruneMode && (GCheckForPruning || nManualPruneHeight > 0) && !GfReindex {
	//	FindFilesToPruneManual(setFilesToPrune, nManualPruneHeight)
	//} else {
	//	FindFilesToPrune(setFilesToPrune, uint64(params.PruneAfterHeight))
	//	GCheckForPruning = false
	//}
	//if !setFilesToPrune.IsEmpty() {
	//	fFlushForPrune = true
	//	if !GHavePruned {
	//		// TODO: pblocktree.WriteFlag("prunedblockfiles", true)
	//		GHavePruned = true
	//	}
	//}
	nNow := time.Now().UnixNano()
	// todo Avoid writing/flushing immediately after startup.
	//if gLastWrite == 0 {
	//	gLastWrite = int(nNow)
	//}
	//if gLastFlush == 0 {
	//	gLastFlush = int(nNow)
	//}
	//if gLastSetChain == 0 {
	//	gLastSetChain = int(nNow)
	//}
	mempoolUsage := int64(0) // todo mempool.mempoolUsage
	coinsTip := utxo.GetUtxoCacheInstance()
	nMempoolSizeMax := int64(tx.DefaultMaxMemPoolSize) * 1000000
	DBPeakUsageFactor := int64(2)
	cacheSize := coinsTip.DynamicMemoryUsage() * DBPeakUsageFactor
	nCoinCacheUsage := 5000 * 300
	nTotalSpace := float64(nCoinCacheUsage) + math.Max(float64(nMempoolSizeMax-mempoolUsage), 0)
	// The cache is large and we're within 10% and 200 MiB or 50% and 50MiB
	// of the limit, but we have time now (not in the middle of a block processing).
	MinBlockCoinsDBUsage := DBPeakUsageFactor
	x := math.Max(float64(nTotalSpace/2), float64(nTotalSpace-float64(MinBlockCoinsDBUsage*1024*1024)))
	MaxBlockCoinsDBUsage := float64(DBPeakUsageFactor * 200)
	y := math.Max(9*nTotalSpace/10, nTotalSpace-MaxBlockCoinsDBUsage*1024*1024)
	fCacheLarge := mode == FlushStatePeriodic && float64(cacheSize) > math.Min(x, y)
	// The cache is over the limit, we have to write now.
	fCacheCritical := mode == FlushStateIfNeeded && float64(cacheSize) > nTotalSpace
	// It's been a while since we wrote the block index to disk. Do this
	// frequently, so we don't need to redownLoad after a crash.
	DataBaseWriteInterval := 60*60
	fPeriodicWrite := mode == FlushStatePeriodic && int(nNow) > GlobalLastWrite+DataBaseWriteInterval*1000000
	// It's been very long since we flushed the cache. Do this infrequently,
	// to optimize cache usage.
	DataBaseFlushInterval := 24*60*60
	fPeriodicFlush := mode == FlushStatePeriodic && int(nNow) > GlobalLastFlush+DataBaseFlushInterval*1000000
	// Combine all conditions that result in a full cache flush.
	fDoFullFlush := mode == FlushStateAlways || fCacheLarge || fCacheCritical || fPeriodicFlush || fFlushForPrune
	// Write blocks and block index to disk.
	if fDoFullFlush || fPeriodicWrite {
		// Depend on nMinDiskSpace to ensure we can write block index
		if !CheckDiskSpace(0) {
			ret = state.Error("out of disk space")
		}
		// First make sure all block and undo data is flushed to disk.
		FlushBlockFile(false)
		// Then update all block file information (which may refer to block and undo files).


		tBlockFileInfoList := make([]*block.BlockFileInfo, 0, len(*GlobalBlockFileInfoMap))
		for _, bfi := range *GlobalBlockFileInfoMap {
			tBlockFileInfoList = append(tBlockFileInfoList, bfi)
		}
		GlobalBlockFileInfoMap = new(BlockFileInfoMap)
		tBlockIndexList := make([]*blockindex.BlockIndex, 0, len(*GlobalBlockIndexMap))
		for _, bi := range *GlobalBlockIndexMap {
			tBlockIndexList = append(tBlockIndexList, bi)
		}
		GlobalBlockIndexMap = new(BlockIndexMap)
		btd := chain.GetBlockTreeDBInstance()
		err := btd.WriteBatchSync(tBlockFileInfoList, GlobalLastBlockFile,  tBlockIndexList)
		if err != nil{
			ret = AbortNode(state, "Failed to write to block index database", "")
		}
		GlobalLastWrite = int(nNow)
		//// todo Finally remove any pruned files
		//if fFlushForPrune {
		//	UnlinkPrunedFiles(setFilesToPrune)
		//}
	}

	// Flush best chain related state. This can only be done if the blocks /
	// block index write was also done.
	if fDoFullFlush {
		// Typical Coin structures on disk are around 48 bytes in size.
		// Pushing a new one to the database can cause it to be written
		// twice (once in the log, and once in the tables). This is already
		// an overestimation, as most will delete an existing entry or
		// overwrite one. Still, use a conservative safety factor of 2.
		if !CheckDiskSpace(uint32(48 * 2 * 2 * coinsTip.GetCacheSize())) {
			ret = state.Error("out of disk space")
		}
		// Flush the chainState (which may refer to block index entries).
		if !coinsTip.Flush() {
			ret = AbortNode(state, "Failed to write to coin database", "")
		}
		GlobalLastFlush = int(nNow)
	}
	if fDoFullFlush || ((mode == FlushStateAlways || mode == FlushStatePeriodic) &&
		int(nNow) > GlobalLastSetChain+DataBaseWriteInterval*1000000) {
		// Update best block in wallet (so we can detect restored wallets).
		// TODO:GetMainSignals().SetBestChain(chainActive.GetLocator())
		GlobalLastSetChain = int(nNow)
	}

	return
}


func CheckDiskSpace(nAdditionalBytes uint32) bool {
	path := conf.GetDataPath()
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		log.Error("can not get disk info")
		return false
	}
	nFreeBytesAvailable := fs.Ffree * uint64(fs.Bsize)

	// Check for nMinDiskSpace bytes (currently 50MB)
	MinDiskSpace := 52428800
	if int(nFreeBytesAvailable) < MinDiskSpace+int(nAdditionalBytes) {
		return AbortNodes("Disk space is low!", "Error: Disk space is low!")
	}
	return true
}

var csLastBlockFile *sync.RWMutex = new(sync.RWMutex)
var gLastBlockFile int  = 0
func FlushBlockFile(fFinalize bool) {
	// todo !!! add file sync.lock, LOCK(cs_LastBlockFile);
	csLastBlockFile.Lock()
	defer csLastBlockFile.Unlock()
	posOld := block.NewDiskBlockPos(gLastBlockFile, 0)

	fileOld := OpenBlockFile(posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64((*GlobalBlockFileInfoMap)[GlobalLastBlockFile].Size))
			fileOld.Sync()
			fileOld.Close()
		}
	}

	fileOld = OpenUndoFile(*posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64((*GlobalBlockFileInfoMap)[GlobalLastBlockFile].UndoSize))
			fileOld.Sync()
			fileOld.Close()
		}
	}
}

func FindBlockPos(state *block.ValidationState, pos *block.DiskBlockPos, nAddSize uint,
	nHeight uint, nTime uint64, fKnown bool) bool {

	//	todo !!! Add sync.Lock in the later, because the concurrency goroutine
	nFile := pos.File
	if !fKnown {
		nFile = gLastBlockFile
	}

	if !fKnown {
		for uint(gInfoBlockFile[nFile].Size)+nAddSize >= MaxBlockFileSize {
			nFile++
		}
		pos.File = nFile
		pos.Pos = int(gInfoBlockFile[nFile].Size)
	}

	if nFile != gLastBlockFile {
		if !fKnown {
			logs.Info(fmt.Sprintf("Leaving block file %d: %s\n", gLastBlockFile,
				gInfoBlockFile[gLastBlockFile].ToString()))
		}
		FlushBlockFile(!fKnown)
		gLastBlockFile = nFile
	}
	return false
}