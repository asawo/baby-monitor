package main

import (
	"fmt"
	"os"
	"time"
)

func logf(format string, a ...any) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stderr, "["+ts+"] "+format+"\n", a...)
}

func logFatal(format string, a ...any) {
	logf(format, a...)
	os.Exit(1)
}
