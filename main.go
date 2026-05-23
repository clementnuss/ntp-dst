package main

import (
	"flag"
	"log"
	"time"
)

func main() {
	port := flag.Int("port", 123, "UDP port to listen on")
	ntpServer := flag.String("ntp", "ch.pool.ntp.org", "Upstream NTP server to sync from")
	clockOffset := flag.Duration("clock-offset", 1*time.Hour, "Fixed UTC offset configured on the clock (e.g. 1h for CET)")
	skew := flag.Duration("skew", 0, "Extra time offset to serve (for debugging, e.g. 5m for +5min)")
	flag.Parse()

	source := NewTimeSource(*ntpServer)
	source.SetSkew(*skew)
	go source.Run()

	scheduler := NewDSTScheduler(source, *clockOffset)
	go scheduler.Run()

	server := NewServer(*port, source)
	server.SetClockOffset(*clockOffset)
	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
