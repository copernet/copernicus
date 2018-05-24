package disk

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	
	blogs "github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/block"
	
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/pow"
	
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/chain/global"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/undo"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/net/wire"
	"github.com/btcboost/copernicus/persist/blkdb"
	"github.com/btcboost/copernicus/util"
)

type FlushStateMode int

const (
	FlushStateNone FlushStateMode = iota
	FlushStateIfNeeded
	FlushStatePeriodic
	FlushStateAlways
)

var GRequestShutdown = new(atomic.Value)

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
	parentPath := GetBlockPosParentFilename()
	os.MkdirAll(parentPath, os.ModePerm)
	filePath := GetBlockPosFilename(pos, prefix)
	flag := 0
	if fReadOnly {
		flag |= os.O_RDONLY
	} else {
		flag |= os.O_APPEND | os.O_WRONLY
	}
	if _, err := os.Stat(filePath); os.IsExist(err) {
		flag |= os.O_CREATE
	}
	file, err := os.OpenFile(filePath, flag, os.ModePerm)
	if file == nil || err != nil {
		log.Error("Unable to open file %s\n", err)
		return nil
	}
	if pos.Pos > 0 {
		if _, err := file.Seek(int64(pos.Pos), 0); err != nil {
			log.Error("Unable to seek to position %u of %s\n", pos.Pos, filePath)
			file.Close()
			return nil
		}
	}

	return file
}

func GetBlockPosFilename(pos block.DiskBlockPos, prefix string) string {
	return GetBlockPosParentFilename() + fmt.Sprintf("%s%05d.dat", prefix, pos.File)
}

func GetBlockPosParentFilename() string {
	return conf.GetDataPath() + "/blocks/"
}



func FindUndoPos(state *block.ValidationState, nFile int, undoPos *block.DiskBlockPos, nAddSize int) error {
	undoPos.File = nFile
	csLastBlockFile.Lock()
	defer csLastBlockFile.Unlock()
	undoPos.Pos = int((global.GetChainGlobalInstance().GlobalBlockFileInfoMap)[nFile].UndoSize)
	global.GetChainGlobalInstance().GlobalBlockFileInfoMap[nFile].UndoSize += uint32(nAddSize)
	nNewSize := global.GetChainGlobalInstance().GlobalBlockFileInfoMap[nFile].UndoSize
	global.GetChainGlobalInstance().GlobalSetDirtyFileInfo[nFile] = true

	nOldChunks := (undoPos.Pos + UndoFileChunkSize - 1) / UndoFileChunkSize
	nNewChunks := (nNewSize + UndoFileChunkSize - 1) / UndoFileChunkSize

	if nNewChunks > uint32(nOldChunks) {

		if CheckDiskSpace(nNewChunks*UndoFileChunkSize - uint32(undoPos.Pos)) {
			file := OpenUndoFile(*undoPos, false)
			if file != nil {
				log.Info("Pre-allocating up to position 0x%x in rev%05u.dat\n",
					nNewChunks*UndoFileChunkSize, undoPos.File)
				AllocateFileRange(file, undoPos.Pos, nNewChunks*UndoFileChunkSize-uint32(undoPos.Pos))
				file.Close()
			} else {
				return errcode.ProjectError{Code: 1002, Desc: "can not find Undo file"}

			}
		} else {
			state.Error("out of disk space")
			return errcode.ProjectError{Code: 1001, Desc: "out of disk space"}
		}
	}

	return nil
}

func AllocateFileRange(file *os.File, offset int, length uint32) {
	// Fallback version
	// TODO: just write one byte per block
	var buf [65536]byte
	file.Seek(int64(offset), 0)
	for length > 0 {
		now := 65536
		if int(length) < now {
			now = int(length)
		}
		// Allowed to fail; this function is advisory anyway.
		_, err := file.Write(buf[:])
		if err != nil {
			panic("the file write failed.")
		}
		length -= uint32(now)
	}
}

func UndoWriteToDisk(bu *undo.BlockUndo, pos *block.DiskBlockPos, hashBlock util.Hash, messageStart wire.BitcoinNet) bool {
	// Open history file to append
	undoFile := OpenUndoFile(*pos, false)
	if undoFile == nil {
		log.Error("OpenUndoFile failed")
		return false
	}
	defer undoFile.Close()
	//undoFile.Write(messageStart)
	buf := bytes.NewBuffer(nil)
	bu.Serialize(buf)
	size := buf.Len() + 32
	buHasher := sha256.New()
	buHasher.Write(hashBlock[:])
	buHasher.Write(buf.Bytes())
	buHash := buHasher.Sum(nil)
	buf.Write(buHash)
	lenBuf := bytes.NewBuffer(nil)
	util.BinarySerializer.PutUint32(lenBuf, binary.LittleEndian, uint32(size))
	undoFile.Write(lenBuf.Bytes())
	undoFile.Write(buf.Bytes())
	return true

}

