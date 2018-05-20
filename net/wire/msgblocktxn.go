package wire

import (
	"fmt"
	"io"

	"github.com/btcboost/copernicus/util"
)

type MsgBlockTxn struct {
	BlockHash util.Hash
	Txn       []*MsgTx
}

func NewMsgBlockTxn(blockHash *util.Hash, indexSize int) *MsgBlockTxn {
	return &MsgBlockTxn{
		BlockHash: *blockHash,
		Txn:       make([]*MsgTx, indexSize),
	}
}

func (msg *MsgBlockTxn) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("blocktxn message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgBlockTxn.BtcEncode", str)
	}
	if err := util.WriteElements(w, msg.BlockHash); err != nil {
		return err
	}
	if err := util.WriteVarInt(w, uint64(len(msg.Txn))); err != nil {
		return err
	}
	for i := 0; i < len(msg.Txn); i++ {
		if err := msg.Txn[i].BtcEncode(w, pver, enc); err != nil {
			return err
		}
	}
	return nil
}

func (msg *MsgBlockTxn) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("blocktxn message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgBlockTxn.BtcDecode", str)
	}

	if err := util.ReadElements(r, &msg.BlockHash); err != nil {
		return err
	}
	txnSize, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	txns := make([]MsgTx, int(txnSize))
	for i := 0; i < len(txns); i++ {
		if err := txns[i].BtcDecode(r, pver, enc); err != nil {
			return err
		}
		msg.Txn[i] = &txns[i]
	}
	return nil
}

func (msg *MsgBlockTxn) Command() string {
	return CmdBlockTxn
}

func (msg *MsgBlockTxn) MaxPayloadLength(pver uint32) uint32 {
	return 32 + 3 + uint32(len(msg.Txn))*MaxBlockPayload
}
