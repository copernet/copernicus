package blockindex

import (
	"fmt"
	"math/big"
	"sort"

	"bytes"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/util"
	"io"
)

/**
 * The block chain is a tree shaped structure starting with the genesis block at
 * the root, with each block potentially having multiple candidates to be the
 * next block. A blockIndex may have multiple prev pointing to it, but at most
 * one of them can be part of the currently active branch.
 */

//const (
//	StatusAllValid uint32 = 1 << iota
//	StatusIndexStored
//	StatusWaitingData
//	StatusDataStored
//	StatusFailed
//	StatusAccepted
//
//	// StatusNone NOTE: This must be defined last in order to avoid influencing iota
//	StatusNone = 0
//)

type BlockIndex struct {
	Header block.BlockHeader
	// pointer to the hash of the block, if any.
	blockHash util.Hash
	// pointer to the index of the predecessor of this block
	Prev *BlockIndex

	// pointer to the index of some further predecessor of this block
	//Skip *BlockIndex

	// height of the entry in the chain. The genesis block has height 0；
	Height int32
	// Which # file this block is stored in (blk?????.dat)
	File int32
	// Byte offset within blk?????.dat where this block's data is stored
	DataPos uint32
	// Byte offset within rev?????.dat where this block's undo data is stored
	UndoPos uint32
	// (memory only) Total amount of work (expected number of hashes) in the
	// chain up to and including this block
	ChainWork big.Int
	// Number of transactions in this block.
	// Note: in a potential headers-first mode, this number cannot be relied
	// upon
	TxCount int32
	// (memory only) Number of transactions in the chain up to and including
	// this block.
	// This value will be non-zero only if and only if transactions for this
	// block and all its parents are available. Change to 64-bit type when
	// necessary; won't happen before 2030
	ChainTxCount int32
	//status of this block. See enum
	Status uint32
	// (memory only) Sequential id assigned to distinguish order in which
	// blocks are received.
	SequenceID uint64
	// (memory only) Maximum time in the chain upto and including this block.
	TimeMax   uint32
	isGenesis bool
}

const medianTimeSpan = 11

func (bIndex *BlockIndex) SetNull() {
	bIndex.Header.SetNull()
	bIndex.blockHash = util.Hash{}
	bIndex.Prev = nil
	//bIndex.Skip = nil

	bIndex.Height = 0
	bIndex.File = -1
	bIndex.DataPos = 0
	bIndex.UndoPos = 0
	bIndex.ChainWork = big.Int{}
	bIndex.ChainTxCount = 0
	bIndex.TxCount = 0
	//bIndex.Status = StatusNone
	bIndex.Status = 0
	bIndex.SequenceID = 0
	bIndex.TimeMax = 0
}

//func (bIndex *BlockIndex) WaitingData() bool {
//	return bIndex.Status&StatusWaitingData != 0
//}
//
//func (bIndex *BlockIndex) AllValid() bool {
//	return bIndex.Status&StatusAllValid != 0
//}
//
//func (bIndex *BlockIndex) IndexStored() bool {
//	return bIndex.Status&StatusIndexStored != 0
//}
//
//func (bIndex *BlockIndex) DataStored() bool {
//	return bIndex.Status&StatusDataStored != 0
//}
//
//func (bIndex *BlockIndex) Accepted() bool {
//	return bIndex.Status&StatusAccepted != 0
//}
//
//func (bIndex *BlockIndex) Failed() bool {
//	return bIndex.Status&StatusFailed != 0
//}

func (bIndex *BlockIndex) GetUndoPos() block.DiskBlockPos {
	ret := block.NewDiskBlkPos()
	if bIndex.HasUndo() {
		ret.File = bIndex.File
		ret.Pos = bIndex.UndoPos
	}

	return ret
}

func (bIndex *BlockIndex) GetBlockPos() block.DiskBlockPos {
	ret := block.NewDiskBlkPos()
	if bIndex.HasData() {
		ret.File = bIndex.File
		ret.Pos = bIndex.DataPos
	}

	return ret
}

func (bIndex *BlockIndex) GetBlockHeader() *block.BlockHeader {

	return &bIndex.Header
}

func (bIndex *BlockIndex) GetBlockHash() *util.Hash {
	bHash := bIndex.blockHash
	if bHash.IsNull() {
		bIndex.blockHash = bIndex.Header.GetHash()
	}
	return &bIndex.blockHash
}

func (bIndex *BlockIndex) SetBlockHash(hash util.Hash) {
	bIndex.blockHash = hash
}

func (bIndex *BlockIndex) GetBlockTime() uint32 {

	return bIndex.Header.Time
}

