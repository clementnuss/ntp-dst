package main

import (
	"flag"
	"fmt"
	"log"
	"time"
)

func main() {
	port := flag.Int("port", 123, "UDP port to listen on")
	ntpServer := flag.String("ntp", "ch.pool.ntp.org", "Upstream NTP server to sync from")
	clockOffset := flag.Duration("clock-offset", 1*time.Hour, "Fixed UTC offset configured on the clock (e.g. 1h for CET)")
	skew := flag.Duration("skew", 0, "Extra time offset to serve (for debugging, e.g. 5m for +5min)")
	simTime := flag.String("simulate", "", "Simulate starting at this time (RFC3339)")
	simSpeed := flag.Float64("speed", 120, "Simulation speed multiplier")
	simDuration := flag.Duration("sim-duration", 3*time.Minute, "Max simulation wall time")
	flag.Parse()

	if *simTime != "" {
		runSimulation(*port, *ntpServer, *clockOffset, *simTime, *simSpeed, *simDuration)
	} else {
		runReal(*port, *ntpServer, *clockOffset, *skew)
	}
}

func runReal(port int, ntpServer string, clockOffset time.Duration, skew time.Duration) {
	source := NewTimeSource(ntpServer, nil)
	source.SetSkew(skew)
	go source.Run()

	scheduler := NewDSTScheduler(source, nil, clockOffset)
	go scheduler.Run()

	server := NewServer(port, source)
	server.SetClockOffset(clockOffset)
	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runSimulation(port int, ntpServer string, clockOffset time.Duration, simTimeStr string, speed float64, maxWallTime time.Duration) {
	startTime, err := time.Parse(time.RFC3339, simTimeStr)
	if err != nil {
		log.Fatalf("Invalid simulate time %q: %v", simTimeStr, err)
	}

	sim := NewSimulatedClock(startTime.UTC())
	source := NewTimeSource(ntpServer, sim)

	scheduler := NewDSTScheduler(source, sim, clockOffset)
	go scheduler.Run()

	server := NewServer(port, source)
	server.SetClockOffset(clockOffset)
	go func() {
		if err := server.Run(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	loc, _ := time.LoadLocation("Europe/Zurich")
	stepPerTick := time.Duration(float64(500*time.Millisecond) * speed)

	fmt.Printf("Simulation: start=%s speed=%.0fx clock-offset=%s\n\n",
		startTime.UTC().Format("2006-01-02 15:04:05 MST"), speed, clockOffset)
	fmt.Printf("%-8s | %-11s | %-20s | %-11s | %-11s\n",
		"Wall(s)", "Sim UTC", "Zurich Real", "NTP Serves", "Clock Shows")
	fmt.Println("-------------------------------------------------------------------------------")

	wallStart := time.Now()
	wallTimeout := time.After(maxWallTime)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sim.Advance(stepPerTick)

			now := sim.Now()
			fakeTime := source.Now()
			clockShows := fakeTime.UTC().Add(clockOffset)

			wallElapsed := time.Since(wallStart).Seconds()
			fmt.Printf("%-8.1f | %-11s | %-20s | %-11s | %-11s\n",
				wallElapsed,
				now.UTC().Format("15:04:05"),
				now.In(loc).Format("2006-01-02 15:04:05 MST"),
				fakeTime.UTC().Format("15:04:05"),
				clockShows.Format("15:04:05"),
			)

		case <-wallTimeout:
			fmt.Printf("\nSimulation complete\n")
			return
		}
	}
}