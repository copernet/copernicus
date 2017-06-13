package peer

import "copernicus/msg"

type StallControlCommand uint8

const (
	SccSendMessage    StallControlCommand = iota
	SccReceiveMessage
	SccHandlerStart
	SccHandlerDone
)

type StallControlMessage struct {
	Command StallControlCommand
	Message msg.Message
}