func (bIndex *BlockIndex) GetBlockTimeMax() uint32 {
	return bIndex.TimeMax
}

func (bIndex *BlockIndex) GetMedianTimePast() int64 {
	median := make([]int64, 0, medianTimeSpan)
	index := bIndex
	numNodes := 0
	for i := 0; i < medianTimeSpan && index != nil; i++ {
		median = append(median, int64(index.GetBlockTime()))
		index = index.Prev
		numNodes++
	}
	median = median[:numNodes]
	sort.Slice(median, func(i, j int) bool {
		return median[i] < median[j]
	})

	if len(median) < 11 {
		return 0
	}
	return median[numNodes/2]
}

func (bIndex *BlockIndex) AddStatus(status uint32) {
	bIndex.Status |= status
}

func (bIndex *BlockIndex) HasData() bool {
	return bIndex.Status&BlockHaveData != 0
}

func (bIndex *BlockIndex) HasUndo() bool {
	return bIndex.Status&BlockHaveUndo != 0
}

func (bIndex *BlockIndex) SubStatus(status uint32) {
	bIndex.Status &= ^status
}

// RaiseValidity Raise the validity level of this block index entry.
// Returns true if the validity was changed.
func (bIndex *BlockIndex) RaiseValidity(upto uint32) bool {
	// Only validity flags allowed.
	if bIndex.IsInvalid() {
		return false
	}

	if bIndex.getValidity() >= upto {
		return false
	}

	bIndex.Status = (bIndex.Status & (^BlockValidMask)) | upto

	return true
}

// IsValid checks whether this block index entry is valid up to the passed validity level.
func (bIndex *BlockIndex) IsValid(upto uint32) bool {
	if bIndex.IsInvalid() {
		return false
	}
	return bIndex.getValidity() >= upto
}

// IsInvalid checks whether this block index entry is valid up to the passed validity level.
func (bIndex *BlockIndex) IsInvalid() bool {
	return bIndex.Status&BlockInvalidMask != 0
}

func (bIndex *BlockIndex) getValidity() uint32 {
	return bIndex.Status & BlockValidMask
}

// GetAncestor efficiently find an ancestor of this block.
func (bIndex *BlockIndex) GetAncestor(height int32) *BlockIndex {
	if height > bIndex.Height || height < 0 {
		return nil
	}
	if height == bIndex.Height {
		return bIndex
	}
	indexWalk := bIndex
	for indexWalk.Prev != nil {
		if indexWalk.Prev.Height == height {
			return indexWalk.Prev
		}
		indexWalk = indexWalk.Prev
	}

	return indexWalk
}

func (bIndex *BlockIndex) String() string {
	hash := bIndex.GetBlockHash()
	return fmt.Sprintf("BlockIndex(pprev=%p, height=%d, merkle=%s, hashBlock=%s)\n", bIndex.Prev,
		bIndex.Height, bIndex.Header.MerkleRoot.String(), hash.String())
}

func (bIndex *BlockIndex) IsGenesis(params *model.BitcoinParams) bool {
	bhash := bIndex.GetBlockHash()
	genesisHash := params.GenesisBlock.GetHash()
	return bhash.IsEqual(&genesisHash)
}

func (bIndex *BlockIndex) GetSerializeList() []string {
	dumpList := []string{"Height", "Status", "TxCount", "File", "DataPos", "UndoPos", "Header"}
	return dumpList
}

func (bIndex *BlockIndex) Serialize(w io.Writer) error {
	buf := bytes.NewBuffer(nil)
	clientVersion := int32(160000)
	err := util.WriteElements(buf, clientVersion, bIndex.Height, bIndex.Status, bIndex.TxCount, bIndex.File, bIndex.DataPos, bIndex.UndoPos)
	if err != nil {
		return err
	}
	err = bIndex.Header.Serialize(buf)
	if err != nil {
		return err
	}
	_, err = w.Write(buf.Bytes())
	return err
}

func (bIndex *BlockIndex) Unserialize(r io.Reader) error {
	clientVersion := int32(160000)

	err := util.ReadElements(r, &clientVersion, &bIndex.Height, &bIndex.Status, &bIndex.TxCount, &bIndex.File, &bIndex.DataPos, &bIndex.UndoPos)
	if err != nil {
		return err
	}
	err = bIndex.Header.Unserialize(r)
	return err
}

func NewBlockIndex(blkHeader *block.BlockHeader) *BlockIndex {
	bi := new(BlockIndex)
	bi.SetNull()
	bi.Header = *blkHeader
	return bi
}
