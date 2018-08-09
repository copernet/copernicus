package block

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/copernet/copernicus/util"
)

type BlockFileInfo struct {
	Blocks      uint32 // number of blocks stored in file
	Size        uint32 // number of used bytes of block file
	UndoSize    uint32 // number of used bytes in the undo file
	HeightFirst int32  // lowest height of block in file
	HeightLast  int32  // highest height of block in file
	timeFirst   uint64 // earliest time of block in file
	timeLast    uint64 // latest time of block in file
	//index       uint32
}

func (bfi *BlockFileInfo) GetSerializeList() []string {
	dumpList := []string{"Blocks", "Size", "UndoSize", "HeightFirst", "HeightLast", "timeFirst", "timeLast", "index"}
	return dumpList
}

func (bfi *BlockFileInfo) Serialize(w io.Writer) error {
	buf := bytes.NewBuffer(nil)
	err := util.WriteElements(buf, bfi.Blocks, bfi.Size, bfi.UndoSize, bfi.HeightFirst, bfi.HeightLast, bfi.timeFirst, bfi.timeLast)
	if err != nil {
		return err
	}
	_, err = w.Write(buf.Bytes())
	return err
}

func (bfi *BlockFileInfo) Unserialize(r io.Reader) error {
	err := util.ReadElements(r, &bfi.Blocks, &bfi.Size, &bfi.UndoSize, &bfi.HeightFirst, &bfi.HeightLast, &bfi.timeFirst, &bfi.timeLast)
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
func (bfi *BlockFileInfo) AddBlock(nHeightIn int32, timeIn uint64) {
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
