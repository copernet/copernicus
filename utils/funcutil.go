package utils

import "net"

type LookupFunc func(string) ([]net.IP, error)

type HashFunc func() (hash *Hash, height int32, err error)

func Max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func Min(a int, b int) int {
	if a <= b {
		return a
	}
	return b
}
