// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"

	"github.com/btcboost/copernicus/model/tx"
)

const minTxPayload = 10

type MsgTx tx.Tx

func (msg *MsgTx) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return (*tx.Tx)(msg).Unserialize(r)
}

func (msg *MsgTx) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return (*tx.Tx)(msg).Serialize(w)
}

func (msg *MsgTx) Command() string {
	return CmdTx
}

func (msg *MsgTx) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}
