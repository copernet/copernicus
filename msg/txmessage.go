package msg

import (
	"io"

	"github.com/btcboost/copernicus/model"
)

type TxMessage struct {
	Tx model.Tx
}

func (txMessage *TxMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (txMessage *TxMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (txMessage *TxMessage) Command() string {
	return CommandTx
}

func (txMessage *TxMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
