package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beevik/ntp"
	_ "time/tzdata"
)

func main() {
	port := flag.Int("port", 123, "UDP port to listen on")
	ntpServer := flag.String("ntp", "ch.pool.ntp.org", "Upstream NTP server to sync from")
	clockOffset := flag.Duration("clock-offset", 1*time.Hour, "Fixed UTC offset configured on the clock (e.g. 1h for CET)")
	skew := flag.Duration("skew", 0, "Extra time offset to serve (for debugging, e.g. 5m for +5min)")
	debug := flag.Bool("debug", false, "Enable debug logging (NTP sync details)")
	query := flag.Bool("query", false, "Query an NTP server and print result (use -query-host and -query-port)")
	queryHost := flag.String("query-host", "localhost", "Host to query (with -query)")
	queryPort := flag.Int("query-port", 1234, "Port to query (with -query)")
	flag.Parse()

	if *query {
		resp, err := ntp.QueryWithOptions(*queryHost, ntp.QueryOptions{Port: *queryPort})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Time:      %s\n", resp.Time.UTC().Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Offset:    %v\n", resp.ClockOffset)
		fmt.Printf("Stratum:   %d\n", resp.Stratum)
		fmt.Printf("Reference: 0x%08X\n", resp.ReferenceID)
		return
	}

	logInit(*debug)

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
