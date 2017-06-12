package msg

import (
	"fmt"
	"io"
	"copernicus/protocol"
	"github.com/pkg/errors"
	"copernicus/utils"
	"copernicus/crypto"
)

type RejectCode uint8

const (
	REJECT_MALFORMED        RejectCode = 0x01
	REJECT_INVALID          RejectCode = 0x10
	REJECT_OBSOLETE         RejectCode = 0X11
	REJECT_DUPLICATE        RejectCode = 0x12
	REJECT_NONSTANDARD      RejectCode = 0x40
	REJECT_DUST             RejectCode = 0x41
	REJECT_INSUFFICIENT_FEE RejectCode = 0x42
	REJECT_CHECKPOINT       RejectCode = 0X43
)

func (code RejectCode) ToString() string {

	switch code {
	case REJECT_CHECKPOINT:
		return "reject_check_point"
	case REJECT_DUPLICATE:
		return "reject_duplicate"
	case REJECT_DUST:
		return "reject_dust"
	case REJECT_INSUFFICIENT_FEE:
		return "reject_insufficient_fee"
	case REJECT_INVALID:
		return "reject_invalid"
	case REJECT_MALFORMED:
		return "reject_malformed"
	case REJECT_OBSOLETE:
		return "reject_obsolete"
	case REJECT_NONSTANDARD:
		return "reject_nonstandard"
	}
	return fmt.Sprintf("Unkown RejectCode (%d)", uint8(code))
}

type RejectMessage struct {
	Command string
	Code    RejectCode
	Reason  string
	Hash    *crypto.Hash
}

func (rejectMessage *RejectMessage) BitcoinParse(reader io.Reader, pver uint32) error {

	if pver < protocol.REJECT_VERSION {
		str := fmt.Sprintf("reject message invalid for protocol version %d", pver)
		return errors.New(str)
	}
	command, err := utils.ReadVarString(reader, pver)
	if err != nil {
		return err
	}
	rejectMessage.Command = command
	err = utils.ReadElement(reader, &rejectMessage.Code)
	if err != nil {
		return err
	}
	reason, err := utils.ReadVarString(reader, pver)
	rejectMessage.Reason = reason
	if rejectMessage.Command == COMMAND_TX || rejectMessage.Command == COMMAND_BLOCK {
		err := utils.ReadElement(reader, rejectMessage.Hash)
		if err != nil {
			return err
		}

	}
	return nil

}
func (rejectMessage *RejectMessage) BitcoinSerialize(w io.Writer, pver uint32) error {
	if pver < protocol.REJECT_VERSION {
		str := fmt.Sprintf("reject message invalid for protocol version %d", pver)
		return errors.New(str)
	}
	err := utils.WriteVarString(w, pver, rejectMessage.Command)
	if err != nil {
		return err
	}
	err = utils.WriteElement(w, rejectMessage.Code)
	if err != nil {
		return err
	}
	err = utils.WriteVarString(w, pver, rejectMessage.Reason)
	if err != nil {
		return err
	}
	if rejectMessage.Command == COMMAND_BLOCK || rejectMessage.Command == COMMAND_TX {
		err := utils.WriteElement(w, rejectMessage.Hash)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rejectMessage *RejectMessage) MaxPayloadLength(pver uint32) uint32 {
	plen := uint32(0)
	if pver >= protocol.REJECT_VERSION {
		plen = protocol.MAX_MESSAGE_PAYLOAD
	}
	return plen
}

func NewRejectMessage(command string, code RejectCode, reason string) *RejectMessage {
	rejectMessage := RejectMessage{
		Command: command,
		Code:    code,
		Reason:  reason,
	}
	return &rejectMessage
}
