package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"github.com/btccom/copernicus/utils"
	"github.com/btccom/copernicus/protocol"
)

type GetHeadersMessage struct {
	ProtocolVersion uint32
	BlockHashes     []*utils.Hash
	HashStop        *utils.Hash
}

func (getHeadersMessage *GetHeadersMessage) AddBlockHash(hash *utils.Hash) error {
	if len(getHeadersMessage.BlockHashes) > MAX_GETBLOCKS_COUNT {
		str := fmt.Sprintf("too many block hashes for message max %v", MAX_GETBLOCKS_COUNT)
		return errors.New(str)

	}
	getHeadersMessage.BlockHashes = append(getHeadersMessage.BlockHashes, hash)
	return nil

}

func (getHeadersMessage *GetHeadersMessage) BitcoinParse(reader io.Reader, size uint32) error {
	err := protocol.ReadElement(reader, &getHeadersMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	count, err := utils.ReadVarInt(reader, size)
	if err != nil {
		return err
	}
	if count > MAX_GETBLOCKS_COUNT {
		str := fmt.Sprintf("too many block hashes for message count:%v,max %v", count, MAX_GETBLOCKS_COUNT)
		return errors.New(str)
	}
	hashes := make([]*utils.Hash, count)
	getHeadersMessage.BlockHashes = make([]*utils.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := hashes[i]
		err := protocol.ReadElement(reader, hash)
		if err != nil {
			return err
		}
		getHeadersMessage.AddBlockHash(hash)
	}
	err = protocol.ReadElement(reader, getHeadersMessage.HashStop)
	return err
}

func (getHeadersMessage *GetHeadersMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	count := len(getHeadersMessage.BlockHashes)
	if count > MAX_GETBLOCKS_COUNT {
		str := fmt.Sprintf("too many block hashes for message count %v,max %v", count, MAX_GETBLOCKS_COUNT)
		return errors.New(str)
	}
	err := protocol.WriteElement(w, getHeadersMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(w, size, uint64(count))
	if err != nil {
		return err
	}

	for _, hash := range getHeadersMessage.BlockHashes {
		err := protocol.WriteElement(w, hash)
		if err != nil {
			return err
		}
	}
	err = protocol.WriteElement(w, getHeadersMessage.HashStop)
	return err
}

func (getHeadersMessage *GetHeadersMessage) MaxPayloadLength(size uint32) uint32 {
	return 4 + MAX_VAR_INT_PAYLOAD + (MAX_GETBLOCKS_COUNT * utils.HASH_SIZE) + utils.HASH_SIZE
}
func (getHeadersMessage *GetHeadersMessage) Command() string {
	return COMMAND_GET_HEADERS
}
func NewGetHeadersMessage() *GetHeadersMessage {
	getHeadersMessage := GetHeadersMessage{BlockHashes: make([]*utils.Hash, 0, MAX_GETBLOCKS_COUNT)}
	return &getHeadersMessage
}
