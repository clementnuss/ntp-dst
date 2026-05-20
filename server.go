package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Server struct {
	port        int
	source      *TimeSource
	clockOffset time.Duration
	mu          sync.Mutex
	refID       uint32
	refTime     NTPTimestamp
}

func NewServer(port int, source *TimeSource) *Server {
	return &Server{
		port:   port,
		source: source,
		refID:  0x474F4C44,
	}
}

func (s *Server) SetClockOffset(offset time.Duration) {
	s.clockOffset = offset
}

func (s *Server) Run() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("resolving UDP addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("listening on UDP %d: %v", s.port, err)
	}
	defer conn.Close()

	log.Printf("NTP server listening on UDP port %d", s.port)

	buf := make([]byte, 48)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		if n < 48 {
			log.Printf("Packet too short: %d bytes", n)
			continue
		}

		go s.handlePacket(conn, remoteAddr, buf[:n])
	}
}

func (s *Server) handlePacket(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	req := UnmarshalNTPPacket(data)
	if req == nil {
		return
	}

	fakeNow := s.source.Now()

	clientVer := (req.Settings >> 3) & 0x7
	clientMode := req.Settings & 0x7
	slewActive := s.source.SlewActive()
	rate, _, _ := s.source.SlewInfo()

	slewInfo := ""
	if slewActive {
		slewInfo = fmt.Sprintf(" slew=%.1fx", rate)
	}

	clockDisplays := fakeNow.UTC().Add(s.clockOffset)

	resp := &NTPPacket{
		Settings:       0x24,
		Stratum:        2,
		Poll:           req.Poll,
		Precision:      -20,
		RootDelay:      0x100,
		RootDispersion: 0x100,
		ReferenceID:    s.refID,
		RefTime:        s.refTime,
		OrigTime:       req.TxTime,
		RxTime:         timeToNTP(fakeNow),
		TxTime:         timeToNTP(fakeNow),
	}

	log.Printf("NTP query from %s: client=v%d m%d stratum=%d poll=%d",
		addr.IP, clientVer, clientMode, req.Stratum, req.Poll)
	log.Printf("  served=%s clock=%s corr=%s resp=v%d s=%d poll=%d prec=%d ref=0x%08X%s",
		fakeNow.UTC().Format("2006-01-02 15:04:05"),
		clockDisplays.Format("15:04:05"),
		s.source.GetDstCorrection(),
		(resp.Settings>>3)&0x7, resp.Stratum, resp.Poll, resp.Precision, s.refID,
		slewInfo,
	)

	s.mu.Lock()
	s.refTime = resp.TxTime
	s.mu.Unlock()

	responseData := resp.Marshal()
	_, err := conn.WriteToUDP(responseData, addr)
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}