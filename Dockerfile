FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o ntp-dst .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/ntp-dst /usr/local/bin/ntp-dst
EXPOSE 123/udp
ENTRYPOINT ["ntp-dst"]