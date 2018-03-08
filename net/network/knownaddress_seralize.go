package network

type SerializedKnownAddress struct {
	AddressString string
	Source        string
	Attempts      int
	TimeStamp     int64
	LastAttempt   int64
	LastSuccess   int64
}
