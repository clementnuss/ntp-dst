package main

import (
	"encoding/binary"
	"time"
)

const ntpEpochOffset = 2208988800

type NTPTimestamp struct {
	Seconds  uint32
	Fraction uint32
}

type NTPPacket struct {
	Settings       uint8
	Stratum        uint8
	Poll           int8
	Precision      int8
	RootDelay      uint32
	RootDispersion uint32
	ReferenceID    uint32
	RefTime        NTPTimestamp
	OrigTime       NTPTimestamp
	RxTime         NTPTimestamp
	TxTime         NTPTimestamp
}

func timeToNTP(t time.Time) NTPTimestamp {
	seconds := t.Unix() + ntpEpochOffset
	nanos := t.Nanosecond()
	fraction := uint32((uint64(nanos) << 32) / 1e9)
	return NTPTimestamp{
		Seconds:  uint32(seconds),
		Fraction: fraction,
	}
}

func ntpToTime(ts NTPTimestamp) time.Time {
	sec := int64(ts.Seconds) - ntpEpochOffset
	nano := int64((uint64(ts.Fraction) * 1e9) >> 32)
	return time.Unix(sec, nano)
}

func (p *NTPPacket) Marshal() []byte {
	buf := make([]byte, 48)
	buf[0] = p.Settings
	buf[1] = p.Stratum
	buf[2] = byte(p.Poll)
	buf[3] = byte(p.Precision)
	binary.BigEndian.PutUint32(buf[4:8], p.RootDelay)
	binary.BigEndian.PutUint32(buf[8:12], p.RootDispersion)
	binary.BigEndian.PutUint32(buf[12:16], p.ReferenceID)
	binary.BigEndian.PutUint32(buf[16:20], p.RefTime.Seconds)
	binary.BigEndian.PutUint32(buf[20:24], p.RefTime.Fraction)
	binary.BigEndian.PutUint32(buf[24:28], p.OrigTime.Seconds)
	binary.BigEndian.PutUint32(buf[28:32], p.OrigTime.Fraction)
	binary.BigEndian.PutUint32(buf[32:36], p.RxTime.Seconds)
	binary.BigEndian.PutUint32(buf[36:40], p.RxTime.Fraction)
	binary.BigEndian.PutUint32(buf[40:44], p.TxTime.Seconds)
	binary.BigEndian.PutUint32(buf[44:48], p.TxTime.Fraction)
	return buf
}

func UnmarshalNTPPacket(data []byte) *NTPPacket {
	if len(data) < 48 {
		return nil
	}
	p := &NTPPacket{}
	p.Settings = data[0]
	p.Stratum = data[1]
	p.Poll = int8(data[2])
	p.Precision = int8(data[3])
	p.RootDelay = binary.BigEndian.Uint32(data[4:8])
	p.RootDispersion = binary.BigEndian.Uint32(data[8:12])
	p.ReferenceID = binary.BigEndian.Uint32(data[12:16])
	p.RefTime = NTPTimestamp{
		Seconds:  binary.BigEndian.Uint32(data[16:20]),
		Fraction: binary.BigEndian.Uint32(data[20:24]),
	}
	p.OrigTime = NTPTimestamp{
		Seconds:  binary.BigEndian.Uint32(data[24:28]),
		Fraction: binary.BigEndian.Uint32(data[28:32]),
	}
	p.RxTime = NTPTimestamp{
		Seconds:  binary.BigEndian.Uint32(data[32:36]),
		Fraction: binary.BigEndian.Uint32(data[36:40]),
	}
	p.TxTime = NTPTimestamp{
		Seconds:  binary.BigEndian.Uint32(data[40:44]),
		Fraction: binary.BigEndian.Uint32(data[44:48]),
	}
	return p
}