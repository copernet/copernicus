package msg

import "io"

type AlertMessage struct {
}

func (alertMessage *AlertMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (alertMessage *AlertMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (alertMessage *AlertMessage) Command() string {
	return COMMAND_ALERT
}

func (alertMessage *AlertMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
