package msg

import (
	"fmt"
	"io"
	"github.com/btccom/copernicus/utils"
	"errors"
)

const DefaultInventoryListAlloc = 1000

type InventoryMessage struct {
	InventoryList []*InventoryVector
}

func (message *InventoryMessage) AddInventoryVector(iv *InventoryVector) error {
	if len(message.InventoryList)+1 > MaxInventoryMessage {
		str := fmt.Sprintf("too many invvect in message max %v", MaxInventoryMessage)
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
	if count > MaxInventoryMessage {
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
	if count > MaxInventoryMessage {
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
	return CommandInv
}
func (message *InventoryMessage) MaxPayloadLength(pver uint32) uint32 {
	return MaxVarIntPayload + (MaxInventoryMessage + MaxInventoryPayload)
}
func NewMessageInventory() *InventoryMessage {
	inventoryMessage := InventoryMessage{InventoryList: make([]*InventoryVector, 0, DefaultInventoryListAlloc)}
	return &inventoryMessage
}

func NewInventoryMessageSizeHint(sizeHint uint) *InventoryMessage {
	if sizeHint > MaxInventoryMessage {
		sizeHint = MaxInventoryMessage
	}
	inventoryMessage := InventoryMessage{InventoryList: make([]*InventoryVector, 0, sizeHint)}
	return &inventoryMessage
}
