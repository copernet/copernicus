package msg


import "io"

type MerkleBlockMessage struct {
}

func (merkleBlockMessage *MerkleBlockMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	return nil
}

func (merkleBlockMessage *MerkleBlockMessage) BitcoinParse(reader io.Reader, size uint32) error {
	return nil
}

func (merkleBlockMessage *MerkleBlockMessage) Command() string {
	return CommandFilterAdd
}

func (merkleBlockMessage *MerkleBlockMessage) MaxPayloadLength(size uint32) uint32 {
	return 0
}
