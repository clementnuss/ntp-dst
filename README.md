# ntp-with-dst

A fake NTP server for Mondaine SBB wall clocks (and similar dumb NTP clocks) that compensates for daylight saving time transitions. The clock applies a fixed UTC offset and doesn't know about DST — this server fakes the NTP time so the clock always shows the correct local time.

## How it works

The Mondaine MSM.25S11 clock treats NTP time as UTC and adds a **fixed** offset configured during setup. It syncs only once per 24 hours to save battery. If configured as UTC+1 (winter/CET), it will always add +1h — even during summer.

This server serves `UTC + dstCorrection` where `dstCorrection = correctLocalOffset - clockOffset`. For a clock configured as UTC+1:

| Season | Correct offset | Clock offset | DST correction | NTP serves | Clock displays |
|--------|---------------|--------------|----------------|------------|----------------|
| CET    | UTC+1         | +1h          | 0              | UTC+0      | UTC+0+1h = correct |
| CEST   | UTC+2         | +1h          | +1h            | UTC+1h     | UTC+1h+1h = correct |

During a DST transition, if the clock happens to sync during the transition window, the server gradually slews the time so the clock adjusts smoothly:

- **Spring forward** (CET→CEST): 2x speed for 1 hour
- **Fall back** (CEST→CET): 0.5x speed for 2 hours

Since the clock syncs only once per day, the slew mostly matters for correctness at the exact moment of sync. If the clock syncs outside the transition window (very likely), it simply gets the correct static offset.

## Quick Start

```bash
go build -o ntp-dst .

# For a clock configured as UTC+1 (CET / winter time) — the default
sudo ./ntp-dst -port 123

# For a clock configured as UTC+2 (CEST / summer time)
sudo ./ntp-dst -port 123 -clock-offset 2h

# High port for testing (no root)
./ntp-dst -port 1234
```

Point your clock's NTP server (via DNS) to the machine running this server.

## Configuration

```
Usage of ntp-dst:
  -port int          UDP port to listen on (default 123)
  -ntp string        Upstream NTP server pool (default "ch.pool.ntp.org")
  -clock-offset      Fixed UTC offset configured on the clock (default 1h)
  -simulate string   Simulate starting at this time (RFC3339)
  -speed float       Simulation speed multiplier (default 120)
  -sim-duration      Max simulation wall time (default 3m)
```

## Simulation Mode

Test DST transitions without waiting for the real thing:

```bash
# Spring forward, clock configured as UTC+1
./ntp-dst -port 1234 -simulate 2025-03-30T00:30:00Z -speed 360 -clock-offset 1h

# Fall back, clock configured as UTC+1
./ntp-dst -port 1234 -simulate 2025-10-25T23:30:00Z -speed 360 -clock-offset 1h
```

The simulation output shows "Clock Shows" — what your physical Mondaine clock would display:

```
Wall(s)  | Sim UTC     | Zurich Real          | NTP Serves  | Clock Shows | Status
-------------------------------------------------------------------------------------
0.5      | 00:33:00    | 2025-03-30 01:33:00 CET | 00:33:00    | 01:33:00    | 
5.0      | 01:00:00    | 2025-03-30 03:00:00 CEST | 01:00:00    | 02:00:00    | 
7.5      | 01:15:00    | 2025-03-30 03:15:00 CEST | 01:30:00    | 02:30:00    | SLEW(2.0x)
15.0     | 02:00:00    | 2025-03-30 04:00:00 CEST | 03:00:00    | 04:00:00    | SLEW(2.0x)
15.5     | 02:03:00    | 2025-03-30 04:03:00 CEST | 03:03:00    | 04:03:00    | 
```

## Architecture

- **`main.go`** — Entry point, CLI flags, simulation runner
- **`scheduler.go`** — DST transition detection and slew triggering
- **`source.go`** — Time source with NTP sync and DST correction
- **`server.go`** — UDP NTP server with query logging
- **`ntp.go`** — NTP packet marshaling/unmarshaling
- **`clock.go`** — Clock interface (real/simulated) for testability

## Docker

```bash
docker build -t ntp-dst .
docker run -d --name ntp-dst \
  -p 123:123/udp \
  ntp-dst -clock-offset 1h
```

## Running Tests

```bash
go test -v ./...
```

## Reference

- [How smart is the Mondaine MSM.25S11 wifi wall clock?](https://lieven.kks36.be/2023/11/08/how-smart-is-the-mondaine-msm-25s11-wifi-wall-clock/) — The Mondaine clock uses an ESP chip, syncs once per 24h via NTP from `pool.ntp.org`, and applies a fixed timezone offset configured at setup time.