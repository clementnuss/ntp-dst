.PHONY: build build-arm64 build-unikernel build-unikernel-arm64 test clean

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ntp-dst .

build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ntp-dst-arm64 .

build-unikernel: build
	ops image create ntp-dst -c ops.json

build-unikernel-arm64: build-arm64
	ops image create ntp-dst-arm64 -c ops.arm64.json --arch=arm64 --nightly

test:
	go test -v ./...

clean:
	rm -f ntp-dst ntp-dst-arm64