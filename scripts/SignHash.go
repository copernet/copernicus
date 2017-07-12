package scripts

type SignHash uint32

const (
	SignAll SignHash = iota + 1
	SignNone
	SignSingle
)
