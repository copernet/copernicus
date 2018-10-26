package wire

/*
import (
	"fmt"
	"io"
	"math"

	"github.com/copernet/copernicus/util"
)

type MsgGetBlockTxn struct {
	BlockHash util.Hash
	Indexes   []uint16
}

func (msg *MsgGetBlockTxn) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("getblocktxn message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgGetBlockTxn.Encode", str)
	}

	if err := util.WriteElements(w, msg.BlockHash); err != nil {
		return err
	}
	indexSize := uint64(len(msg.Indexes))
	if err := util.WriteVarInt(w, indexSize); err != nil {
		return err
	}
	for i := 0; i < len(msg.Indexes); i++ {
		var index uint16
		if i == 0 {
			index = msg.Indexes[i]
		} else {
			index = msg.Indexes[i] - (msg.Indexes[i-1] + 1)
		}
		if err := util.WriteVarInt(w, uint64(index)); err != nil {
			return err
		}
	}
	return nil
}

func (msg *MsgGetBlockTxn) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("getblocktxn message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgGetBlockTxn.Decode", str)
	}
	if err := util.ReadElements(r, &msg.BlockHash); err != nil {
		return err
	}
	indexSize, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	indexes := make([]uint16, indexSize)
	for i := 0; i < len(indexes); i++ {
		index, err := util.ReadVarInt(r)
		if err != nil {
			return err
		}
		if index > math.MaxUint16 {
			return messageError("MsgGetBlockTxn.Decode", fmt.Sprintf("index overflowed 16-bits"))
		}
		indexes[i] = uint16(index)
	}
	offset := uint16(0)
	for j := 0; j < len(indexes); j++ {
		if uint64(indexes[j])+uint64(offset) > math.MaxUint16 {
			return messageError("MsgGetBlockTxn.Decode", fmt.Sprintf("index overflowed 16-bits"))
		}
		indexes[j] += offset
		offset = indexes[j] + 1
	}
	msg.Indexes = indexes
	return nil
}

func (msg *MsgGetBlockTxn) Command() string {
	return CmdGetBlockTxn
}

func (msg *MsgGetBlockTxn) MaxPayloadLength(pver uint32) uint64 {
	return 32 + 3 + uint64(len(msg.Indexes))*3
}
*/
