package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"copernicus/utils"
	"copernicus/crypto"
)

type GetHeadersMessage struct {
	ProtocolVersion uint32
	BlockHashes     []*crypto.Hash
	HashStop        crypto.Hash
}

func (getHeadersMessage *GetHeadersMessage) AddBlockHash(hash *crypto.Hash) error {
	if len(getHeadersMessage.BlockHashes) > MAX_GETBLOCKS_COUNT {
		str := fmt.Sprintf("too many block hashes for message max %v", MAX_GETBLOCKS_COUNT)
		return errors.New(str)

	}
	getHeadersMessage.BlockHashes = append(getHeadersMessage.BlockHashes, hash)
	return nil

}

func (getHeadersMessage *GetHeadersMessage) BitcoinParse(reader io.Reader, size uint32) error {
	err := utils.ReadElement(reader, &getHeadersMessage.ProtocolVersion)
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
	hashes := make([]*crypto.Hash, count)
	getHeadersMessage.BlockHashes = make([]*crypto.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := hashes[i]
		err := utils.ReadElement(reader, hash)
		if err != nil {
			return err
		}
		getHeadersMessage.AddBlockHash(hash)
	}
	err = utils.ReadElement(reader, getHeadersMessage.HashStop)
	return err
}

func (getHeadersMessage *GetHeadersMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	count := len(getHeadersMessage.BlockHashes)
	if count > MAX_GETBLOCKS_COUNT {
		str := fmt.Sprintf("too many block hashes for message count %v,max %v", count, MAX_GETBLOCKS_COUNT)
		return errors.New(str)
	}
	err := utils.WriteElement(w, getHeadersMessage.ProtocolVersion)
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(w, size, uint64(count))
	if err != nil {
		return err
	}

	for _, hash := range getHeadersMessage.BlockHashes {
		err := utils.WriteElement(w, hash)
		if err != nil {
			return err
		}
	}
	err = utils.WriteElement(w, getHeadersMessage.HashStop)
	return err
}

func (getHeadersMessage *GetHeadersMessage) MaxPayloadLength(size uint32) uint32 {
	return 4 + MAX_VAR_INT_PAYLOAD + (MAX_GETBLOCKS_COUNT * crypto.HASH_SIZE) + crypto.HASH_SIZE
}
func NewGetHeadersMessage() *GetHeadersMessage {
	getHeadersMessage := GetHeadersMessage{BlockHashes: make([]*crypto.Hash, 0, MAX_GETBLOCKS_COUNT)}
	return &getHeadersMessage
}
