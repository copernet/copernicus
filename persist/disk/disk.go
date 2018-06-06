package disk

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"reflect"
	"sync/atomic"
	"syscall"
	"time"
	
	blogs "github.com/astaxie/beego/logs"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"
	
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/pow"
	

"github.com/copernet/copernicus/errcode"
"github.com/copernet/copernicus/model/chainparams"
"github.com/copernet/copernicus/model/undo"
"github.com/copernet/copernicus/model/utxo"
"github.com/copernet/copernicus/net/wire"
"github.com/copernet/copernicus/persist/blkdb"
"github.com/copernet/copernicus/persist/global"
"github.com/copernet/copernicus/util"

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
	e := os.MkdirAll(parentPath, os.ModePerm)
	if e!=nil{
		log.Error("e=========",e)
		panic("OpenDiskFile.os.MkdirAll(parentPath err")
	}
	filePath := GetBlockPosFilename(pos, prefix)
	flag := 0
	if fReadOnly {
		flag |= os.O_RDONLY
	} else {
		flag |= os.O_APPEND | os.O_WRONLY
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		flag |= os.O_CREATE
	}
	file, err := os.OpenFile(filePath, flag, os.ModePerm)
	if file == nil || err != nil {
		log.Error("Unable to open file %s\n", err)
		panic("Unable to open file ======")
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




func AllocateFileRange(file *os.File, offset uint32, length uint32) {
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

func UndoWriteToDisk(bu *undo.BlockUndo, pos *block.DiskBlockPos, hashBlock util.Hash, messageStart wire.BitcoinNet) error {
	// Open history file to append
	undoFile := OpenUndoFile(*pos, false)
	if undoFile == nil {
		log.Error("OpenUndoFile failed")
		return errcode.New(errcode.ErrorOpenUndoFileFailed)
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
	return nil

}

func UndoReadFromDisk(pos *block.DiskBlockPos, hashblock util.Hash) (*undo.BlockUndo, bool) {
	file := OpenUndoFile(*pos, true)
	if file == nil {
		log.Error(fmt.Sprintf("%s: OpenUndoFile failed", pos.String()))
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
	bu := undo.NewBlockUndo(0)
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
		log.Error("ReadBlockFromDiskByPos: Unserialize or I/O error - %s at %s", err.Error(), pos.String())
	}
	
	// Check the header
	pow := pow.Pow{}
	blockHash := blk.GetHash()
	if !pow.CheckProofOfWork(&blockHash, blk.Header.Bits, param) {
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
	blockHash := blk.GetHash()
	if bytes.Equal(blockHash[:], hash[:]) {
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
	lenData := lenBuf.Bytes()
	file.Write(lenData)
	file.Write(buf.Bytes())
	return true
}

func FlushStateToDisk( mode FlushStateMode, nManualPruneHeight int) error {
	// global.CsMain.Lock()
	global.CsLastBlockFile.Lock()

	// defer global.CsMain.Unlock()
	defer global.CsLastBlockFile.Unlock()
	
	gPersist := global.GetInstance()
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.Error("System error while flushing:", r)
	// 		// return errcode.New(errcode.SystemErrorWhileFlushing)
	//
	// 	}
	// }()
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
	nMempoolSizeMax := int64(global.DefaultMaxMemPoolSize) * 1000000
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
	fPeriodicWrite := mode == FlushStatePeriodic && int(nNow) > gPersist.GlobalLastWrite+DataBaseWriteInterval*1000000
	// It's been very long since we flushed the cache. Do this infrequently,
	// to optimize cache usage.
	DataBaseFlushInterval := 24 * 60 * 60
	fPeriodicFlush := mode == FlushStatePeriodic && int(nNow) > gPersist.GlobalLastFlush+DataBaseFlushInterval*1000000
	// Combine all conditions that result in a full cache flush.
	fDoFullFlush := mode == FlushStateAlways || fCacheLarge || fCacheCritical || fPeriodicFlush || fFlushForPrune
	// Write blocks and block index to disk.
	if fDoFullFlush || fPeriodicWrite {
		// Depend on nMinDiskSpace to ensure we can write block index
		if !CheckDiskSpace(0) {
			return errcode.New(errcode.ErrorOutOfDiskSpace)
		}
		// First make sure all block and undo data is flushed to disk.
		FlushBlockFile(false)
		// Then update all block file information (which may refer to block and undo files).

		dirtyBlockFileInfoList := make([]*block.BlockFileInfo, 0, len(gPersist.GlobalDirtyFileInfo))
		for k, _ := range gPersist.GlobalDirtyFileInfo {
			dirtyBlockFileInfoList = append(dirtyBlockFileInfoList, gPersist.GlobalBlockFileInfo[k])
			
		}
		gPersist.GlobalDirtyFileInfo = make(map[int32]bool , 0)
		dirtyBlockIndexList := make([]*blockindex.BlockIndex, 0, len(gPersist.GlobalDirtyBlockIndex))
		for _, bi := range gPersist.GlobalDirtyBlockIndex {
			dirtyBlockIndexList = append(dirtyBlockIndexList, bi)
		}
		gPersist.GlobalDirtyBlockIndex = make(global.DirtyBlockIndex)
		btd := blkdb.GetInstance()
		err := btd.WriteBatchSync(dirtyBlockFileInfoList, int(gPersist.GlobalLastBlockFile), dirtyBlockIndexList)
		if err != nil {
			return errcode.New(errcode.ErrorFailedToWriteToBlockIndexDatabase)
			
		}
		gPersist.GlobalLastWrite = int(nNow)
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
			return errcode.New(errcode.ErrorFailedToWriteToCoinDatabase)
			
		}
		gPersist.GlobalLastFlush = int(nNow)
	}
	if fDoFullFlush || ((mode == FlushStateAlways || mode == FlushStatePeriodic) &&
		int(nNow) > gPersist.GlobalLastSetChain+DataBaseWriteInterval*1000000) {
		// Update best block in wallet (so we can detect restored wallets).
		gPersist.GlobalLastSetChain = int(nNow)
	}

	return nil
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
	n := int(nAdditionalBytes)
	needSize := uint64(MinDiskSpace+n)
	if nFreeBytesAvailable < needSize {
		return AbortNodes("Disk space is low!", "Error: Disk space is low!")
	}
	return true
}


func FlushBlockFile(fFinalize bool) {
	// global.CsLastBlockFile.Lock()
	// defer global.CsLastBlockFile.Unlock()
	gPersist := global.GetInstance()
	posOld := block.NewDiskBlockPos(gPersist.GlobalLastBlockFile, 0)

	fileOld := OpenBlockFile(posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64(gPersist.GlobalBlockFileInfo[gPersist.GlobalLastBlockFile].Size))
			fileOld.Sync()
			fileOld.Close()
		}
	}

	fileOld = OpenUndoFile(*posOld, false)
	if fileOld != nil {
		if fFinalize {
			os.Truncate(fileOld.Name(), int64(gPersist.GlobalBlockFileInfo[gPersist.GlobalLastBlockFile].UndoSize))
			fileOld.Sync()
			fileOld.Close()
		}
	}
}

func FindBlockPos(pos *block.DiskBlockPos, nAddSize uint32,
	nHeight int32, nTime uint64, fKnown bool) bool {
	global.CsLastBlockFile.Lock()
	defer global.CsLastBlockFile.Unlock()
	if nAddSize > global.MaxBlockFileSize{
		log.Error("FindBlockPos nAddSize [%#v] is too large more then global.MaxBlockFileSize ", nAddSize)
		panic("FindBlockPos nAddSize  is too large more then global.MaxBlockFileSize")
	}
	gPersist := global.GetInstance()
	ret := false
	nFile := pos.File
	if !fKnown {
		nFile = gPersist.GlobalLastBlockFile
	}
	if len(gPersist.GlobalBlockFileInfo) <= int(nFile){
		gPersist.GlobalBlockFileInfo = append(gPersist.GlobalBlockFileInfo, block.NewBlockFileInfo())
	}
	if !fKnown {
		for gPersist.GlobalBlockFileInfo[nFile].Size+nAddSize >= global.MaxBlockFileSize {
			nFile++
			if int(nFile) >= len(gPersist.GlobalBlockFileInfo){
				gPersist.GlobalBlockFileInfo = append(gPersist.GlobalBlockFileInfo, block.NewBlockFileInfo())
			}
		}
		pos.File = nFile
		pos.Pos = gPersist.GlobalBlockFileInfo[nFile].Size
	}

	if nFile != gPersist.GlobalLastBlockFile {
		if !fKnown {
			log.Info(fmt.Sprintf("Leaving block file %d: %s\n", int(gPersist.GlobalLastBlockFile),
				gPersist.GlobalBlockFileInfo[gPersist.GlobalLastBlockFile].String()))
		}
		FlushBlockFile(!fKnown)
		gPersist.GlobalLastBlockFile = nFile
	}
	gPersist.GlobalBlockFileInfo[nFile].AddBlock(nHeight, nTime)
	if fKnown{
		maxSize := uint32(pos.Pos) + nAddSize
		if maxSize < gPersist.GlobalBlockFileInfo[nFile].Size{
			maxSize =  gPersist.GlobalBlockFileInfo[nFile].Size
		}
		gPersist.GlobalBlockFileInfo[nFile].Size = maxSize
		ret = true
	}else{
		gPersist.GlobalBlockFileInfo[nFile].Size += nAddSize
		nNewSize := gPersist.GlobalBlockFileInfo[nFile].Size
		
		nOldChunks := (pos.Pos + global.BlockFileChunkSize - 1)/global.BlockFileChunkSize
		nNewChunks := (nNewSize + global.BlockFileChunkSize -1)/global.BlockFileChunkSize
		if nNewChunks > nOldChunks{
			allocateSize := nNewChunks*global.BlockFileChunkSize - pos.Pos
			if CheckDiskSpace(allocateSize){
				file := OpenBlockFile(pos, false)
				if file != nil{
					log.Info("pre-allocating up to position %#v in blk%05u.dat\n", nNewChunks*global.BlockFileChunkSize, pos.File)
					AllocateFileRange(file, pos.Pos, allocateSize)
					file.Close()
					ret = true
				}else{
					ret = false
				}
			}else{
				ret = false
			}
		}else{
			ret = true
		}
	}
	gPersist.GlobalDirtyFileInfo[nFile] = true
	return ret
}

func FindUndoPos( nFile int32, undoPos *block.DiskBlockPos, nAddSize int) error {
	undoPos.File = nFile
	global.CsLastBlockFile.Lock()
	defer global.CsLastBlockFile.Unlock()
	gPersist := global.GetInstance()
	undoPos.Pos = (gPersist.GlobalBlockFileInfo)[nFile].UndoSize
	gPersist.GlobalBlockFileInfo[nFile].UndoSize += uint32(nAddSize)
	nNewSize := gPersist.GlobalBlockFileInfo[nFile].UndoSize
	gPersist.GlobalDirtyFileInfo[nFile] = true
	
	nOldChunks := (undoPos.Pos + global.UndoFileChunkSize - 1) / global.UndoFileChunkSize
	nNewChunks := (nNewSize + global.UndoFileChunkSize - 1) / global.UndoFileChunkSize
	
	if nNewChunks > uint32(nOldChunks) {
		
		if CheckDiskSpace(nNewChunks*global.UndoFileChunkSize - undoPos.Pos) {
			file := OpenUndoFile(*undoPos, false)
			if file != nil {
				log.Info("Pre-allocating up to position 0x%x in rev%05u.dat\n",
					nNewChunks*global.UndoFileChunkSize, undoPos.File)
				AllocateFileRange(file, undoPos.Pos, nNewChunks*global.UndoFileChunkSize-uint32(undoPos.Pos))
				file.Close()
			} else {
				return errcode.New(errcode.ErrorNotFindUndoFile)
				
				
			}
		} else {
			return errcode.New(errcode.ErrorOutOfDiskSpace)
		}
	}
	
	return nil
}