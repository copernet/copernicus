package p2p

import "github.com/btcboost/copernicus/net/msg"

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
