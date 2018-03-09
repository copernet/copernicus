package msg

import "io"

type MempoolMessage struct {
}

func (mempoolMessage *MempoolMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (mempoolMessage *MempoolMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (mempoolMessage *MempoolMessage) Command() string {
	return CommandMempool
}

func (mempoolMessage *MempoolMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
