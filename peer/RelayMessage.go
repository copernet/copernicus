package peer

import "copernicus/msg"

type RelayMessage struct {
	InventoryVector *msg.InventoryVector
	Data            interface{}
}
