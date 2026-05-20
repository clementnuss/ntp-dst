# AGENTS.md

## Build & Test

- `go build -o ntp-dst .` — build binary
- `go test -v ./...` — run tests (all in `package main`, no sub-packages)
- No linting, formatting, or typecheck config beyond Go defaults

## Architecture

Single `main` package; no internal packages or modules.

- `main.go` — CLI flags, simulation runner, real-server startup
- `scheduler.go` — `DSTScheduler`: detects CET/CEST transitions via `time.LoadLocation("Europe/Zurich")`, triggers slew
- `source.go` — `TimeSource`: serves faked time (UTC + dstCorrection + slew). Holds mutex-protected state for NTP offset, DST correction, slew rate/skew
- `server.go` — UDP NTP server; responds to queries with faked time
- `ntp.go` — NTP packet marshal/unmarshal (48-byte SNTP)
- `clock.go` — `Clock` interface with `RealClock` and `SimulatedClock` implementations. `SimulatedClock.Advance()` drives time in tests and simulation mode

## Key Design Facts

- Timezone is hardcoded to **Europe/Zurich** (CET/CEST)
- DST detection compares `Zone()` offset: `7200` = CEST (summer), `3600` = CET (winter)
- Slew during transitions: spring forward = **2x rate for 1h**, fall back = **0.5x rate for 2h**
- Clock syncs once per 24h (hardware constraint of [Mondaine MSM.25S11](https://lieven.kks36.be/2023/11/08/how-smart-is-the-mondaine-msm-25s11-wifi-wall-clock/)); clock treats NTP time as UTC and applies a fixed offset, has no DST awareness
- Go module path is `github.com/clementnuss/ntp-with-dst` (note: "ntp-with-dst", not "ntp-dst")
- Port 123 (default) requires root; use `-port 1234` for local testing

## Dockerfile Note

Dockerfile uses `golang:1.23-alpine` but `go.mod` specifies `go 1.26.2`. Build may fail if the Dockerfile Go version is older than the module's Go directive. Adjust Dockerfile base image if needed.

