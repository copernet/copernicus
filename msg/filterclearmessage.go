package msg

import "io"

type FilterClearMessage struct {
}

func (filterClearMessage *FilterClearMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (filterClearMessage *FilterClearMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (filterClearMessage *FilterClearMessage) Command() string {
	return CommandFilterAdd
}

func (filterClearMessage *FilterClearMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
