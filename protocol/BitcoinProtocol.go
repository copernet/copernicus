package protocol

import (
	"time"
	"fmt"
)

const MAX_MESSAGE_PAYLOAD = 1024 * 1024 * 32
const (
	COPERNICUS                          = "0.0.1"
	BITCOIN_PROTOCOL_VERSION     uint32 = 70012
	PEER_ADDRESS_TIME_VERSION    uint32 = 31402
	MAX_USER_AGENT_LEN                  = 256
	SF_NODE_NETWORK_AS_FULL_NODE        = 1 << iota
	MULTIPLE_ADDRESS_VERSION     uint32 = 209

	REJECT_VERSION  uint32 = 70002
	BIP0037_VERSION uint32 = 70001
)

var LocalUserAgent string

func init() {
	LocalUserAgent = getLocalUserAgent()

}
func getLocalUserAgent() string {
	return fmt.Sprintf("/copernicus%s/", COPERNICUS)
}

type ServiceFlag uint64

// InventoryType represents the allowed types of inventory vectors.  See InvVect.
type InventoryType uint32

// BitcoinNet represents which bitcoin network a msg belongs to.
type BitcoinNet uint32

// BloomUpdateType specifies how the filter is updated when a match is found
type BloomUpdateType uint8

// RejectCode represents a numeric value by which a remote peer indicates
// why a msg was rejected.
type RejectCode uint8

type Uint32Time time.Time
type Int64Time time.Time
