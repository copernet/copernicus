package consensus

const (
	// DefaultPermitBareMultiSig  Default for -permitBareMultiSig
	DefaultPermitBareMultiSig      = true
	DefaultCheckPointsEnabled      = true
	DefaultTxIndex                 = false
	DefaultBanscoreThreshold  uint = 100
	// MinBlocksToKeep of chainActive.Tip() will not be pruned.
	MinBlocksToKeep      = 288
	DefaultMaxTipAge     = 24 * 60 * 60
	DefaultRelayPriority = true

	// DefaultMemPoolExpiry Default for -memPoolExpiry, expiration time
	// for memPool transactions in hours
	DefaultMemPoolExpiry       = 336
	MemPoolDumpVersion         = 1
	DefaultLimitFreeRelay      = 0
	DefaultAncestorLimit       = 25
	DefaultAncestorSizeLimit   = 101
	DefaultDescendantLimit     = 25
	DefaultDescendantSizeLimit = 101
	MaxFeeEstimationTipAge     = 3 * 60 * 60
	// MinDiskSpace Minimum disk space required - used in CheckDiskSpace()
	MinDiskSpace = 52428800
)

// Reject codes greater or equal to this can be returned by AcceptToMemPool for
// transactions, to signal internal conditions. They cannot and should not be
// sent over the P2P network.
const (
	RejectInternal = 0x100
	// RejectHighFee too high fee. Can not be triggered by P2P transactions
	RejectHighFee = 0x100
	// RejectAlreadyKnown Transaction is already known (either in memPool or blockChain)
	RejectAlreadyKnown = 0x101
	// RejectConflict transaction conflicts with a transaction already known
	RejectConflict = 0x102
)
