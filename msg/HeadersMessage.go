package msg

import (
	"copernicus/model"
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
	return COMMAND_HEADERS
}

func (headersMessage *HeadersMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
