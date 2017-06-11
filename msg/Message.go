package msg

import (
	"copernicus/protocol"
	"io"
)

const (
	// MessageHeaderSize is the number of bytes in a bitcoin msg header.
	// Bitcoin network (magic) 4 bytes + command 12 bytes + payload length 4 bytes +
	// checksum 4 bytes.
	MESSAGE_HEADER_SIZE = 24
)

type Message interface {
	BitcoinParse(reader io.Reader, size uint32) error
	BitcoinSerialize(writer io.Writer, size uint32) error
	MaxPayloadLength(version uint32) uint32
}

type messageHeader struct {
	Net      protocol.BitcoinNet // 4 bytes
	Command  string              // 12 bytes
	Length   uint32              // 4 bytes
	Checksum [4]byte             // 4 bytes
}
