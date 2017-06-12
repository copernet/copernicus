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

const (
	COMMAND_VERSION      = "version"
	COMMAND_VERSION_ACK  = "verack"
	COMMAND_GET_ADDRESS  = "getaddr"
	COMMAND_GET_BLOCKS   = "getblocks"
	COMMAND_INV          = "inv"
	COMMAND_GET_DATA     = "getdata"
	COMMAND_NOT_FOUND    = "notfound"
	COMMAND_BLOCK        = "block"
	COMMAND_TX           = "tx"
	COMMAND_GET_HEADERS  = "getheaders"
	COMMAND_HEADERS      = "headers"
	COMMAND_PING         = "ping"
	COMMAND_PONG         = "pong"
	COMMAND_ALERT        = "alert"
	COMMAND_MEMPOOL      = "mempool"
	COMMAND_FILTER_ADD   = "filteradd"
	COMMAND_FILTER_CLEAR = "filterclear"
	COMMAND_FILTER_LOAD  = "filterrload"
	COMMAND_MERKLE_BLOCK = "merkleblock"
	COMMAND_REJECT       = "reject"
	COMMAND_SEND_HEADERS = "sendheaders"
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
