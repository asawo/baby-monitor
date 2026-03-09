package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"babymonitor/server/internal/api"
	"babymonitor/server/internal/config"
	"babymonitor/server/internal/middleware"
	"babymonitor/server/internal/notify"
	"babymonitor/server/internal/state"
	"babymonitor/server/internal/store"
)

func main() {
	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("config: %v", err)
	}

	db, err := store.New(cfg.DBPath, logger)
	if err != nil {
		logger.Fatalf("store: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Printf("store close: %v", err)
		}
	}()

	s := state.New(db, logger)
	h := api.New(s, logger)
	ntfy := notify.New(cfg.NotifySocket, cfg.WatchdogUsec, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			logger.Printf("healthz write: %v", err)
		}
	})
	mux.HandleFunc("GET /api/status", h.StatusHandler)
	mux.HandleFunc("GET /api/logs", h.LogsHandler)
	mux.HandleFunc("GET /api/notifications", h.GetNotificationsHandler)
	mux.HandleFunc("POST /api/notifications", h.ToggleNotificationsHandler)
	mux.HandleFunc("GET /api/cry", h.GetCryHandler)
	mux.HandleFunc("POST /api/cry", h.RecordCryHandler)
	mux.HandleFunc("GET /api/fart", h.GetFartHandler)
	mux.HandleFunc("POST /api/fart", h.RecordFartHandler)
	mux.HandleFunc("GET /api/detect-status", h.GetDetectStatusHandler)
	mux.HandleFunc("POST /api/detect-status", h.SetDetectStatusHandler)
	mux.HandleFunc("GET /api/events", h.EventsHandler)
	mux.Handle("/", middleware.NoCache(http.FileServer(http.Dir("./server/web"))))

	addr := ":80"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}

	logger.Printf("Serving on http://0.0.0.0%s", addr)

	if err := ntfy.Notify("READY=1"); err != nil {
		logger.Printf("sd_notify: %v", err)
	}
	ntfy.StartWatchdog()

	if err := http.Serve(ln, middleware.PrivateNetwork(mux)); err != nil {
		logger.Fatalf("%v", err)
	}
}
