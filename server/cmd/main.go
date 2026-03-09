package main

import (
	"net/http"

	"babymonitor/server/internal/api"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", api.StatusHandler)
	mux.HandleFunc("/api/logs", api.LogsHandler)
	mux.HandleFunc("/api/notifications", api.NotificationsHandler)
	mux.HandleFunc("/api/cry", api.CryHandler)
	mux.HandleFunc("/api/fart", api.FartHandler)
	mux.HandleFunc("/api/detect-status", api.DetectStatusHandler)
	mux.Handle("/", noCacheMiddleware(http.FileServer(http.Dir("./server/web"))))

	addr := ":80"
	logf("Serving on http://0.0.0.0%s", addr)

	if err := http.ListenAndServe(addr, privateNetworkMiddleware(mux)); err != nil {
		logFatal("%v", err)
	}
}
