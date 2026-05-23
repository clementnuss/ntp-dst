package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/beevik/ntp"
)

type TimeSource struct {
	mu            sync.RWMutex
	ntpOffset     time.Duration
	dstCorrection time.Duration
	skew          time.Duration
	clock         Clock
	ntpServer     string
	syncInterval  time.Duration
}

func NewTimeSource(ntpServer string, clock Clock) *TimeSource {
	if clock == nil {
		clock = RealClock{}
	}
	return &TimeSource{
		ntpServer:    ntpServer,
		syncInterval: 5 * time.Minute,
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
	return realUTC.Add(ts.dstCorrection)
}

func (ts *TimeSource) SetDstCorrection(correction time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.dstCorrection = correction
}

func (ts *TimeSource) SetSkew(skew time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.skew = skew
}