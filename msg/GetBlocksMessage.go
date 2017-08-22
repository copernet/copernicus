package msg

import (
	"fmt"
	"github.com/btcboost/copernicus/protocol"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
	"io"
)

const MaxGetBlocksCount = 500

type GetBlocksMessage struct {
	ProtocolVersion uint32
	BlockHashes     []*utils.Hash
	HashStop        *utils.Hash
}

func (getBlocksMessage *GetBlocksMessage) AddBlockHash(hash *utils.Hash) error {
	if len(getBlocksMessage.BlockHashes) > MaxAddressesCount-1 {
		str := fmt.Sprintf("block hashes is too many %v", MaxAddressesCount)
		return errors.New(str)
	}
	getBlocksMessage.BlockHashes = append(getBlocksMessage.BlockHashes, hash)
	return nil
}
func (getBlocksMessage *GetBlocksMessage) BitcoinParse(reader io.Reader, size uint32) error {
	err := protocol.ReadElement(reader, &getBlocksMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	count, err := utils.ReadVarInt(reader)
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
		getBlocksMessage.AddBlockHash(hash)
	}
	err = protocol.ReadElement(reader, getBlocksMessage.HashStop)
	return err

}

func (getBlocksMessage *GetBlocksMessage) BitcoinSerialize(w io.Writer, size uint32) error {

	count := len(getBlocksMessage.BlockHashes)
	if count > MaxGetBlocksCount {
		str := fmt.Sprintf("too many block hashes for message count:%v,max %v", count, MaxGetBlocksCount)
		return errors.New(str)
	}
	err := protocol.WriteElement(w, getBlocksMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(w, uint64(count))
	if err != nil {
		return err
	}
	for _, hash := range getBlocksMessage.BlockHashes {
		err = protocol.WriteElement(w, hash)
		if err != nil {
			return err
		}
	}
	err = protocol.WriteElement(w, &getBlocksMessage.HashStop)
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
