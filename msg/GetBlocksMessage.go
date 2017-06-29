package msg

import (
	"fmt"
	"github.com/btccom/copernicus/protocol"
	"github.com/btccom/copernicus/utils"
	"github.com/pkg/errors"
	"io"
)

const MaxGetBlocksCount = 500

type GetBlocksMessage struct {
	ProtocolVersion uint32
	BlockHashes     []*utils.Hash
	HashStop        *utils.Hash
}

func (getBlockMessage *GetBlocksMessage) AddBlockHash(hash *utils.Hash) error {
	if len(getBlockMessage.BlockHashes) > MaxAddressesCount-1 {
		str := fmt.Sprintf("block hashes is too many %v", MaxAddressesCount)
		return errors.New(str)
	}
	getBlockMessage.BlockHashes = append(getBlockMessage.BlockHashes, hash)
	return nil
}
func (getBlockMessage *GetBlocksMessage) BitcoinParse(reader io.Reader, size uint32) error {
	err := protocol.ReadElement(reader, &getBlockMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	count, err := utils.ReadVarInt(reader, size)
	if err != nil {
		return err
	}
	if count > MaxAddressesCount {
		str := fmt.Sprintf("block hashes is too many %v ,max %v", count, MaxAddressesCount)
		return errors.New(str)
	}
	// Create a contiguous slice of hashes to deserialize into in order to
	// reduce the number of allocations.
	hashes := make([]*utils.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := hashes[i]
		err = protocol.ReadElement(reader, hash)
		if err != nil {
			return err
		}
		getBlockMessage.AddBlockHash(hash)
	}
	err = protocol.ReadElement(reader, getBlockMessage.HashStop)
	return err

}

func (getBlockMessage *GetBlocksMessage) BitcoinSerialize(w io.Writer, size uint32) error {

	count := len(getBlockMessage.BlockHashes)
	if count > MaxGetBlocksCount {
		str := fmt.Sprintf("too many block hashes for message count:%v,max %v", count, MaxGetBlocksCount)
		return errors.New(str)
	}
	err := protocol.WriteElement(w, getBlockMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(w, size, uint64(count))
	if err != nil {
		return err
	}
	for _, hash := range getBlockMessage.BlockHashes {
		err = protocol.WriteElement(w, hash)
		if err != nil {
			return err
		}
	}
	err = protocol.WriteElement(w, &getBlockMessage.HashStop)
	return err

}
func (getBlocksMessage *GetBlocksMessage) MaxPayloadLength(size uint32) uint32 {
	return 4 + MaxVarIntPayload + (MaxGetBlocksCount * utils.HashSize) + utils.HashSize
}
func (getBlocksMessage *GetBlocksMessage) Command() string {
	return CommandGetBlocks
}

func NewGetBlocksMessage(hashStop *utils.Hash) *GetBlocksMessage {
	getBlockMessage := GetBlocksMessage{
		ProtocolVersion: protocol.BitcoinProtocolVersion,
		BlockHashes:     make([]*utils.Hash, 0, MaxBlockSize),
		HashStop:        hashStop,
	}
	return &getBlockMessage
}
