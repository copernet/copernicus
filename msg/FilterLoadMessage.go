package msg

import "io"

type FilterLoadMessage struct {
}

func (filterLoadMessage *FilterLoadMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (filterLoadMessage *FilterLoadMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (filterLoadMessage *FilterLoadMessage) Command() string {
	return COMMAND_FILTER_ADD
}

func (filterLoadMessage *FilterLoadMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
