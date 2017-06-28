package msg

import "io"

type GetAddressMessage struct {
}

func (getAddressMessage *GetAddressMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (getAddressMessage *GetAddressMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (getAddressMessage *GetAddressMessage) Command() string {
	return CommandGetAddress
}

func (getAddressMessage *GetAddressMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}

func NewGetAddressMessage() *GetAddressMessage {
	return &GetAddressMessage{}
}
