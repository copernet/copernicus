package network

type AddressPriority int

const (
	InterfacePriority AddressPriority = iota
	BoundPriority
	UpnpPriority
	HTTPPriority
	ManualPriority
)
