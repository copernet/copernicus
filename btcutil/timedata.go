package btcutil

import "sync"

type medianTime struct {
	sync.Mutex
}
