package msg

import (
	"io"
	"github.com/btccom/copernicus/protocol"
	"fmt"
	"github.com/pkg/errors"
)

type PongMessage struct {
	Nonce uint64
}

func (pongMessage *PongMessage) BitcoinParse(reader io.Reader, pver uint32) error {
	if pver <= protocol.Bip0031Version {
		str := fmt.Sprintf("pong message invalid for protocol version %d", pver)
		return errors.New(str)
	}
	err := protocol.ReadElement(reader, &pongMessage.Nonce)
	return err
}

func (pongMessage *PongMessage) BitcoinSerialize(w io.Writer, pver uint32) error {
	if pver <= protocol.Bip0031Version {
		str := fmt.Sprintf("pong message invalid for protocol version %d", pver)
		return errors.New(str)
		
	}
	err := protocol.WriteElement(w, pongMessage.Nonce)
	return err
}

func (pongMessage *PongMessage) Command() string {
	return CommandPong
}

func (pongMessage *PongMessage) MaxPayloadLength(pver uint32) uint32 {
	payloadLength := uint32(0)
	if pver > protocol.Bip0031Version {
		payloadLength += 8
	}
	return payloadLength
}
func InitPondMessage(nonce uint64) *PongMessage {
	pongMessage := PongMessage{Nonce: nonce}
	return &pongMessage
}
