package msg

import (
	"copernicus/model"
	"io"
)

const (
	HEADER_SIZE        = 80
	ALLOWED_TIME_DRIFT = 2 * 60 * 60
	MAX_BLOCK_SIZE     = 1 * 1000 * 1000
)

type BlockMessage struct {
	Message
	Block *model.Block
	Txs   []*TxMessage
}

func (msg *BlockMessage) AddTx(tx *TxMessage) error {
	msg.Txs = append(msg.Txs, tx)
	return nil
}

func (msg *BlockMessage) ClearTxs() {
	msg.Txs = make([]*TxMessage, 0, 2048)
}

func (blockMessage *BlockMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (blockMessage *BlockMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (blockMessage *BlockMessage) Command() string {
	return COMMAND_BLOCK
}

func (blockMessage *BlockMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
