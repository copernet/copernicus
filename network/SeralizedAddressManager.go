package network

type SerializedAddressManager struct {
	Version      int
	Key          [32]byte
	Addresses    []*SerializedKnownAddress
	NewBuckets   [NEW_BUCKET_COUNT][]string
	TriedBuckets [TRIED_BUCKET_COUNT][]string
}
