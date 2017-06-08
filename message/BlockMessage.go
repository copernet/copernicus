package message

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
func (msg *BlockMessage) BitcoinParse(reader io.Reader, size uint32) {

}
