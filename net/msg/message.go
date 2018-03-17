package msg

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/btcboost/copernicus/utils"
)

const (
	CommandSize        = 12
	MaxRejectReasonLen = 250
	LockTimeThreshold  = 5E8 // Tue Nov 5 00:53:20 1985 UTC
	SafeChars          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890 .,;_/:?@"
)

const (
	CommandVersion     = "version"
	CommandVersionAck  = "verack"
	CommandGetAddress  = "getaddr"
	CommandAddress     = "addr"
	CommandGetBlocks   = "getblocks"
	CommandInv         = "inv"
	CommandGetData     = "getdata"
	CommandNotFound    = "notfound"
	CommandBlock       = "block"
	CommandTx          = "tx"
	CommandGetHeaders  = "getheaders"
	CommandHeaders     = "headers"
	CommandPing        = "ping"
	CommandPong        = "pong"
	CommandAlert       = "alert"
	CommandMempool     = "mempool"
	CommandFilterAdd   = "filteradd"
	CommandFilterClear = "filterclear"
	CommandFilterLoad  = "filterrload"
	CommandMerkleBlock = "merkleblock"
	CommandReject      = "reject"
	CommandSendHeaders = "sendheaders"
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
		case InventoryTypeError:
			return fmt.Sprintf("error %v", iv.Hash)
		case InventoryTypeBlock:
			return fmt.Sprintf("block %v", iv.Hash)
		case InventoryTypeTx:
			return fmt.Sprintf("unknown %d ,%v", uint32(iv.Type), iv.Hash)

		}
	}
	return fmt.Sprintf("size %d", invLen)
}
func LocatorSummary(locator []*utils.Hash, stopHash *utils.Hash) string {
	if len(locator) > 0 {
		return fmt.Sprintf("locator %v , stop %v", locator[0], stopHash)
	}
	return fmt.Sprintf("no locator , stop %v", stopHash)
}

func MessageSummary(msg Message) string {
	switch msgType := msg.(type) {
	case *VersionMessage:
		return fmt.Sprintf("agent %s, pver %d, block %d",
			msgType.UserAgent, msgType.ProtocolVersion, msgType.LastBlock)
	case *AddressMessage:
		return fmt.Sprintf("%d address", len(msgType.AddressList))
	case *TxMessage:
		return fmt.Sprintf("tx hash %s,%d inputs,%d outputs ,lock %s", msgType.Tx.Hash,
			len(msgType.Tx.Ins), len(msgType.Tx.Outs), LockTimeToString(msgType.Tx.LockTime))
	case *BlockMessage:
		return fmt.Sprintf("hash %s, ver %d, %d txs ,%d", msgType.Block.Hash,
			msgType.Block.BlockHeader.Version, len(msgType.Block.Txs), msgType.Block.BlockHeader.Time)
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
		rejCommand := SanitizeString(msgType.Command(), CommandSize)
		rejReason := SanitizeString(msgType.Reason, MaxRejectReasonLen)
		summary := fmt.Sprintf("command %v, code %v,reason %v", rejCommand, msgType.Code, rejReason)
		if rejCommand == CommandTx || rejCommand == CommandBlock {
			summary = fmt.Sprintf("%s, hash %v", summary, msgType.Hash)
		}
		return summary
	}
	return ""
}

func SanitizeString(str string, maxLength uint) string {
	str = strings.Map(func(r rune) rune {
		if strings.ContainsRune(SafeChars, r) {
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
	if lockTime < LockTimeThreshold {
		return fmt.Sprintf("height %d", lockTime)
	}
	return time.Unix(int64(lockTime), 0).String()
}

//todo add other message
func makeEmptyMessage(command string) (Message, error) {
	var message Message
	switch command {
	case CommandVersion:
		message = &VersionMessage{}
	case CommandAddress:
		message = &AddressMessage{}
		//todo getBlocks and getBlock
	case CommandGetBlocks:
		message = &GetBlocksMessage{}
	case CommandGetHeaders:
		message = &GetHeadersMessage{}
	case CommandReject:
		message = &RejectMessage{}

	default:
		return nil, fmt.Errorf("unknown command %s", command)

	}
	return message, nil
}
