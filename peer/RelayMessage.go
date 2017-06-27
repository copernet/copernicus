package peer

import "github.com/btccom/copernicus/msg"

type RelayMessage struct {
	InventoryVector *msg.InventoryVector
	Data            interface{}
}
