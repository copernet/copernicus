package disk

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"reflect"
	"syscall"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"

	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/pow"

	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/chainparams"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/global"
	"github.com/copernet/copernicus/util"
	"gopkg.in/fatih/set.v0"
)

type FlushStateMode int

var (
	ChainActive chain.Chain
	gps         = global.InitPruneState()
)

const (
	FlushStateNone FlushStateMode = iota
	FlushStateIfNeeded
	FlushStatePeriodic
	FlushStateAlways
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
	parentPath := GetBlockPosParentFilename()
	e := os.MkdirAll(parentPath, os.ModePerm)
	if e != nil {
		log.Error("e=========", e)
		panic("OpenDiskFile.os.MkdirAll(parentPath err")
	}
	filePath := GetBlockPosFilename(pos, prefix)
	flag := 0
	if fReadOnly {
		flag |= os.O_RDONLY
	} else {
		//flag |= os.O_APPEND | os.O_WRONLY
		flag |= os.O_WRONLY
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		flag |= os.O_CREATE
	}
	file, err := os.OpenFile(filePath, flag, os.ModePerm)
	if file == nil || err != nil {
		log.Error("Unable to open file %s\n", err)
		panic("Unable to open file ======")
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
	return conf.Cfg.DataDir + "/blocks/"
}

func AllocateFileRange(file *os.File, offset uint32, length uint32) {
	// Fallback version
	// TODO: just write one byte per block
	var buf [65536]byte
	file.Seek(int64(offset), os.SEEK_SET)
	for length > 0 {
		now := 65536
		if int(length) < now {
			now = int(length)
		}
		// Allowed to fail; this function is advisory anyway.
		_, err := file.Write(buf[:now])
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
	err := bu.Serialize(buf)
	if err != nil {
		log.Error("disk:serialize block undo failed.")
		return err
	}
	size := buf.Len() + 32
	buHasher := sha256.New()
	_, err = buHasher.Write(hashBlock[:])
	if err != nil {
		log.Error("disk:write hashBlock failed.")
		return err
	}
	_, err = buHasher.Write(buf.Bytes())
	if err != nil {
		log.Error("disk:write buf.Bytes() failed.")
		return err
	}
	buHash := buHasher.Sum(nil)
	_, err = buf.Write(buHash)
	if err != nil {
		log.Error("disk:write block undo hash failed.")
		return err
	}
	lenBuf := bytes.NewBuffer(nil)
	util.BinarySerializer.PutUint32(lenBuf, binary.LittleEndian, uint32(size))

	_, err = undoFile.Write(lenBuf.Bytes())
	if err != nil {
		log.Error("disk:WriteUndoFile failed")
		return err
	}
	_, err = undoFile.Write(buf.Bytes())
	if err != nil {
		log.Error("disk:WriteUndoFile failed")
		return err
	}

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
	buf := make([]byte, size)
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
	_, err = buHasher.Write(hashblock[:])
	if err != nil {
		log.Error("disk:write block undo hashBlock failed.")
	}
	_, err = buHasher.Write(undoData)
	if err != nil {
		log.Error("disk:write block undo undoData failed.")
	}
	buHash := buHasher.Sum(nil)
	return bu, reflect.DeepEqual(checkSumData, buHash)

}

func readBlockFromDiskByPos(pos block.DiskBlockPos, param *chainparams.BitcoinParams) (*block.Block, bool) {

	// Open history file to read
	file := OpenBlockFile(&pos, true)
	if file == nil {
		log.Error("ReadBlockFromDisk: OpenBlockFile failed for %s", pos.String())
		return nil, false
	}
	defer file.Close()

	size, err := util.BinarySerializer.Uint32(file, binary.LittleEndian)
	if err != nil {
		log.Error("ReadBlockFromDisk: read block file len failed for %s", pos.String())
		return nil, false
	}
	//read block data to tmp buff
	tmp := make([]byte, size)
	n, err := file.Read(tmp)
	if err != nil || n != int(size) {
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
	blk, ret := readBlockFromDiskByPos(pindex.GetBlockPos(), param)
	if !ret {
		return nil, false
	}
	hash := pindex.GetBlockHash()
	pos := pindex.GetBlockPos()
	blockHash := blk.GetHash()
	if !bytes.Equal(blockHash[:], hash[:]) {
		log.Error(fmt.Sprintf("ReadBlockFromDisk(CBlock&, CBlockIndex*): GetHash()"+
			"doesn't match index for %s at %s", pindex.String(), pos.String()))
		return blk, false
	}
	return blk, true
}

func WriteBlockToDisk(block *block.Block, pos *block.DiskBlockPos) bool {
	// Open history file to append
	file := OpenBlockFile(pos, false)
	if file == nil {
		log.Error("OpenBlockFile failed")
		return false
	}
	defer file.Close()
	buf := bytes.NewBuffer(nil)
	err := block.Serialize(buf)
	if err != nil {
		log.Error("Serialize buf failed, please check.")
		return false
	}
	size := buf.Len()
	lenBuf := bytes.NewBuffer(nil)
	err = util.BinarySerializer.PutUint32(lenBuf, binary.LittleEndian, uint32(size))
	if err != nil {
		log.Error("Write Block To Disk failed")
		return false
	}
	lenData := lenBuf.Bytes()
	_, err = file.Write(lenData)
	if err != nil {
		log.Error("Write Block To Disk failed")
		return false
	}
	_, err = file.Write(buf.Bytes())
	if err != nil {
		log.Error("Write Block To Disk failed")
		return false
	}
	return true
}

func FlushStateToDisk(mode FlushStateMode, nManualPruneHeight int) error {
	var (
		params          = chainparams.ActiveNetParams
		setFilesToPrune = set.New()
	)

	global.CsLastBlockFile.Lock()
	defer global.CsLastBlockFile.Unlock()

	coinsTip := utxo.GetUtxoCacheInstance()
	blockTree := blkdb.GetInstance()
	gPersist := global.GetInstance()
	mem := mempool.GetInstance()
	flushForPrune := false
	dbPeakUsageFactor := int64(2)
	maxBlockCoinsDBUsage := float64(dbPeakUsageFactor * 200)
	coinCacheUsage := 5000 * 300
	dataBaseWriteInterval := 60 * 60
	dataBaseFlushInterval := 24 * 60 * 60
	minBlockCoinsDBUsage := 50 * dbPeakUsageFactor

	if gps.PruneMode && (gps.CheckForPruning || nManualPruneHeight > 0) && !gps.Reindex {
		FindFilesToPruneManual(setFilesToPrune, nManualPruneHeight)
	} else {
		FindFilesToPrune(setFilesToPrune, uint64(params.PruneAfterHeight))
		gps.CheckForPruning = false
	}
	if !setFilesToPrune.IsEmpty() {
		flushForPrune = true
		if !gps.HavePruned {
			err := blockTree.WriteFlag("prunedblockfiles", true)
			if err != nil {
				log.Error("write flag prunedblockfiles failed.")
			}
			gps.HavePruned = true
		}
	}

	nNow := time.Now().UnixNano()
	// Avoid writing/flushing immediately after startup.
	if gPersist.GlobalLastWrite == 0 {
		gPersist.GlobalLastWrite = int(nNow)
	}
	if gPersist.GlobalLastFlush == 0 {
		gPersist.GlobalLastFlush = int(nNow)
	}
	if gPersist.GlobalLastSetChain == 0 {
		gPersist.GlobalLastSetChain = int(nNow)
	}

	mempoolUsage := mem.GetPoolUsage()
	mempoolSizeMax := int64(global.DefaultMaxMemPoolSize) * 1000000
	cacheSize := coinsTip.DynamicMemoryUsage() * dbPeakUsageFactor
	totalSpace := float64(coinCacheUsage) + math.Max(float64(mempoolSizeMax-mempoolUsage), 0)
	// The cache is large and we're within 10% and 200 MiB or 50% and 50MiB
	// of the limit, but we have time now (not in the middle of a block processing).
	x := math.Max(totalSpace/2, totalSpace-float64(minBlockCoinsDBUsage*1024*1024))
	y := math.Max(9*totalSpace/10, totalSpace-maxBlockCoinsDBUsage*1024*1024)
	cacheLarge := mode == FlushStatePeriodic && float64(cacheSize) > math.Min(x, y)
	// The cache is over the limit, we have to write now.
	cacheCritical := mode == FlushStateIfNeeded && float64(cacheSize) > totalSpace
	// It's been a while since we wrote the block index to disk. Do this
	// frequently, so we don't need to redownLoad after a crash.
	periodicWrite := mode == FlushStatePeriodic && int(nNow) > gPersist.GlobalLastWrite+dataBaseWriteInterval*1000000
	// It's been very long since we flushed the cache. Do this infrequently,
	// to optimize cache usage.
	periodicFlush := mode == FlushStatePeriodic && int(nNow) > gPersist.GlobalLastFlush+dataBaseFlushInterval*1000000
	// Combine all conditions that result in a full cache flush.
	doFullFlush := mode == FlushStateAlways || cacheLarge || cacheCritical || periodicFlush || flushForPrune
	// Write blocks and block index to disk.
	if doFullFlush || periodicWrite {
		// Depend on nMinDiskSpace to ensure we can write block index
		if !CheckDiskSpace(0) {
			return errcode.New(errcode.ErrorOutOfDiskSpace)
		}
		// Make sure all block and undo data is flushed to disk.
		FlushBlockFile(false)

		// Update dirty block file information (which may refer to block and undo files).
		dirtyBlockFileInfoList := make(map[int32]*block.BlockFileInfo)
		for k, ok := range gPersist.GlobalDirtyFileInfo {
			if ok {
				dirtyBlockFileInfoList[k] = gPersist.GlobalBlockFileInfo[k]
			}
		}
		gPersist.GlobalDirtyFileInfo = make(map[int32]bool)

		// Update dirty block index
		dirtyBlockIndexList := make([]*blockindex.BlockIndex, 0, len(gPersist.GlobalDirtyBlockIndex))
		for _, bi := range gPersist.GlobalDirtyBlockIndex {
			dirtyBlockIndexList = append(dirtyBlockIndexList, bi)
		}
		gPersist.GlobalDirtyBlockIndex = make(map[util.Hash]*blockindex.BlockIndex)

		// Write dirty block file info, last blockfile and dirty blockindex to db
		err := blockTree.WriteBatchSync(dirtyBlockFileInfoList, int(gPersist.GlobalLastBlockFile), dirtyBlockIndexList)
		if err != nil {
			return errcode.New(errcode.ErrorFailedToWriteToBlockIndexDatabase)
		}

		// Finally remove any pruned files
		if flushForPrune {
			UnlinkPrunedFiles(setFilesToPrune)
		}
		gPersist.GlobalLastWrite = int(nNow)
	}

	// Flush best chain related state. This can only be done if the blocks
	// block index write was also done.
	if doFullFlush {
		// Typical Coin structures on disk are around 48 bytes in size.
		// Pushing a new one to the database can cause it to be written
		// twice (once in the log, and once in the tables). This is already
		// an overestimation, as most will delete an existing entry or
		// overwrite one. Still, use a conservative safety factor of 2.
		if !CheckDiskSpace(uint32(48 * 2 * 2 * coinsTip.GetCacheSize())) {
			log.Error("out of disk space!")
		}
		// Flush the chainState (which may refer to block index entries).
		if !coinsTip.Flush() {
			log.Error(errcode.New(errcode.ErrorFailedToWriteToCoinDatabase))
			panic("write db failed, please check.")

		}
		gPersist.GlobalLastFlush = int(nNow)
	}
	if doFullFlush || ((mode == FlushStateAlways || mode == FlushStatePeriodic) &&
		int(nNow) > gPersist.GlobalLastSetChain+dataBaseWriteInterval*1000000) {
		// Update best block in wallet (so we can detect restored wallets).
		gPersist.GlobalLastSetChain = int(nNow)
	}

	return nil
}

func CheckDiskSpace(nAdditionalBytes uint32) bool {
	path := conf.Cfg.DataDir
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
	needSize := uint64(MinDiskSpace + n)
	if nFreeBytesAvailable < needSize {
		log.Error("Error: Disk space is low!")
		panic("Error: Disk space is low!")
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
	if nAddSize > global.MaxBlockFileSize {
		log.Error("FindBlockPos nAddSize [%#v] is too large more then global.MaxBlockFileSize ", nAddSize)
		panic("FindBlockPos nAddSize  is too large more then global.MaxBlockFileSize")
	}
	gPersist := global.GetInstance()
	ret := false
	nFile := pos.File
	if !fKnown {
		nFile = gPersist.GlobalLastBlockFile
	}
	if len(gPersist.GlobalBlockFileInfo) <= int(nFile) {
		gPersist.GlobalBlockFileInfo = append(gPersist.GlobalBlockFileInfo, block.NewBlockFileInfo())
	}
	if !fKnown {
		for gPersist.GlobalBlockFileInfo[nFile].Size+nAddSize >= global.MaxBlockFileSize {
			nFile++
			if len(gPersist.GlobalBlockFileInfo) <= int(nFile) {
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
	if fKnown {
		maxSize := pos.Pos + nAddSize
		if maxSize < gPersist.GlobalBlockFileInfo[nFile].Size {
			maxSize = gPersist.GlobalBlockFileInfo[nFile].Size
		}
		gPersist.GlobalBlockFileInfo[nFile].Size = maxSize
		ret = true
	} else {
		gPersist.GlobalBlockFileInfo[nFile].Size += nAddSize
		nNewSize := gPersist.GlobalBlockFileInfo[nFile].Size

		nOldChunks := (pos.Pos + global.BlockFileChunkSize - 1) / global.BlockFileChunkSize
		nNewChunks := (nNewSize + global.BlockFileChunkSize - 1) / global.BlockFileChunkSize
		if nNewChunks > nOldChunks {
			allocateSize := nNewChunks*global.BlockFileChunkSize - pos.Pos
			if CheckDiskSpace(allocateSize) {
				file := OpenBlockFile(pos, false)
				if file != nil {
					log.Info("pre-allocating up to position %#v in blk%05d.dat\n", nNewChunks*global.BlockFileChunkSize, pos.File)
					AllocateFileRange(file, pos.Pos, allocateSize)
					file.Close()
					ret = true
				} else {
					ret = false
				}
			} else {
				ret = false
			}
		} else {
			ret = true
		}
	}
	gPersist.GlobalDirtyFileInfo[nFile] = true
	return ret
}

func FindUndoPos(nFile int32, undoPos *block.DiskBlockPos, nAddSize int) error {
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

	if nNewChunks > nOldChunks {

		if CheckDiskSpace(nNewChunks*global.UndoFileChunkSize - undoPos.Pos) {
			file := OpenUndoFile(*undoPos, false)
			if file != nil {
				log.Info("Pre-allocating up to position 0x%x in rev%05u.dat\n",
					nNewChunks*global.UndoFileChunkSize, undoPos.File)
				AllocateFileRange(file, undoPos.Pos, nNewChunks*global.UndoFileChunkSize-undoPos.Pos)
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

/**
 * BLOCK PRUNING CODE
 */

// CalculateCurrentUsage Calculate the amount of disk space the block & undo files currently use
func CalculateCurrentUsage() uint64 {
	gPersist := global.GetInstance()
	var retval uint64
	for _, file := range gPersist.GlobalBlockFileInfo {
		retval += uint64(file.Size + file.UndoSize)
	}
	return retval
}

// FindFilesToPrune calculate the block/rev files that should be deleted to remain under target
func FindFilesToPrune(setFilesToPrune *set.Set, nPruneAfterHeight uint64) {
	gPersist := global.GetInstance()

	if ChainActive.Tip() == nil || gps.PruneTarget == 0 {
		return
	}
	if uint64(ChainActive.Tip().Height) <= nPruneAfterHeight {
		return
	}
	nLastBlockWeCanPrune := ChainActive.Tip().Height - block.MinBlocksToKeep
	nCurrentUsage := CalculateCurrentUsage()
	// We don't check to prune until after we've allocated new space for files,
	// so we should leave a buffer under our target to account for another
	// allocation before the next pruning.
	nBuffer := uint64(global.BlockFileChunkSize + global.UndoFileChunkSize)
	count := 0
	if nCurrentUsage+nBuffer >= gps.PruneTarget {
		for fileNumber := 0; int32(fileNumber) < gPersist.GlobalLastBlockFile; fileNumber++ {
			nBytesToPrune := uint64(gPersist.GlobalBlockFileInfo[fileNumber].Size + gPersist.GlobalBlockFileInfo[fileNumber].UndoSize)
			if gPersist.GlobalBlockFileInfo[fileNumber].Size == 0 {
				continue
			}
			// are we below our target?
			if nCurrentUsage+nBuffer < gps.PruneTarget {
				break
			}
			// don't prune files that could have a block within
			// MIN_BLOCKS_TO_KEEP of the main chain's tip but keep scanning
			if gPersist.GlobalBlockFileInfo[fileNumber].HeightLast > nLastBlockWeCanPrune {
				continue
			}

			PruneOneBlockFile(int32(fileNumber))
			// Queue up the files for removal
			setFilesToPrune.Add(fileNumber)
			nCurrentUsage -= nBytesToPrune
			count++
		}
	}

	log.Info("prune", "Prune: target=%dMiB actual=%dMiB diff=%dMiB max_prune_height=%d removed %d blk/rev pairs\n",
		gps.PruneTarget/1024/1024, nCurrentUsage/1024/1024, (gps.PruneTarget-nCurrentUsage)/1024/1024, nLastBlockWeCanPrune, count)
}

func FindFilesToPruneManual(setFilesToPrune *set.Set, manualPruneHeight int) {
	gPersist := global.GetInstance()
	if gps.PruneMode && manualPruneHeight <= 0 {
		panic("the PruneMode is false and manualPruneHeight equal zero")
	}

	global.CsLastBlockFile.Lock()
	defer global.CsLastBlockFile.Unlock()

	if ChainActive.Tip() == nil {
		return
	}

	// last block to prune is the lesser of (user-specified height, MIN_BLOCKS_TO_KEEP from the tip)
	lastBlockWeCanPrune := math.Min(float64(manualPruneHeight), float64(ChainActive.Tip().Height-block.MinBlocksToKeep))
	count := 0
	for fileNumber := 0; int32(fileNumber) < gPersist.GlobalLastBlockFile; fileNumber++ {
		if gPersist.GlobalBlockFileInfo[fileNumber].Size == 0 || gPersist.GlobalBlockFileInfo[fileNumber].HeightLast > gPersist.GlobalLastBlockFile {
			continue
		}
		PruneOneBlockFile(int32(fileNumber))
		setFilesToPrune.Add(fileNumber)
		count++
	}
	log.Info("Prune (Manual): prune_height=%d removed %d blk/rev pairs\n", lastBlockWeCanPrune, count)
}

// PruneOneBlockFile prune a block file (modify associated database entries)
func PruneOneBlockFile(fileNumber int32) {
	bm := make(map[util.Hash]*blockindex.BlockIndex)
	gPersist := global.GetInstance()
	for _, value := range bm {
		pindex := value
		if pindex.File == fileNumber {
			pindex.Status &= ^blockindex.BlockHaveData
			pindex.Status &= ^blockindex.BlockHaveUndo
			pindex.File = 0
			pindex.DataPos = 0
			pindex.UndoPos = 0
			gPersist.AddDirtyBlockIndex(pindex)

			// Prune from mapBlocksUnlinked -- any block we prune would have
			// to be downloaded again in order to consider its chain, at which
			// point it would be considered as a candidate for
			// mapBlocksUnlinked or setBlockIndexCandidates.
			ranges := gPersist.GlobalMapBlocksUnlinked[pindex.Prev]
			tmpRange := make([]*blockindex.BlockIndex, len(ranges))
			copy(tmpRange, ranges)
			for len(tmpRange) > 0 {
				v := tmpRange[0]
				tmpRange = tmpRange[1:]
				if v == pindex {
					tmp := make([]*blockindex.BlockIndex, len(ranges)-1)
					for _, val := range tmpRange {
						if val != v {
							tmp = append(tmp, val)
						}
					}
					gPersist.GlobalMapBlocksUnlinked[pindex.Prev] = tmp
				}
			}
		}
	}

	gPersist.GlobalBlockFileInfo[fileNumber].SetNull()
	gPersist.GlobalDirtyFileInfo[fileNumber] = true
}

func UnlinkPrunedFiles(setFilesToPrune *set.Set) {
	lists := setFilesToPrune.List()
	for key, value := range lists {
		v := value.(int32)
		pos := &block.DiskBlockPos{
			File: v,
			Pos:  0,
		}
		os.Remove(GetBlockPosFilename(*pos, "blk"))
		os.Remove(GetBlockPosFilename(*pos, "rev"))
		log.Info("Prune: %s deleted blk/rev (%05u)\n", key)
	}
}

func GetPruneState() *global.PruneState {
	return gps
}
