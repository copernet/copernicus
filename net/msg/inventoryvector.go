package msg

import (
	"fmt"
	"io"

	"github.com/btcboost/copernicus/net/protocol"
	"github.com/btcboost/copernicus/utils"
)

const (
	MaxInventoryMessage = 50000
	MaxInventoryPayload = 4 + HeaderSize
)
const (
	InventoryTypeError         protocol.InventoryType = 0
	InventoryTypeTx            protocol.InventoryType = 1
	InventoryTypeBlock         protocol.InventoryType = 2
	InventoryTypeFilteredBlock protocol.InventoryType = 3
)

type InventoryVector struct {
	Type protocol.InventoryType
	Hash *utils.Hash
}

func NewInventoryVecror(typ protocol.InventoryType, hash *utils.Hash) *InventoryVector {
	inventoryVector := InventoryVector{Type: typ, Hash: hash}
	return &inventoryVector
}

func InventoryTypeToString(inventoryType protocol.InventoryType) string {
	switch inventoryType {
	case InventoryTypeBlock:
		return "msg_block"
	case InventoryTypeError:
		return "error"
	case InventoryTypeFilteredBlock:
		return "msg_filtered_block"
	case InventoryTypeTx:
		return "msg_filtered_block"
	}
	return fmt.Sprintf("Unknown Inventory type (%d)", uint32(inventoryType))

}

func ReadInventoryVector(r io.Reader, iv *InventoryVector) error {
	return protocol.ReadElements(r, &iv.Type, &iv.Hash)
}
func WriteInvVect(w io.Writer, iv *InventoryVector) error {
	return protocol.WriteElements(w, iv.Type, iv.Hash)
}
