package wire

import (
	"fmt"
	"io"

	"github.com/btcboost/copernicus/util"
)

type MsgSendCmpct struct {
	AnnounceUsingCmpctBlock bool
	CmpctBlockVersion       uint64
}

func (msg *MsgSendCmpct) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("sendcmpct message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgSendCmpt.BtcDecode", str)
	}
	return util.ReadElements(r, &msg.AnnounceUsingCmpctBlock, &msg.CmpctBlockVersion)
}

func (msg *MsgSendCmpct) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("sendcmpct message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgSendCmpct.BtcEncode", str)
	}
	return util.WriteElements(w, msg.AnnounceUsingCmpctBlock, msg.CmpctBlockVersion)
}

func (msg *MsgSendCmpct) Command() string {
	return CmdSendCmpct
}

func (msg *MsgSendCmpct) MaxPayloadLength(pver uint32) uint32 {
	return 9
}

func NewMsgSendCmpct(announce bool, version uint64) *MsgSendCmpct {
	return &MsgSendCmpct{
		AnnounceUsingCmpctBlock: announce,
		CmpctBlockVersion:       version,
	}
}
