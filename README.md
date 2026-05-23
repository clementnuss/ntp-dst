# ntp-with-dst

A fake NTP server for Mondaine SBB wall clocks (and similar dumb NTP clocks) that compensates for daylight saving time transitions. The clock applies a fixed UTC offset and doesn't know about DST — this server fakes the NTP time so the clock always shows the correct local time.

## How it works

The Mondaine MSM.25S11 clock treats NTP time as UTC and adds a **fixed** offset configured during setup. It syncs only once per 24 hours to save battery. If configured as UTC+1 (winter/CET), it will always add +1h — even during summer.

This server serves `UTC + dstCorrection` where `dstCorrection = correctLocalOffset - clockOffset`. When a DST transition is detected, the correction is applied immediately. For a clock configured as UTC+1:

| Season | Correct offset | Clock offset | DST correction | NTP serves | Clock displays |
|--------|---------------|--------------|----------------|------------|----------------|
| CET    | UTC+1         | +1h          | 0              | UTC+0      | UTC+0+1h = correct |
| CEST   | UTC+2         | +1h          | +1h            | UTC+1h     | UTC+1h+1h = correct |

Since the clock syncs only once per day, the correction takes effect the next time it syncs.

## Quick Start

```bash
make build

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
```

## Architecture

- **`main.go`** — Entry point, CLI flags
- **`scheduler.go`** — DST transition detection and correction
- **`source.go`** — Time source with NTP sync and DST correction
- **`server.go`** — UDP NTP server with query logging
- **`ntp.go`** — NTP packet marshaling/unmarshaling

## Docker

```bash
docker build -t ntp-dst .
docker run -d --name ntp-dst \
  -p 123:123/udp \
  ntp-dst -clock-offset 1h
```

## Unikernel (Nanos)

Run as a [Nanos](https://nanos.org) unikernel via [OPS](https://ops.city) — no OS to maintain, no SSH, minimal attack surface.

```bash
# Install OPS
curl https://ops.city/get.sh -sSfL | sh

# x86_64 (VM.Standard.E2.1.Micro)
make build-unikernel
ops run -c ops.json ntp-dst

# ARM64 / Ampere A1 (VM.Standard.A1.Flex)
make build-unikernel-arm64
ops run -c ops.arm64.json ntp-dst-arm64
```

### Deploy to OCI

Fill in `BucketName` and `BucketNamespace` in `ops.json` or `ops.arm64.json`, then:

```bash
# x86_64
ops image create ntp-dst -t oci -c ops.json
ops instance create ntp-dst -t oci -c ops.json

# ARM64
ops image create ntp-dst-arm64 -t oci -c ops.arm64.json --arch=arm64
ops instance create ntp-dst-arm64 -t oci -c ops.arm64.json
```

Remember to open UDP 123 in your OCI security list for the instance's VCN.

## Running Tests

```bash
make test
```

## Reference

- [How smart is the Mondaine MSM.25S11 wifi wall clock?](https://lieven.kks36.be/2023/11/08/how-smart-is-the-mondaine-msm-25s11-wifi-wall-clock/) — The Mondaine clock uses an ESP chip, syncs once per 24h via NTP from `pool.ntp.org`, and applies a fixed timezone offset configured at setup time.