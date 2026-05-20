package main

import (
	"testing"
	"time"
)

func TestSimulatedClock(t *testing.T) {
	start := time.Date(2025, 3, 30, 0, 0, 0, 0, time.UTC)
	sim := NewSimulatedClock(start)

	if !sim.Now().Equal(start) {
		t.Errorf("expected %v, got %v", start, sim.Now())
	}

	sim.Advance(1 * time.Hour)
	if !sim.Now().Equal(start.Add(1 * time.Hour)) {
		t.Errorf("expected %v, got %v", start.Add(1*time.Hour), sim.Now())
	}
}

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

func TestTimeSource_Winter(t *testing.T) {
	sim := NewSimulatedClock(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	source := NewTimeSource("", sim)
	source.SetDstCorrection(0)

	now := source.Now()

	if now.UTC().Format("15:04:05") != "10:00:00" {
		t.Errorf("Winter: NTP should serve UTC (no correction), got %s", now.UTC().Format("15:04:05"))
	}
}

func TestTimeSource_Summer(t *testing.T) {
	sim := NewSimulatedClock(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC))
	source := NewTimeSource("", sim)
	source.SetDstCorrection(1 * time.Hour)

	now := source.Now()

	if now.UTC().Format("15:04:05") != "11:00:00" {
		t.Errorf("Summer: NTP should serve UTC+1h, got %s", now.UTC().Format("15:04:05"))
	}
}

func TestSpringForwardSlew(t *testing.T) {
	slewStart := time.Date(2025, 3, 30, 1, 0, 0, 0, time.UTC)
	sim := NewSimulatedClock(slewStart)
	source := NewTimeSource("", sim)

	source.SetSlew(slewStart, 2.0, 0)

	cases := []struct {
		name       string
		advance    time.Duration
		wantServed string
	}{
		{"01:00 UTC → serves 01:00 (clock+1h=02:00)", 0, "01:00"},
		{"01:15 UTC → serves 01:30 (clock+1h=02:30)", 15 * time.Minute, "01:30"},
		{"01:30 UTC → serves 02:00 (clock+1h=03:00)", 15 * time.Minute, "02:00"},
		{"01:45 UTC → serves 02:30 (clock+1h=03:30)", 15 * time.Minute, "02:30"},
		{"02:00 UTC → serves 03:00 (clock+1h=04:00)", 15 * time.Minute, "03:00"},
	}

	for _, tc := range cases {
		if tc.advance > 0 {
			sim.Advance(tc.advance)
		}
		fakeTime := source.Now()
		if got := fakeTime.UTC().Format("15:04"); got != tc.wantServed {
			t.Errorf("%s: served %s, want %s", tc.name, got, tc.wantServed)
		}
	}

	source.ClearSlew(1 * time.Hour)
	sim.Advance(30 * time.Minute)

	if got := source.Now().UTC().Format("15:04"); got != "03:30" {
		t.Errorf("After clear: served %s, want 03:30", got)
	}
}

func TestFallBackSlew(t *testing.T) {
	slewStart := time.Date(2025, 10, 26, 0, 0, 0, 0, time.UTC)
	sim := NewSimulatedClock(slewStart)
	source := NewTimeSource("", sim)

	source.SetSlew(slewStart, 0.5, 1*time.Hour)

	cases := []struct {
		name       string
		advance    time.Duration
		wantServed string
	}{
		{"00:00 UTC → serves 01:00 (clock+1h=02:00)", 0, "01:00"},
		{"00:30 UTC → serves 01:15 (clock+1h=02:15)", 30 * time.Minute, "01:15"},
		{"01:00 UTC → serves 01:30 (clock+1h=02:30)", 30 * time.Minute, "01:30"},
		{"01:30 UTC → serves 01:45 (clock+1h=02:45)", 30 * time.Minute, "01:45"},
		{"02:00 UTC → serves 02:00 (clock+1h=03:00)", 30 * time.Minute, "02:00"},
	}

	for _, tc := range cases {
		if tc.advance > 0 {
			sim.Advance(tc.advance)
		}
		fakeTime := source.Now()
		if got := fakeTime.UTC().Format("15:04"); got != tc.wantServed {
			t.Errorf("%s: served %s, want %s", tc.name, got, tc.wantServed)
		}
	}

	source.ClearSlew(0)
	sim.Advance(30 * time.Minute)

	if got := source.Now().UTC().Format("15:04"); got != "02:30" {
		t.Errorf("After clear: served %s, want 02:30", got)
	}
}

func TestSpringForwardMath(t *testing.T) {
	slewStart := time.Date(2025, 3, 30, 1, 0, 0, 0, time.UTC)
	baseCorrection := time.Duration(0)

	cases := []struct {
		elapsed  time.Duration
		wantServed time.Time
	}{
		{0, time.Date(2025, 3, 30, 1, 0, 0, 0, time.UTC)},
		{15 * time.Minute, time.Date(2025, 3, 30, 1, 30, 0, 0, time.UTC)},
		{30 * time.Minute, time.Date(2025, 3, 30, 2, 0, 0, 0, time.UTC)},
		{45 * time.Minute, time.Date(2025, 3, 30, 2, 30, 0, 0, time.UTC)},
		{60 * time.Minute, time.Date(2025, 3, 30, 3, 0, 0, 0, time.UTC)},
	}

	for _, tc := range cases {
		fakeElapsed := time.Duration(float64(tc.elapsed) * 2.0)
		result := slewStart.Add(baseCorrection + fakeElapsed)
		if !result.Equal(tc.wantServed) {
			t.Errorf("elapsed %v: served %v, want %v", tc.elapsed, result.Format("15:04:05"), tc.wantServed.Format("15:04:05"))
		}
	}
}

func TestFallBackMath(t *testing.T) {
	slewStart := time.Date(2025, 10, 26, 0, 0, 0, 0, time.UTC)
	baseCorrection := 1 * time.Hour

	cases := []struct {
		elapsed  time.Duration
		wantServed time.Time
	}{
		{0, time.Date(2025, 10, 26, 1, 0, 0, 0, time.UTC)},
		{30 * time.Minute, time.Date(2025, 10, 26, 1, 15, 0, 0, time.UTC)},
		{60 * time.Minute, time.Date(2025, 10, 26, 1, 30, 0, 0, time.UTC)},
		{90 * time.Minute, time.Date(2025, 10, 26, 1, 45, 0, 0, time.UTC)},
		{120 * time.Minute, time.Date(2025, 10, 26, 2, 0, 0, 0, time.UTC)},
	}

	for _, tc := range cases {
		fakeElapsed := time.Duration(float64(tc.elapsed) * 0.5)
		result := slewStart.Add(baseCorrection + fakeElapsed)
		if !result.Equal(tc.wantServed) {
			t.Errorf("elapsed %v: served %v, want %v", tc.elapsed, result.Format("15:04:05"), tc.wantServed.Format("15:04:05"))
		}
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

func TestClearSlew(t *testing.T) {
	ts := NewTimeSource("", nil)

	ts.SetSlew(time.Now().UTC(), 2.0, 0)
	if !ts.SlewActive() {
		t.Error("Expected slew to be active")
	}

	ts.SetDstCorrection(1 * time.Hour)
	if ts.SlewActive() {
		t.Error("Expected slew to be cleared after SetDstCorrection")
	}
}