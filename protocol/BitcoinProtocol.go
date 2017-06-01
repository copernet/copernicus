package protocol

import "time"

const (
	BITCOIN_PROTOCOL_VERSION uint32 = 70012
	NET_ADDRESS_TIME_VERSION uint32 = 31402
)

type ServiceFlag uint64
// InvType represents the allowed types of inventory vectors.  See InvVect.
type InvType uint32
// BitcoinNet represents which bitcoin network a message belongs to.
type BitcoinNet uint32
// BloomUpdateType specifies how the filter is updated when a match is found
type BloomUpdateType uint8


// RejectCode represents a numeric value by which a remote peer indicates
// why a message was rejected.
type RejectCode uint8

type Uint32Time time.Time
type Int64Time time.Time
