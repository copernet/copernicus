package peer

import "github.com/btccom/copernicus/msg"

type StallControlCommand uint8

const (
	SccSendMessage StallControlCommand = iota
	SccReceiveMessage
	SccHandlerStart
	SccHandlerDone
)

type StallControlMessage struct {
	Command StallControlCommand
	Message msg.Message
}