func UndoReadFromDisk(pos *block.DiskBlockPos, hashblock util.Hash) (*undo.BlockUndo, bool) {
	file := OpenUndoFile(*pos, true)
	if file == nil {
		log.Error(fmt.Sprintf("%s: OpenUndoFile failed", log.TraceLog()))
		return nil, false
	}
	defer file.Close()
	size, err := util.BinarySerializer.Uint32(file, binary.LittleEndian)
	if err != nil {
		log.Error("UndoReadFromDisk===", err)
		return nil, false
	}
	buf := make([]byte, size, size)
	// Read block
	num, err := file.Read(buf)
	if uint32(num) < size {
		log.Error("UndoReadFromDisk===read undo num < size")
		return nil, false
	}
	bu := undo.NewBlockUndo()
	undoData := buf[:len(buf)-32]
	checkSumData := buf[len(buf)-32:]
	buff := bytes.NewBuffer(undoData)
	err = bu.Unserialize(buff)
	if err != nil {
		log.Error("UndoReadFromDisk===", err)
		return bu, false
	}

	// Verify checksum
	buHasher := sha256.New()
	buHasher.Write(hashblock[:])
	buHasher.Write(undoData)
	buHash := buHasher.Sum(nil)
	return bu, reflect.DeepEqual(checkSumData, buHash)

}

func ReadBlockFromDiskByPos(pos block.DiskBlockPos, param *chainparams.BitcoinParams) (*block.Block, bool) {
	
	// Open history file to read
	file := OpenBlockFile(&pos, true)
	if file == nil {
		log.Error("ReadBlockFromDisk: OpenBlockFile failed for %s", pos.String())
		return nil, false
	}
	defer file.Close()
	
	size, err := util.BinarySerializer.Uint32(file, binary.LittleEndian)
	if err != nil{
		log.Error("ReadBlockFromDisk: read block file len failed for %s", pos.String())
		return nil, false
	}
	//read block data to tmp buff
	tmp := make([]byte, size, size)
	n, err := file.Read(tmp)
	if err != nil || n != int(size){
		log.Error("ReadBlockFromDisk: read block file len != size failed for %s, %s", pos.String(), err)
		return nil, false
	}
	buf := bytes.NewBuffer(tmp)
	// Read block
	blk := block.NewBlock()
	if err := blk.Unserialize(buf); err != nil {
		log.Error("%s: Deserialize or I/O error - %s at %s", log.TraceLog(), err.Error(), pos.String())
	}
	
	// Check the header
	pow := pow.Pow{}
	if !pow.CheckProofOfWork(blk.GetHash(), blk.Header.Bits, param) {
		log.Error(fmt.Sprintf("ReadBlockFromDisk: Errors in block header at %s", pos.String()))
		return nil, false
	}
	return blk, true
}

func ReadBlockFromDisk(pindex *blockindex.BlockIndex, param *chainparams.BitcoinParams) (*block.Block, bool) {
	blk, ret := ReadBlockFromDiskByPos(pindex.GetBlockPos(), param)
	if !ret {
		return nil, false
	}
	hash := pindex.GetBlockHash()
	pos := pindex.GetBlockPos()
	if bytes.Equal(blk.GetHash()[:], hash[:]) {
		blogs.Error(fmt.Sprintf("ReadBlockFromDisk(CBlock&, CBlockIndex*): GetHash()"+
			"doesn't match index for %s at %s", pindex.String(), pos.String()))
		return blk, false
	}
	return blk, true
}


func WriteBlockToDisk(block *block.Block, pos *block.DiskBlockPos) bool {
	// Open history file to append
	file := OpenBlockFile(pos, false)
	if file == nil {
		log.Error("OpenUndoFile failed")
		return false
	}
	defer file.Close()
	buf := bytes.NewBuffer(nil)
	block.Serialize(buf)
	size := buf.Len()
	lenBuf := bytes.NewBuffer(nil)
	util.BinarySerializer.PutUint32(lenBuf, binary.LittleEndian, uint32(size))
	file.Write(lenBuf.Bytes())
	file.Write(buf.Bytes())
	return true
}

