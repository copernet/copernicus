package algorithm

import (
	"sync"
	"fmt"
	"time"
	"math"
)

const (
	LIFE_TIME       = 1800
	PRECOMPUTED_LEN = 64
	HALF_LIFE       = 60
	LAMBDA          = math.Ln2 / HALF_LIFE
)

type DynamicBanScore struct {
	lastUnix   int64
	transient  float64
	persistent uint32
	lock       sync.Mutex
}

var precomputedFactor [PRECOMPUTED_LEN]float64

func init() {
	for i := range precomputedFactor {
		precomputedFactor[i] = math.Exp(-1.0 * float64(i) * LAMBDA)
	}
}
func decayFactor(t int64) float64 {
	if t < PRECOMPUTED_LEN {
		return precomputedFactor[t]

	}
	return math.Exp(-1.0 * float64(t) * LAMBDA)
}

func (dynamicBanScore *DynamicBanScore) String() string {
	dynamicBanScore.lock.Lock()
	defer dynamicBanScore.lock.Unlock()
	return fmt.Sprintf("persitent %v , transient %v at %v =%v as of now", dynamicBanScore.persistent, dynamicBanScore.transient, dynamicBanScore.lastUnix, dynamicBanScore.Int())
}
func (dynamicBanScore *DynamicBanScore) persistentInt(t time.Time) uint32 {
	dt := t.Unix() - dynamicBanScore.lastUnix
	if dynamicBanScore.transient < 1 || dt < 0 || LIFE_TIME < dt {
		return dynamicBanScore.persistent
	}
	return dynamicBanScore.persistent + uint32(dynamicBanScore.transient*decayFactor(dt))
}
func (dynamicBanScore *DynamicBanScore) Int() uint32 {
	dynamicBanScore.lock.Lock()
	defer dynamicBanScore.lock.Unlock()
	return dynamicBanScore.persistentInt(time.Now())

}

func (dynamicBanScore *DynamicBanScore) increase(persistent, transient uint32, t time.Time) uint32 {
	dynamicBanScore.persistent += persistent
	timeUnix := t.Unix()
	dt := timeUnix - dynamicBanScore.lastUnix
	if transient > 0 {
		if LIFE_TIME < dt {
			dynamicBanScore.transient = 0
		} else if dynamicBanScore.transient > 1 && dt > 0 {
			dynamicBanScore.transient += float64(transient)
			dynamicBanScore.lastUnix = timeUnix
		}
	}
	return dynamicBanScore.persistent + uint32(dynamicBanScore.transient)
}

func (dynamicBanScore *DynamicBanScore) Increase(persistent, transient uint32) uint32 {
	dynamicBanScore.lock.Lock()
	defer dynamicBanScore.lock.Unlock()
	return dynamicBanScore.increase(persistent, transient, time.Now())
}

func (dynamicBanScore *DynamicBanScore) Reset() {
	dynamicBanScore.lock.Lock()
	dynamicBanScore.lock.Unlock()
	dynamicBanScore.persistent = 0
	dynamicBanScore.transient = 0
	dynamicBanScore.lastUnix = 0
}
