package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/beevik/ntp"
)

type TimeSource struct {
	mu             sync.RWMutex
	ntpOffset      time.Duration
	dstCorrection  time.Duration
	slewStart      time.Time
	slewRate       float64
	slewBase       time.Duration
	skew           time.Duration
	clock          Clock
	ntpServer      string
	syncInterval   time.Duration
}

func NewTimeSource(ntpServer string, clock Clock) *TimeSource {
	if clock == nil {
		clock = RealClock{}
	}
	return &TimeSource{
		ntpServer:    ntpServer,
		syncInterval: 5 * time.Minute,
		slewRate:     1.0,
		clock:        clock,
	}
}

func (ts *TimeSource) Run() {
	if _, ok := ts.clock.(*SimulatedClock); ok {
		log.Println("Simulation mode: skipping NTP sync")
		return
	}
	for {
		if err := ts.sync(); err != nil {
			log.Printf("NTP sync failed: %v", err)
		} else {
			log.Printf("NTP sync ok, offset: %v, dst-correction: %v", ts.GetNtpOffset(), ts.GetDstCorrection())
		}
		<-ts.clock.After(ts.syncInterval)
	}
}

func (ts *TimeSource) sync() error {
	response, err := ntp.Query(ts.ntpServer)
	if err != nil {
		return fmt.Errorf("querying %s: %w", ts.ntpServer, err)
	}
	ts.mu.Lock()
	ts.ntpOffset = response.ClockOffset
	ts.mu.Unlock()
	return nil
}

func (ts *TimeSource) GetNtpOffset() time.Duration {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.ntpOffset
}

func (ts *TimeSource) GetDstCorrection() time.Duration {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.dstCorrection
}

func (ts *TimeSource) Now() time.Time {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	realUTC := ts.clock.Now().UTC().Add(ts.ntpOffset).Add(ts.skew)

	if ts.slewRate == 1.0 {
		return realUTC.Add(ts.dstCorrection)
	}

	elapsed := realUTC.Sub(ts.slewStart)
	if elapsed < 0 {
		elapsed = 0
	}
	fakeCorrection := ts.slewBase + time.Duration(float64(elapsed)*ts.slewRate)
	return ts.slewStart.Add(fakeCorrection)
}

func (ts *TimeSource) SetDstCorrection(correction time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.dstCorrection = correction
	ts.slewRate = 1.0
	ts.slewBase = 0
}

func (ts *TimeSource) SetSlew(slewStartUTC time.Time, rate float64, baseCorrection time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.slewStart = slewStartUTC
	ts.slewRate = rate
	ts.slewBase = baseCorrection
}

func (ts *TimeSource) ClearSlew(finalCorrection time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.slewRate = 1.0
	ts.dstCorrection = finalCorrection
	ts.slewBase = 0
}

func (ts *TimeSource) SlewActive() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.slewRate != 1.0
}

func (ts *TimeSource) SlewInfo() (rate float64, start time.Time, base time.Duration) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.slewRate, ts.slewStart, ts.slewBase
}

func (ts *TimeSource) SetSkew(skew time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.skew = skew
}