package msg

import "io"

type NotFoundMessage struct {
	InventoryList []*InventoryVector
}

func (notFoundMessage *NotFoundMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (notFoundMessage *NotFoundMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (notFoundMessage *NotFoundMessage) Command() string {
	return COMMAND_NOT_FOUND
}

func (notFoundMessage *NotFoundMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
