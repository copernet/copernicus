package wire

/*
import (
	"fmt"
	"io"

	"github.com/copernet/copernicus/util"
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

func (msg *MsgBlockTxn) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("blocktxn message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgBlockTxn.Encode", str)
	}
	if err := util.WriteElements(w, msg.BlockHash); err != nil {
		return err
	}
	if err := util.WriteVarInt(w, uint64(len(msg.Txn))); err != nil {
		return err
	}
	for i := 0; i < len(msg.Txn); i++ {
		if err := msg.Txn[i].Encode(w, pver, enc); err != nil {
			return err
		}
	}
	return nil
}

func (msg *MsgBlockTxn) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("blocktxn message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgBlockTxn.Decode", str)
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
		if err := txns[i].Decode(r, pver, enc); err != nil {
			return err
		}
		msg.Txn[i] = &txns[i]
	}
	return nil
}

func (msg *MsgBlockTxn) Command() string {
	return CmdBlockTxn
}

func (msg *MsgBlockTxn) MaxPayloadLength(pver uint32) uint64 {
	return 32 + 3 + uint64(len(msg.Txn))*MaxBlockPayload
}
*/
