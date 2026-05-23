# AGENTS.md

## Build & Test

- `make build` — build static Linux binary (with tzdata embedded, stripped)
- `make build-unikernel` — build Nanos unikernel image via `ops`
- `make test` — run tests (all in `package main`, no sub-packages)
- No linting, formatting, or typecheck config beyond Go defaults

## Architecture

Single `main` package; no internal packages or modules.

- `main.go` — CLI flags, real-server startup
- `scheduler.go` — `DSTScheduler`: detects CET/CEST transitions via `time.LoadLocation("Europe/Zurich")`, applies correction immediately
- `source.go` — `TimeSource`: serves faked time (UTC + dstCorrection). Holds mutex-protected state for NTP offset, DST correction, skew
- `server.go` — UDP NTP server; responds to queries with faked time
- `ntp.go` — NTP packet marshal/unmarshal (48-byte SNTP)

## Key Design Facts

- Timezone is hardcoded to **Europe/Zurich** (CET/CEST)
- DST detection compares `Zone()` offset: `7200` = CEST (summer), `3600` = CET (winter)
- DST corrections are applied immediately on transition (no slew/gradual adjustment)
- Clock syncs once per 24h (hardware constraint of [Mondaine MSM.25S11](https://lieven.kks36.be/2023/11/08/how-smart-is-the-mondaine-msm-25s11-wifi-wall-clock/)); clock treats NTP time as UTC and applies a fixed offset, has no DST awareness
- Go module path is `github.com/clementnuss/ntp-with-dst` (note: "ntp-with-dst", not "ntp-dst")
- Port 123 (default) requires root; use `-port 1234` for local testing

## Dockerfile Note

Dockerfile uses `golang:1.23-alpine` but `go.mod` specifies `go 1.26.2`. Build may fail if the Dockerfile Go version is older than the module's Go directive. Adjust Dockerfile base image if needed.

## Unikernel (Nanos/OPS)

- `ops.json` — x86_64 config (Flavor: `VM.Standard.E2.1.Micro`)
- `ops.arm64.json` — ARM64/Ampere A1 config (Flavor: `VM.Standard.A1.Flex`)
- Build uses `-tags tzdata` to embed timezone data (no `/usr/share/zoneinfo` needed in image)
- `CGO_ENABLED=0` ensures pure-Go DNS resolver (no NSS shared libs needed)
- No CA certs or shared libraries needed — NTP uses UDP, not TLS