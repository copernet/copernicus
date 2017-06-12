package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"copernicus/utils"
	"copernicus/crypto"
	"copernicus/protocol"
)

const MAX_GETBLOCKS_COUNT = 500

type GetBlocksMessage struct {
	ProtocolVersion uint32
	BlockHashes     []*crypto.Hash
	HashStop        *crypto.Hash
}

func (getBlockMessage *GetBlocksMessage) AddBlockHash(hash *crypto.Hash) error {
	if len(getBlockMessage.BlockHashes) > MAX_ADDRESSES_COUNT-1 {
		str := fmt.Sprintf("block hashes is too many %v", MAX_ADDRESSES_COUNT)
		return errors.New(str)
	}
	getBlockMessage.BlockHashes = append(getBlockMessage.BlockHashes, hash)
	return nil
}
func (getBlockMessage *GetBlocksMessage) BitcoinParse(reader io.Reader, size uint32) error {
	err := utils.ReadElement(reader, &getBlockMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	count, err := utils.ReadVarInt(reader, size)
	if err != nil {
		return err
	}
	if count > MAX_ADDRESSES_COUNT {
		str := fmt.Sprintf("block hashes is too many %v ,max %v", count, MAX_ADDRESSES_COUNT)
		return errors.New(str)
	}
	// Create a contiguous slice of hashes to deserialize into in order to
	// reduce the number of allocations.
	hashes := make([]*crypto.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := hashes[i]
		err = utils.ReadElement(reader, hash)
		if err != nil {
			return err
		}
		getBlockMessage.AddBlockHash(hash)
	}
	err = utils.ReadElement(reader, getBlockMessage.HashStop)
	if err != nil {
		return err
	}
	return nil

}

func (getBlockMessage*GetBlocksMessage) BitcoinSerialize(w io.Writer, size uint32) error {

	count := len(getBlockMessage.BlockHashes)
	if count > MAX_GETBLOCKS_COUNT {
		str := fmt.Sprintf("too many block hashes for message count:%v,max %v", count, MAX_GETBLOCKS_COUNT)
		return errors.New(str)
	}
	err := utils.WriteElement(w, getBlockMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(w, size, uint64(count))
	if err != nil {
		return err
	}
	for _, hash := range getBlockMessage.BlockHashes {
		err = utils.WriteElement(w, hash)
		if err != nil {
			return err
		}
	}
	err = utils.WriteElement(w, &getBlockMessage.HashStop)
	if err != nil {
		return err
	}
	return nil
}
func (getBlocksMessage *GetBlocksMessage) MaxPayloadLength(size uint32) uint32 {
	return 4 + MAX_VAR_INT_PAYLOAD + (MAX_GETBLOCKS_COUNT * crypto.HASH_SIZE) + crypto.HASH_SIZE
}

func NewGetBlocksMessage(hashStop *crypto.Hash) *GetBlocksMessage {
	getBlockMessage := GetBlocksMessage{
		ProtocolVersion: protocol.BITCOIN_PROTOCOL_VERSION,
		BlockHashes:     make([]*crypto.Hash, 0, MAX_BLOCK_SIZE),
		HashStop:        hashStop,
	}
	return &getBlockMessage
}
