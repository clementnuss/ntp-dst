package main

import "log"

var debugEnabled bool

func logInit(debug bool) {
	debugEnabled = debug
}

func debugf(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf(format, args...)
	}
}