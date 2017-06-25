package utils

import "net"

type LookupFunc func(string) ([]net.IP, error)