package p2p

import "github.com/btcboost/copernicus/net/msg"

type RelayMessage struct {
	InventoryVector *msg.InventoryVector
	Data            interface{}
}
