package main

import (
	"log"
	"time"
)

type DSTScheduler struct {
	source      *TimeSource
	clockOffset time.Duration
	lastIsDST   bool
	initialized bool
}

func NewDSTScheduler(source *TimeSource, clockOffset time.Duration) *DSTScheduler {
	return &DSTScheduler{source: source, clockOffset: clockOffset}
}

func (d *DSTScheduler) Run() {
	loc, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		log.Fatalf("Failed to load timezone: %v", err)
	}

	for {
		nowLocal := time.Now().In(loc)
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
			d.source.SetDstCorrection(d.dstCorrection(true))
		} else if d.lastIsDST && !isDST {
			log.Println("DST transition: CEST -> CET (fall back)")
			d.source.SetDstCorrection(d.dstCorrection(false))
		}

		d.lastIsDST = isDST
		time.Sleep(10 * time.Second)
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