func FlushStateToDisk(state *block.ValidationState, mode FlushStateMode, nManualPruneHeight int) (ret bool) {
	ret = true
	csMain.Lock()
	csLastBlockFile.Lock()

	defer csMain.Unlock()
	defer csLastBlockFile.Unlock()
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
	nMempoolSizeMax := int64(DefaultMaxMemPoolSize) * 1000000
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
	DataBaseWriteInterval := 60 * 60
	fPeriodicWrite := mode == FlushStatePeriodic && int(nNow) > global.GetChainGlobalInstance().GlobalLastWrite+DataBaseWriteInterval*1000000
	// It's been very long since we flushed the cache. Do this infrequently,
	// to optimize cache usage.
	DataBaseFlushInterval := 24 * 60 * 60
	fPeriodicFlush := mode == FlushStatePeriodic && int(nNow) > global.GetChainGlobalInstance().GlobalLastFlush+DataBaseFlushInterval*1000000
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

		tBlockFileInfoList := make([]*block.BlockFileInfo, 0, len(global.GetChainGlobalInstance().GlobalBlockFileInfoMap))
		for _, bfi := range global.GetChainGlobalInstance().GlobalBlockFileInfoMap {
			tBlockFileInfoList = append(tBlockFileInfoList, bfi)
		}
		global.GetChainGlobalInstance().GlobalBlockFileInfoMap = make(global.BlockFileInfoMap)
		tBlockIndexList := make([]*blockindex.BlockIndex, 0, len(global.GetChainGlobalInstance().GlobalBlockIndexMap))
		for _, bi := range global.GetChainGlobalInstance().GlobalBlockIndexMap {
			tBlockIndexList = append(tBlockIndexList, bi)
		}
		global.GetChainGlobalInstance().GlobalBlockIndexMap = make(global.BlockIndexMap)
		btd := blkdb.GetBlockTreeDBInstance()
		err := btd.WriteBatchSync(tBlockFileInfoList, int(global.GetChainGlobalInstance().GlobalLastBlockFile), tBlockIndexList)
		if err != nil {
			ret = AbortNode(state, "Failed to write to block index database", "")
		}
		global.GetChainGlobalInstance().GlobalLastWrite = int(nNow)
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
		// todo
		// if !CheckDiskSpace(uint32(48 * 2 * 2 * coinsTip.GetCacheSize())) {
		// 	ret = state.Error("out of disk space")
		// }
		// Flush the chainState (which may refer to block index entries).
		if !coinsTip.Flush() {
			ret = AbortNode(state, "Failed to write to coin database", "")
		}
		global.GetChainGlobalInstance().GlobalLastFlush = int(nNow)
	}
	if fDoFullFlush || ((mode == FlushStateAlways || mode == FlushStatePeriodic) &&
		int(nNow) > global.GetChainGlobalInstance().GlobalLastSetChain+DataBaseWriteInterval*1000000) {
		// Update best block in wallet (so we can detect restored wallets).
		// TODO:GetMainSignals().SetBestChain(chainActive.GetLocator())
		global.GetChainGlobalInstance().GlobalLastSetChain = int(nNow)
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

var csMain *sync.RWMutex = new(sync.RWMutex)

var csLastBlockFile *sync.RWMutex = new(sync.RWMutex)

func FlushBlockFile(fFinalize bool) {
	csLastBlockFile.Lock()
	defer csLastBlockFile.Unlock()
	posOld := block.NewDiskBlockPos(int(global.GetChainGlobalInstance().GlobalLastBlockFile), 0)

	fileOld := OpenBlockFile(posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64(global.GetChainGlobalInstance().GlobalBlockFileInfoMap[global.GetChainGlobalInstance().GlobalLastBlockFile].Size))
			fileOld.Sync()
			fileOld.Close()
		}
	}

	fileOld = OpenUndoFile(*posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64(global.GetChainGlobalInstance().GlobalBlockFileInfoMap[global.GetChainGlobalInstance().GlobalLastBlockFile].UndoSize))
			fileOld.Sync()
			fileOld.Close()
		}
	}
}

func FindBlockPos(pos *block.DiskBlockPos, nAddSize uint,
	nHeight int32, nTime uint64, fKnown bool) bool {
	csLastBlockFile.Lock()
	defer csLastBlockFile.Unlock()

	nFile := pos.File
	if !fKnown {
		nFile = int(global.GetChainGlobalInstance().GlobalLastBlockFile)
	}

	if !fKnown {
		for uint(global.GetChainGlobalInstance().GlobalBlockFileInfoMap[nFile].Size)+nAddSize >= MaxBlockFileSize {
			nFile++
		}
		pos.File = nFile
		pos.Pos = int(global.GetChainGlobalInstance().GlobalBlockFileInfoMap[nFile].Size)
	}

	if nFile != int(global.GetChainGlobalInstance().GlobalLastBlockFile) {
		if !fKnown {
			log.Info(fmt.Sprintf("Leaving block file %d: %s\n", int(global.GetChainGlobalInstance().GlobalLastBlockFile),
				global.GetChainGlobalInstance().GlobalBlockFileInfoMap[int(global.GetChainGlobalInstance().GlobalLastBlockFile)].String()))
		}
		FlushBlockFile(!fKnown)
		int(global.GetChainGlobalInstance().GlobalLastBlockFile) = nFile
	}

	return false
}
