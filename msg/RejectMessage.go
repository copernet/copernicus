package msg

import (
	"fmt"
	"io"

	"github.com/btcboost/copernicus/protocol"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

type RejectCode uint8

const (
	RejectMalformed       RejectCode = 0x01
	RejectInvalid         RejectCode = 0x10
	RejectObsolete        RejectCode = 0X11
	RejectDuplicate       RejectCode = 0x12
	RejectNonstandard     RejectCode = 0x40
	RejectDust            RejectCode = 0x41
	RejectInsufficientFee RejectCode = 0x42
	RejectCheckpoint      RejectCode = 0X43
)

func (code RejectCode) ToString() string {

	switch code {
	case RejectCheckpoint:
		return "reject_check_point"
	case RejectDuplicate:
		return "reject_duplicate"
	case RejectDust:
		return "reject_dust"
	case RejectInsufficientFee:
		return "reject_insufficient_fee"
	case RejectInvalid:
		return "reject_invalid"
	case RejectMalformed:
		return "reject_malformed"
	case RejectObsolete:
		return "reject_obsolete"
	case RejectNonstandard:
		return "reject_nonstandard"
	}
	return fmt.Sprintf("Unkown RejectCode (%d)", uint8(code))
}

type RejectMessage struct {
	Cmd    string
	Code   RejectCode
	Reason string
	Hash   *utils.Hash
}

func (rejectMessage *RejectMessage) BitcoinParse(reader io.Reader, version uint32) error {

	if version < protocol.RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol version ")
		return errors.New(str)
	}
	command, err := utils.ReadVarString(reader)
	if err != nil {
		return err
	}
	rejectMessage.Cmd = command
	err = protocol.ReadElement(reader, &rejectMessage.Code)
	if err != nil {
		return err
	}
	reason, err := utils.ReadVarString(reader)
	rejectMessage.Reason = reason
	if rejectMessage.Cmd == CommandTx || rejectMessage.Cmd == CommandBlock {
		err := protocol.ReadElement(reader, rejectMessage.Hash)
		if err != nil {
			return err
		}

	}
	return nil

}
func (rejectMessage *RejectMessage) BitcoinSerialize(w io.Writer, version uint32) error {
	if version < protocol.RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol version %d", version)
		return errors.New(str)
	}
	err := utils.WriteVarString(w, rejectMessage.Cmd)
	if err != nil {
		return err
	}
	err = protocol.WriteElement(w, rejectMessage.Code)
	if err != nil {
		return err
	}
	err = utils.WriteVarString(w, rejectMessage.Reason)
	if err != nil {
		return err
	}
	if rejectMessage.Cmd == CommandBlock || rejectMessage.Cmd == CommandTx {
		err := protocol.WriteElement(w, rejectMessage.Hash)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rejectMessage *RejectMessage) MaxPayloadLength(pver uint32) uint32 {
	plen := uint32(0)
	if pver >= protocol.RejectVersion {
		plen = protocol.MaxMessagePayload
	}
	return plen
}
func (rejectMessage *RejectMessage) Command() string {
	return CommandReject
}

func NewRejectMessage(command string, code RejectCode, reason string) *RejectMessage {
	rejectMessage := RejectMessage{
		Cmd:    command,
		Code:   code,
		Reason: reason,
	}
	return &rejectMessage
}
