package protocol

import (
	"copernicus/crypto"
)

type HashFunc func() (hash *crypto.Hash, height int32, err error)

type HostToNetAddrFunc func(host string, port uint16, serviceFlag ServiceFlag)


