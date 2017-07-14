package msg

import (
	"github.com/btcboost/copernicus/model"
	"io"
)

type HeadersMessage struct {
	Blocks []*model.Block
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
