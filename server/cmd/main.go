package main

import (
	"net"
	"net/http"

	"babymonitor/server/internal/api"
	"babymonitor/server/internal/config"
	"babymonitor/server/internal/notify"
	"babymonitor/server/internal/state"
	"babymonitor/server/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logFatal("config: %v", err)
	}

	db, err := store.New(cfg.DBPath)
	if err != nil {
		logFatal("store: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logf("store close: %v", err)
		}
	}()

	s := state.New(db)
	h := api.New(s)
	ntfy := notify.New(cfg.NotifySocket, cfg.WatchdogUsec)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/status", h.StatusHandler)
	mux.HandleFunc("/api/logs", h.LogsHandler)
	mux.HandleFunc("/api/notifications", h.NotificationsHandler)
	mux.HandleFunc("/api/cry", h.CryHandler)
	mux.HandleFunc("/api/fart", h.FartHandler)
	mux.HandleFunc("/api/detect-status", h.DetectStatusHandler)
	mux.Handle("/", noCacheMiddleware(http.FileServer(http.Dir("./server/web"))))

	addr := ":80"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logFatal("listen: %v", err)
	}

	logf("Serving on http://0.0.0.0%s", addr)

	if err := ntfy.Notify("READY=1"); err != nil {
		logf("sd_notify: %v", err)
	}
	ntfy.StartWatchdog()

	if err := http.Serve(ln, privateNetworkMiddleware(mux)); err != nil {
		logFatal("%v", err)
	}
}
