package protocol

import (
	"fmt"
	"time"
)

const MaxMessagePayload = 1024 * 1024 * 32
const (
	Copernicus                    = "0.0.1"
	BitcoinProtocolVersion uint32 = 70012
	PeerAddressTimeVersion uint32 = 31402
	MaxUserAgentLen               = 256
	MultipleAddressVersion uint32 = 209
	MaxProtocolVersion     uint32 = 70012
	RejectVersion          uint32 = 70002
	Bip0037Version         uint32 = 70001
	Bip0031Version         uint32 = 60000
	Bip0111Version         uint32 = 70011

	MaxKnownInventory = 1000
)
const (
	SFNodeNetworkAsFullNode = 1 << iota
	SFNodeGetUtxo
	SFNodeBloomFilter
)

// InventoryType represents the allowed types of inventory vectors.  See InvVect.
type InventoryType uint32

// BloomUpdateType specifies how the filter is updated when a match is found
type BloomUpdateType uint8

// RejectCode represents a numeric value by which a remote p2p indicates
// why a msg was rejected.
type RejectCode uint8

type Uint32Time time.Time
type Int64Time time.Time

var LocalUserAgent string

func init() {
	LocalUserAgent = getLocalUserAgent()

}
func getLocalUserAgent() string {
	return fmt.Sprintf("/Copernicus%s/", Copernicus)
}
