package msg

import (
	"github.com/btccom/copernicus/protocol"
	"fmt"
	"io"
	"github.com/btccom/copernicus/utils"
)

const (
	MAX_INVENTORY_MESSAGE = 50000
	MAX_INVENTORY_PAYLOAD = 4 + HEADER_SIZE
)
const (
	INVENTORY_TYPE_ERROR          protocol.InventoryType = 0
	INVENTORY_TYPE_TX             protocol.InventoryType = 1
	INVENTORY_TYPE_BLOCK          protocol.InventoryType = 2
	INVENTORY_TYPE_FILTERED_BLOCK protocol.InventoryType = 3
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
	case INVENTORY_TYPE_BLOCK:
		return "msg_block"
	case INVENTORY_TYPE_ERROR:
		return "error"
	case INVENTORY_TYPE_FILTERED_BLOCK:
		return "msg_filtered_block"
	case INVENTORY_TYPE_TX:
		return "msg_filtered_block"
	}
	return fmt.Sprintf("Unkonwn Inventory type (%d)", uint32(inventoryType))
	
}

func ReadInventoryVector(r io.Reader, pver uint32, iv *InventoryVector) error {
	return protocol.ReadElements(r, &iv.Type, &iv.Hash)
}
func WriteInvVect(w io.Writer, pver uint32, iv *InventoryVector) error {
	return protocol.WriteElements(w, iv.Type, iv.Hash)
}
