package msg

import "io"

type GetDataMessage struct {
}

func (getDataMessage *GetDataMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (getDataMessage *GetDataMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (getDataMessage *GetDataMessage) Command() string {
	return COMMAND_GET_DATA
}


func (getDataMessage *GetDataMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
