package network

type SerializedAddressManager struct {
	Version      int
	Key          [32]byte
	Addresses    []*SerializedKnownAddress
	NewBuckets   [BucketCount][]string
	TriedBuckets [TriedBucketCount][]string
}
