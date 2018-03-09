package utils

import "sync"

type medianTime struct {
	sync.Mutex
}
