// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"io"

	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/util"
)

const maxFlagsPerMerkleBlock = maxTxPerBlock / 8

// maxFlagsPerMerkleBlock is the maximum number of flag bytes that could
// possibly fit into a merkle block.  Since each transaction is represented by
// a single bit, this is the max number of transactions per block divided by
// 8 bits per byte.  Then an extra one to cover partials.

// MsgMerkleBlock implements the Message interface and represents a bitcoin
// merkleblock message which is used to reset a Bloom filter.
//
// This message was not added until protocol version BIP0037Version.
type MsgMerkleBlock struct {
	Header       block.BlockHeader
	Transactions uint32
	Hashes       []*util.Hash
	Flags        []byte
}

// AddTxHash adds a new transaction hash to the message.
func (msg *MsgMerkleBlock) AddTxHash(hash *util.Hash) error {
	if uint64(len(msg.Hashes)+1) > maxTxPerBlock {
		str := fmt.Sprintf("too many tx hashes for message [max %v]",
			maxTxPerBlock)
		return messageError("MsgMerkleBlock.AddTxHash", str)
	}

	msg.Hashes = append(msg.Hashes, hash)
	return nil
}

// Decode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgMerkleBlock) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("merkleblock message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgMerkleBlock.Decode", str)
	}

	err := msg.Header.Unserialize(r)
	if err != nil {
		return err
	}

	err = util.ReadElements(r, &msg.Transactions)
	if err != nil {
		return err
	}

	// Read num block locator hashes and limit to max.
	count, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	if count > maxTxPerBlock {
		str := fmt.Sprintf("too many transaction hashes for message "+
			"[count %v, max %v]", count, maxTxPerBlock)
		return messageError("MsgMerkleBlock.Decode", str)
	}

	// Create a contiguous slice of hashes to deserialize into in order to
	// reduce the number of allocations.
	hashes := make([]util.Hash, count)
	msg.Hashes = make([]*util.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := &hashes[i]
		err := util.ReadElements(r, hash)
		if err != nil {
			return err
		}
		msg.AddTxHash(hash)
	}

	msg.Flags, err = util.ReadVarBytes(r, maxFlagsPerMerkleBlock,
		"merkle block flags size")
	return err
}

// Encode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgMerkleBlock) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("merkleblock message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgMerkleBlock.Encode", str)
	}

	// Read num transaction hashes and limit to max.
	numHashes := uint64(len(msg.Hashes))
	if numHashes > maxTxPerBlock {
		str := fmt.Sprintf("too many transaction hashes for message "+
			"[count %v, max %v]", numHashes, maxTxPerBlock)
		return messageError("MsgMerkleBlock.Decode", str)
	}
	numFlagBytes := uint64(len(msg.Flags))
	if numFlagBytes > maxFlagsPerMerkleBlock {
		str := fmt.Sprintf("too many flag bytes for message [count %v, "+
			"max %v]", numFlagBytes, maxFlagsPerMerkleBlock)
		return messageError("MsgMerkleBlock.Decode", str)
	}

	err := msg.Header.Serialize(w)
	if err != nil {
		return err
	}

	err = util.WriteElements(w, msg.Transactions)
	if err != nil {
		return err
	}

	err = util.WriteVarInt(w, numHashes)
	if err != nil {
		return err
	}
	for _, hash := range msg.Hashes {
		err = util.WriteElements(w, hash)
		if err != nil {
			return err
		}
	}

	return util.WriteVarBytes(w, msg.Flags)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgMerkleBlock) Command() string {
	return CmdMerkleBlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgMerkleBlock) MaxPayloadLength(pver uint32) uint64 {
	return MaxBlockPayload
}

// NewMsgMerkleBlock returns a new bitcoin merkleblock message that conforms to
// the Message interface.  See MsgMerkleBlock for details.
func NewMsgMerkleBlock(bh *block.BlockHeader) *MsgMerkleBlock {
	return &MsgMerkleBlock{
		Header:       *bh,
		Transactions: 0,
		Hashes:       make([]*util.Hash, 0),
		Flags:        make([]byte, 0),
	}
}
