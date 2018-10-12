// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"github.com/copernet/copernicus/errcode"
	"io"

	"github.com/copernet/copernicus/util"
)

// MsgReject implements the Message interface and represents a bitcoin reject
// message.
//
// This message was not added until protocol version RejectVersion.
type MsgReject struct {
	// Cmd is the command for the message which was rejected such as
	// as CmdBlock or CmdTx.  This can be obtained from the Command function
	// of a Message.
	Cmd string

	// RejectCode is a code indicating why the command was rejected.  It
	// is encoded as a uint8 on the wire.
	Code errcode.RejectCode

	// Reason is a human-readable string with specific details (over and
	// above the reject code) about why the command was rejected.
	Reason string

	// Hash identifies a specific block or transaction that was rejected
	// and therefore only applies the MsgBlock and MsgTx messages.
	Hash util.Hash
}

// Decode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgReject) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgReject.Decode", str)
	}

	// Command that was rejected.
	cmd, err := util.ReadVarString(r)
	if err != nil {
		return err
	}
	msg.Cmd = cmd

	// Code indicating why the command was rejected.
	err = util.ReadElements(r, &msg.Code)
	if err != nil {
		return err
	}

	// Human readable string with specific details (over and above the
	// reject code above) about why the command was rejected.
	reason, err := util.ReadVarString(r)
	if err != nil {
		return err
	}
	msg.Reason = reason

	// CmdBlock and CmdTx messages have an additional hash field that
	// identifies the specific block or transaction.
	if msg.Cmd == CmdBlock || msg.Cmd == CmdTx {
		err := util.ReadElements(r, &msg.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

// Encode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgReject) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgReject.Encode", str)
	}

	// Command that was rejected.
	err := util.WriteVarString(w, msg.Cmd)
	if err != nil {
		return err
	}

	// Code indicating why the command was rejected.
	err = util.WriteElements(w, msg.Code)
	if err != nil {
		return err
	}

	// Human readable string with specific details (over and above the
	// reject code above) about why the command was rejected.
	err = util.WriteVarString(w, msg.Reason)
	if err != nil {
		return err
	}

	// CmdBlock and CmdTx messages have an additional hash field that
	// identifies the specific block or transaction.
	if msg.Cmd == CmdBlock || msg.Cmd == CmdTx {
		err := util.WriteElements(w, &msg.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgReject) Command() string {
	return CmdReject
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgReject) MaxPayloadLength(pver uint32) uint64 {
	plen := uint64(0)
	// The reject message did not exist before protocol version
	// RejectVersion.
	if pver >= RejectVersion {
		// Unfortunately the bitcoin protocol does not enforce a sane
		// limit on the length of the reason, so the max payload is the
		// overall maximum message payload.
		plen = MaxMessagePayload
	}

	return plen
}

// NewMsgReject returns a new bitcoin reject message that conforms to the
// Message interface.  See MsgReject for details.
func NewMsgReject(command string, code errcode.RejectCode, reason string) *MsgReject {
	return &MsgReject{
		Cmd:    command,
		Code:   code,
		Reason: reason,
	}
}
