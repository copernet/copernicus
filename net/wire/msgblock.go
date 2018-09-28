// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"

	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/util"
)

// MaxBlocksPerMsg is the maximum number of blocks allowed per message.
const MaxBlocksPerMsg = 500

// MaxBlockPayload is the maximum bytes a block message can be in bytes.
const MaxBlockPayload = 32 * util.OneMegaByte

const maxTxPerBlock = (MaxBlockPayload / minTxPayload) + 1

type MsgBlock block.Block

func (msg *MsgBlock) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return (*block.Block)(msg).Unserialize(r)
}

func (msg *MsgBlock) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return (*block.Block)(msg).Serialize(w)
}

func (msg *MsgBlock) Command() string {
	return CmdBlock
}

func (msg *MsgBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}
