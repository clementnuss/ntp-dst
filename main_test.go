package main

import (
	"testing"
	"time"
)

func TestDstCorrection_Winter(t *testing.T) {
	clockOffset := 1 * time.Hour

	scheduler := &DSTScheduler{clockOffset: clockOffset}

	correction := scheduler.dstCorrection(false)
	if correction != 0 {
		t.Errorf("CET (winter): correction should be 0, got %v", correction)
	}
}

func TestDstCorrection_Summer(t *testing.T) {
	clockOffset := 1 * time.Hour

	scheduler := &DSTScheduler{clockOffset: clockOffset}

	correction := scheduler.dstCorrection(true)
	if correction != 1*time.Hour {
		t.Errorf("CEST (summer): correction should be 1h, got %v", correction)
	}
}

func TestDstCorrection_ClockOffset0(t *testing.T) {
	scheduler := &DSTScheduler{clockOffset: 0}

	if correction := scheduler.dstCorrection(false); correction != 1*time.Hour {
		t.Errorf("CET with clock+0: correction should be 1h, got %v", correction)
	}
	if correction := scheduler.dstCorrection(true); correction != 2*time.Hour {
		t.Errorf("CEST with clock+0: correction should be 2h, got %v", correction)
	}
}

func TestDstCorrection_ClockOffset2h(t *testing.T) {
	scheduler := &DSTScheduler{clockOffset: 2 * time.Hour}

	if correction := scheduler.dstCorrection(false); correction != -1*time.Hour {
		t.Errorf("CET with clock+2h: correction should be -1h, got %v", correction)
	}
	if correction := scheduler.dstCorrection(true); correction != 0 {
		t.Errorf("CEST with clock+2h: correction should be 0, got %v", correction)
	}
}

func TestNTPPacketMarshalUnmarshal(t *testing.T) {
	now := time.Now()
	pkt := &NTPPacket{
		Settings:       0x24,
		Stratum:        2,
		Poll:           4,
		Precision:      -20,
		RootDelay:      0x100,
		RootDispersion: 0x200,
		ReferenceID:    0x474F4C44,
		RefTime:        timeToNTP(now),
		OrigTime:       timeToNTP(now.Add(-1 * time.Second)),
		RxTime:         timeToNTP(now),
		TxTime:         timeToNTP(now),
	}

	data := pkt.Marshal()
	parsed := UnmarshalNTPPacket(data)

	if parsed.Settings != pkt.Settings {
		t.Errorf("Settings mismatch")
	}
	if parsed.Stratum != pkt.Stratum {
		t.Errorf("Stratum mismatch")
	}
	if parsed.ReferenceID != pkt.ReferenceID {
		t.Errorf("ReferenceID mismatch")
	}
}

func TestTimeToNTPRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	ts := timeToNTP(now)
	result := ntpToTime(ts)

	diff := now.Sub(result)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Millisecond {
		t.Errorf("Round-trip error: %v", diff)
	}
}