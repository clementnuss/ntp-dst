package main

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time                    { return time.Now() }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type SimulatedClock struct {
	mu    sync.RWMutex
	now   time.Time
	after struct {
		mu    sync.Mutex
		chans []simAfter
	}
}

type simAfter struct {
	fireAt time.Time
	ch     chan time.Time
	fired  bool
}

func NewSimulatedClock(start time.Time) *SimulatedClock {
	return &SimulatedClock{
		now: start,
	}
}

func (sc *SimulatedClock) Now() time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.now
}

func (sc *SimulatedClock) After(d time.Duration) <-chan time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	fireAt := sc.now.Add(d)
	ch := make(chan time.Time, 1)

	sc.after.mu.Lock()
	sc.after.chans = append(sc.after.chans, simAfter{fireAt: fireAt, ch: ch})
	sc.after.mu.Unlock()

	return ch
}

func (sc *SimulatedClock) Advance(d time.Duration) {
	sc.mu.Lock()
	sc.now = sc.now.Add(d)
	currentNow := sc.now
	sc.mu.Unlock()

	sc.after.mu.Lock()
	defer sc.after.mu.Unlock()

	stillActive := make([]simAfter, 0, len(sc.after.chans))
	for _, sa := range sc.after.chans {
		if !sa.fired && !currentNow.Before(sa.fireAt) {
			sa.ch <- currentNow
			sa.fired = true
		}
		if !sa.fired {
			stillActive = append(stillActive, sa)
		}
	}
	sc.after.chans = stillActive
}