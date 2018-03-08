package utils

import (
	"math/rand"
	"time"
)

type FastRandomContext struct {
	rz uint32
	rw uint32
}

func NewFastRandomContext(fDeterministic bool) *FastRandomContext {
	fastRandomContext := FastRandomContext{}

	if fDeterministic {
		fastRandomContext.rw = 11
		fastRandomContext.rz = 11
	} else {
		rand.Seed(time.Now().UnixNano())
		tmp := rand.Uint32()
		for tmp == 0 || tmp == 0x9068ffff {
			tmp = rand.Uint32()
		}
		fastRandomContext.rz = tmp

		tmp = rand.Uint32()
		for tmp == 0 || tmp == 0x464fffff {
			tmp = rand.Uint32()
		}
		fastRandomContext.rw = tmp
	}
	return &fastRandomContext
}

func (f *FastRandomContext) Rand32() uint32 {
	f.rz = 36969*(f.rz&65535) + (f.rz >> 16)
	f.rw = 18000*(f.rw&65535) + (f.rw >> 16)
	return (f.rw << 16) + f.rz
}
