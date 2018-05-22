package block

import (
	"fmt"
	"time"

	"github.com/btcboost/copernicus/persist/db"
	"io"
)


type BlockFileInfo struct {
	Blocks      uint32 // number of blocks stored in file
	Size        uint32 // number of used bytes of block file
	UndoSize    uint32 // number of used bytes in the undo file
	HeightFirst uint32 // lowest height of block in file
	HeightLast  uint32 // highest height of block in file
	timeFirst   uint64 // earliest time of block in file
	timeLast    uint64 // latest time of block in file
	index uint32

}


func (bfi *BlockFileInfo) GetSerializeList()[]string{
	dumpList := []string{"Blocks","Size", "UndoSize", "HeightFirst", "HeightLast","timeFirst","timeLast","index"}
	return dumpList
}

func (bfi *BlockFileInfo) Serialize(writer io.Writer) error {
	return db.SerializeOP(writer, bfi)
}

func (bfi *BlockFileInfo) Unserialize(reader io.Reader) error {
	err := db.UnserializeOP(reader, bfi)
	return err

}
func (bfi *BlockFileInfo) SetNull() {
	bfi.Size = 0
	bfi.timeFirst = 0
	bfi.UndoSize = 0
	bfi.timeLast = 0
	bfi.HeightLast = 0
	bfi.HeightFirst = 0
	bfi.Blocks = 0
}
func (bfi *BlockFileInfo) AddBlock(nHeightIn uint32, timeIn uint64) {
	if bfi.Blocks == 0 || bfi.HeightFirst > nHeightIn {
		bfi.HeightFirst = nHeightIn
	}
	if bfi.Blocks == 0 || bfi.timeFirst > timeIn {
		bfi.timeFirst = timeIn
	}
	bfi.Blocks++
	if nHeightIn > bfi.HeightLast {
		bfi.HeightLast = nHeightIn
	}
	if timeIn > bfi.timeLast {
		bfi.timeLast = timeIn
	}
}

func (bfi *BlockFileInfo) GetIndex()uint32{
	return bfi.index
}
func (bfi *BlockFileInfo) SetIndex(idx uint32){
	bfi.index = idx
}
func (bfi *BlockFileInfo) String() string {
	return fmt.Sprintf("BlockFileInfo(blocks=%d, size=%d, heights=%d...%d, time=%s...%s)",
		bfi.Blocks, bfi.Size, bfi.HeightFirst, bfi.HeightLast,
		time.Unix(int64(bfi.timeFirst), 0).Format(time.RFC3339),
		time.Unix(int64(bfi.timeLast), 0).Format(time.RFC3339))
}

func NewBlockFileInfo() *BlockFileInfo {
	bfi := new(BlockFileInfo)
	return bfi
}
