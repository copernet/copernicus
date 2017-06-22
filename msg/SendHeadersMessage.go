package msg

import "io"

type SendHeadersMessage struct {
}

func (sendHeadersMessage *SendHeadersMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (sendHeadersMessage *SendHeadersMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (sendHeadersMessage *SendHeadersMessage) Command() string {
	return COMMAND_FILTER_ADD
}

func (sendHeadersMessage *SendHeadersMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
