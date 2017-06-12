package msg

import (
	"io"
	"fmt"
	"time"
	"copernicus/crypto"
	"strings"
)

const (
	// MessageHeaderSize is the number of bytes in a bitcoin msg header.
	// Bitcoin network (magic) 4 bytes + command 12 bytes + payload length 4 bytes +
	// checksum 4 bytes.
	COMMAND_SIZE          = 12
	MAX_REJECT_REASON_LEN = 250

	LOCK_TIME_THRESHOLD = 5E8 // Tue Nov 5 00:53:20 1985 UTC
	SAFE_CHARS          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890 .,;_/:?@"
)

const (
	COMMAND_VERSION      = "version"
	COMMAND_VERSION_ACK  = "verack"
	COMMAND_GET_ADDRESS  = "getaddr"
	COMMMAND_ADDRESS     = "addr"
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
	Command() string
}

func InventorySummary(invList []*InventoryVector) string {
	invLen := len(invList)
	if invLen == 0 {
		return "empty"
	}
	if invLen == 1 {
		iv := invList[0]
		switch iv.Type {
		case INVENTORY_TYPE_ERROR:
			return fmt.Sprintf("error %s", iv.Hash)
		case INVENTORY_TYPE_BLOCK:
			return fmt.Sprintf("block %s", iv.Hash)
		case INVENTORY_TYPE_TX:
			return fmt.Sprintf("unkonwn %d ,%s", uint32(iv.Type), iv.Hash)

		}
	}
	return fmt.Sprintf("size %d", invLen)
}
func LocatorSummary(locator []*crypto.Hash, stopHash *crypto.Hash) string {
	if len(locator) > 0 {
		return fmt.Sprintf("locator %s , stop %s", locator[0], stopHash)
	}
	return fmt.Sprintf("no locator , stop %s", stopHash)
}

func MessageSummary(msg Message) string {
	switch msgType := msg.(type) {
	case *VersionMessage:
		return fmt.Sprintf("agent %s, pver %d, block %d",
			msgType.UserAgent, msgType.ProtocolVersion, msgType.LastBlock)
	case *PeerAddressMessage:
		return fmt.Sprintf("%d address", len(msgType.AddressList))
	case *TxMessage:
		return fmt.Sprintf("tx hash %s,%d inputs,%d outputs ,lock %s", msgType.Tx.Hash,
			len(msgType.Tx.Ins), len(msgType.Tx.Outs), LockTimeToString(msgType.Tx.LockTime))
	case *BlockMessage:
		return fmt.Sprintf("hash %s, ver %d, %d txs ,%s", msgType.Block.Hash,
			msgType.Block.Version, len(msgType.Block.Transactions), msgType.Block.BlockTime)
	case *InventoryMessage:
		return InventorySummary(msgType.InventoryList)
	case *NotFoundMessage:
		return InventorySummary(msgType.InventoryList)
	case *GetBlocksMessage:
		return LocatorSummary(msgType.BlockHashes, msgType.HashStop)
	case *GetHeadersMessage:
		return LocatorSummary(msgType.BlockHashes, msgType.HashStop)
	case *HeadersMessage:
		return fmt.Sprintf("num %d", len(msgType.Blocks))
	case *RejectMessage:
		rejCommand := SanitizeString(msgType.Command(), COMMAND_SIZE)
		rejReason := SanitizeString(msgType.Reason, MAX_REJECT_REASON_LEN)
		summary := fmt.Sprintf("command %v, code %v,reason %v", rejCommand, msgType.Code, rejReason)
		if rejCommand == COMMAND_TX || rejCommand == COMMAND_BLOCK {
			summary = fmt.Sprintf("%s, hash %v", summary, msgType.Hash)
		}
		return summary
	}
	return ""
}

func SanitizeString(str string, maxLength uint) string {
	str = strings.Map(func(r rune) rune {
		if strings.IndexRune(SAFE_CHARS, r) >= 0 {
			return r
		}
		return -1
	}, str)
	if maxLength > 0 && uint(len(str)) > maxLength {
		str = str[:maxLength]
		str = str + "..."
	}
	return str

}
func LockTimeToString(lockTime uint32) string {
	if lockTime < LOCK_TIME_THRESHOLD {
		return fmt.Sprintf("height %d", lockTime)
	}
	return time.Unix(int64(lockTime), 0).String()
}

//todo add other message
func makeEmptyMessage(command string) (Message, error) {
	var message Message
	switch command {
	case COMMAND_VERSION:
		message = &VersionMessage{}
	case COMMMAND_ADDRESS:
		message = &PeerAddressMessage{}
		//todo getBlocks and getBlock
	case COMMAND_GET_BLOCKS:
		message = &GetBlocksMessage{}
	case COMMAND_GET_HEADERS:
		message = &GetHeadersMessage{}
	case COMMAND_REJECT:
		message = &RejectMessage{}

	default:
		return nil, fmt.Errorf("unkonwn command %s", command)

	}
	return message, nil
}
