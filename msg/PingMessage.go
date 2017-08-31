package msg

import (
	"io"

	"github.com/btcboost/copernicus/protocol"
)

type PingMessage struct {
	Nonce uint64
}

func (pingMessage *PingMessage) BitcoinParse(reader io.Reader, pver uint32) error {
	if pver > protocol.Bip0031Version {
		err := protocol.ReadElement(reader, &pingMessage.Nonce)
		if err != nil {
			return err
		}
	}
	return nil
}
func (pingMessage *PingMessage) BitcoinSerialize(w io.Writer, pver uint32) error {
	if pver > protocol.Bip0031Version {
		err := protocol.WriteElements(w, pingMessage.Nonce)
		if err != nil {
			return err
		}
	}
	return nil
}
func (pingMessage *PingMessage) Command() string {
	return CommandPing
}
func (pingMessage *PingMessage) MaxPayloadLength(pver uint32) uint32 {
	payloadLength := uint32(0)
	if pver > protocol.Bip0031Version {
		payloadLength += 8
	}
	return payloadLength

}
func InitPingMessage(nonce uint64) *PingMessage {
	pingMessage := PingMessage{Nonce: nonce}
	return &pingMessage
}
