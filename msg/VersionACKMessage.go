package msg

import "io"

type VersionACKMessage struct {
}

func (versionACKMessage *VersionACKMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (versionACKMessage *VersionACKMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}
func (versionACKMessage *VersionACKMessage) Command() string {
	return CommandVersionAck
}
func (versionACKMessage *VersionACKMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}

func InitVersionACKMessage() *VersionACKMessage {
	return &VersionACKMessage{}
}
