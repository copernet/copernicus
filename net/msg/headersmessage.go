package msg

import (
	"io"

	"github.com/btcboost/copernicus/core"
)

type HeadersMessage struct {
	Blocks []*core.Block
}

func (headersMessage *HeadersMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (headersMessage *HeadersMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (headersMessage *HeadersMessage) Command() string {
	return CommandHeaders
}

func (headersMessage *HeadersMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
