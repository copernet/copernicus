package msg

import "io"

type VersionACKMessage struct {
}

func (versionACKMessage *VersionACKMessage) BitcoinParse(reader io.Reader, pver uint32) error {
	return nil
}

func (versionACKMessage *VersionACKMessage) BitcoinSerialize(w io.Writer, pver uint32) error {
	return nil
}
func (versionACKMessage *VersionACKMessage) Command() string {
	return COMMAND_VERSION_ACK
}
func (versionACKMessage *VersionACKMessage) MaxPayloadLength(pver uint32) uint32 {
	return 0
}
func InitVersionACKMessage() *VersionACKMessage {
	return &VersionACKMessage{}
}
