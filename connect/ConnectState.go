package connect

type ConnectState uint8

const (
	ConnectPending ConnectState = iota
	ConnectEstablished
	ConnectDisconnected
	ConnectFailed
)
