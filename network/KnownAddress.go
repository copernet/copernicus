package network

import (
	"time"
)

type KnownAddress struct {
	NetAddress  *PeerAddress
	SrcAddress  *PeerAddress
	attempts    int
	LastAttempt time.Time
	lastSuccess time.Time
	tried       bool
	refs        int
}

func (knownAddress *KnownAddress) Chance() float64 {
	now := time.Now()
	lastSeen := now.Sub(knownAddress.NetAddress.Timestamp)
	lastAttempt := now.Sub(knownAddress.LastAttempt)
	if lastSeen < 0 {
		lastSeen = 0
	}
	if lastAttempt < 0 {
		lastAttempt = 0
	}
	chance := .0
	if lastAttempt < 10*time.Minute {
		chance *= 0.01
	}
	for i := knownAddress.attempts; i > 0; i-- {
		chance /= 1.5
	}
	return chance
}
func (knownAddress *KnownAddress) IsBad() bool {
	if knownAddress.LastAttempt.After(time.Now().Add(-1 * time.Minute)) {
		return false
	}
	if knownAddress.NetAddress.Timestamp.After(time.Now().Add(10 * time.Minute)) {
		return true
	}
	if knownAddress.NetAddress.Timestamp.Before(time.Now().Add(-1 * NumMissingDays * time.Hour * 24)) {
		return true
	}
	if knownAddress.lastSuccess.IsZero() && knownAddress.attempts >= NumReties {
		return true
	}
	if !knownAddress.lastSuccess.After(time.Now().Add(-1*MinBadDays*time.Hour*24)) && knownAddress.attempts >= MaxFailures {
		return true
	}
	return false

}
