package utils

import "net"

type LookupFunc func(string) ([]net.IP, error)

type HashFunc func() (hash *Hash, height int32, err error)
