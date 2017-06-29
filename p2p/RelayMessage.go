package p2p

import "github.com/btccom/copernicus/msg"

type RelayMessage struct {
	InventoryVector *msg.InventoryVector
	Data            interface{}
}
