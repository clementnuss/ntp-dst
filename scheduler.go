package main

import (
	"log"
	"time"
)

type DSTScheduler struct {
	source      *TimeSource
	clock       Clock
	clockOffset time.Duration
	lastIsDST   bool
	initialized bool
}

func NewDSTScheduler(source *TimeSource, clock Clock, clockOffset time.Duration) *DSTScheduler {
	if clock == nil {
		clock = RealClock{}
	}
	return &DSTScheduler{source: source, clock: clock, clockOffset: clockOffset}
}

func (d *DSTScheduler) Run() {
	loc, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		log.Fatalf("Failed to load timezone: %v", err)
	}

	for {
		nowLocal := d.clock.Now().In(loc)
		_, offset := nowLocal.Zone()
		isDST := offset == 2*3600

		if !d.initialized {
			correction := d.dstCorrection(isDST)
			d.source.SetDstCorrection(correction)
			log.Printf("Initial: %s, clock-offset=%s, correction=%s",
				tzName(isDST), d.clockOffset, correction)
			d.lastIsDST = isDST
			d.initialized = true
		}

		if !d.lastIsDST && isDST {
			log.Println("DST transition: CET -> CEST (spring forward)")
			nowUTC := d.clock.Now().UTC()
			slewStart := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 1, 0, 0, 0, time.UTC)
			slewEnd := slewStart.Add(1 * time.Hour)
			baseCorrection := d.dstCorrection(false)
			d.source.SetSlew(slewStart, 2.0, baseCorrection)
			remaining := slewEnd.Sub(nowUTC)
			if remaining > 0 {
				go func() {
					<-d.clock.After(remaining)
					d.source.ClearSlew(d.dstCorrection(true))
					log.Println("Spring-forward slew complete")
				}()
			} else {
				d.source.ClearSlew(d.dstCorrection(true))
				log.Println("Spring-forward slew already complete")
			}
		} else if d.lastIsDST && !isDST {
			log.Println("DST transition: CEST -> CET (fall back)")
			nowUTC := d.clock.Now().UTC()
			slewStart := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
			slewEnd := slewStart.Add(2 * time.Hour)
			baseCorrection := d.dstCorrection(true)
			d.source.SetSlew(slewStart, 0.5, baseCorrection)
			remaining := slewEnd.Sub(nowUTC)
			if remaining > 0 {
				go func() {
					<-d.clock.After(remaining)
					d.source.ClearSlew(d.dstCorrection(false))
					log.Println("Fall-back slew complete")
				}()
			} else {
				d.source.ClearSlew(d.dstCorrection(false))
				log.Println("Fall-back slew already complete")
			}
		}

		d.lastIsDST = isDST
		<-d.clock.After(10 * time.Second)
	}
}

func (d *DSTScheduler) dstCorrection(isDST bool) time.Duration {
	var correctOffset time.Duration
	if isDST {
		correctOffset = 2 * time.Hour
	} else {
		correctOffset = 1 * time.Hour
	}
	return correctOffset - d.clockOffset
}

func tzName(isDST bool) string {
	if isDST {
		return "CEST"
	}
	return "CET"
}