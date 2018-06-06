// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"io"

	"github.com/copernet/copernicus/util"
)

const (
	// MaxInvPerMsg is the maximum number of inventory vectors that can be in a
	// single bitcoin inv message.
	MaxInvPerMsg = 50000

	// Maximum payload size for an inventory vector.
	maxInvVectPayload = 4 + util.Hash256Size

	// InvWitnessFlag denotes that the inventory vector type is requesting,
	// or sending a version which includes witness data.
	InvWitnessFlag = 1 << 30
)

// InvType represents the allowed types of inventory vectors.  See InvVect.
type InvType uint32

// These constants define the various supported inventory vector types.
const (
	InvTypeError         InvType = 0
	InvTypeTx            InvType = 1
	InvTypeBlock         InvType = 2
	InvTypeFilteredBlock InvType = 3
	//bip 152
	InvTypeCompatedBlock InvType = 4
	MsgExtFlag           InvType = 1 << 29
	MsgTypeMask          InvType = 0xffffffff >> 3
	//Extension block
	MsgExtTx    InvType = InvTypeTx | MsgExtFlag
	MsgExtBlock InvType = InvTypeBlock | MsgExtFlag
)

// Map of service flags back to their constant names for pretty printing.
var ivStrings = map[InvType]string{
	InvTypeError:         "ERROR",
	InvTypeTx:            "MSG_TX",
	InvTypeBlock:         "MSG_BLOCK",
	InvTypeFilteredBlock: "MSG_FILTERED_BLOCK",
	InvTypeCompatedBlock: "MSG_COMPATED_BLOCK",
	MsgExtTx:             "MSG_EXT_TX",
	MsgExtBlock:          "MSG_EXT_BLOCK",
}

// String returns the InvType in human-readable form.
func (invtype InvType) String() string {
	if s, ok := ivStrings[invtype]; ok {
		return s
	}

	return fmt.Sprintf("Unknown InvType (%d)", uint32(invtype))
}

// InvVect defines a bitcoin inventory vector which is used to describe data,
// as specified by the Type field, that a peer wants, has, or does not have to
// another peer.
type InvVect struct {
	Type InvType   // Type of data
	Hash util.Hash // Hash of the data
}

// NewInvVect returns a new InvVect using the provided type and hash.
func NewInvVect(typ InvType, hash *util.Hash) *InvVect {
	return &InvVect{
		Type: typ,
		Hash: *hash,
	}
}

// readInvVect reads an encoded InvVect from r depending on the protocol
// version.
func readInvVect(r io.Reader, pver uint32, iv *InvVect) error {
	return util.ReadElements(r, &iv.Type, &iv.Hash)
}

// writeInvVect serializes an InvVect to w depending on the protocol version.
func writeInvVect(w io.Writer, pver uint32, iv *InvVect) error {
	return util.WriteElements(w, iv.Type, &iv.Hash)
}
