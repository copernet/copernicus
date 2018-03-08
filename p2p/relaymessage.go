package p2p

import "github.com/btcboost/copernicus/msg"

type RelayMessage struct {
	InventoryVector *msg.InventoryVector
	Data            interface{}
}
