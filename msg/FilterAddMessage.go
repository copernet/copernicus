package msg

import "io"

type FilterAddMessage struct {
}

func (filterAddMessage *FilterAddMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (filterAddMessage *FilterAddMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (filterAddMessage *FilterAddMessage) Command() string {
	return COMMAND_FILTER_ADD
}

func (filterAddMessage *FilterAddMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
