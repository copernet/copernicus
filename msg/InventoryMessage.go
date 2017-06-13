package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"copernicus/utils"
)

const DEFAULT_INVENTORY_LIST_ALLOC = 1000

type InventoryMessage struct {
	InventoryList []*InventoryVector
}

func (message *InventoryMessage) AddInventoryVector(iv *InventoryVector) error {
	if len(message.InventoryList)+1 > MAX_INVENTORY_MESSAGE {
		str := fmt.Sprintf("too many invvect in message max %v", MAX_INVENTORY_MESSAGE)
		return errors.New(str)
	}
	message.InventoryList = append(message.InventoryList, iv)
	return nil
}

func (message *InventoryMessage) BitcoinParse(reader io.Reader, pver uint32) error {
	count, err := utils.ReadVarInt(reader, pver)
	if err != nil {
		return err
	}
	if count > MAX_INVENTORY_MESSAGE {
		str := fmt.Sprintf("too many inventory in message %v", count)
		return errors.New(str)
	}
	inventoryList := make([]InventoryVector, count)
	message.InventoryList = make([]*InventoryVector, 0, count)
	for i := uint64(0); i < count; i++ {
		iv := &inventoryList[i]
		err := ReadInventoryVector(reader, pver, iv)
		if err != nil {
			return err
		}
		message.AddInventoryVector(iv)

	}
	return nil

}
func (message *InventoryMessage) BitcoinSerialize(w io.Writer, pver uint32) error {
	count := len(message.InventoryList)
	if count > MAX_INVENTORY_MESSAGE {
		str := fmt.Sprintf("too many inventory in message %v", count)
		return errors.New(str)
	}
	err := utils.WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}
	for _, iv := range message.InventoryList {
		err := WriteInvVect(w, pver, iv)
		if err != nil {
			return err
		}
	}
	return nil
}

func (message *InventoryMessage) Command() string {
	return COMMAND_INV
}
func (message *InventoryMessage) MaxPayloadLength(pver uint32) uint32 {
	return MAX_VAR_INT_PAYLOAD + (MAX_INVENTORY_MESSAGE + MAX_INVENTORY_PAYLOAD)
}
func InitMessageInventory() *InventoryMessage {
	inventoryMessage := InventoryMessage{InventoryList: make([]*InventoryVector, 0, DEFAULT_INVENTORY_LIST_ALLOC)}
	return &inventoryMessage
}

func InitMessageInvSizeHine(sizeHint uint) *InventoryMessage {
	if sizeHint > MAX_INVENTORY_MESSAGE {
		sizeHint = MAX_INVENTORY_MESSAGE
	}
	inventoryMessage := InventoryMessage{InventoryList: make([]*InventoryVector, 0, sizeHint)}
	return &inventoryMessage
}
