package main

import (
	"log"
	"net/http"

	"babymonitor/server/internal/api"
)

func privateNetworkMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Access-Control-Request-Private-Network") == "true" {
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", api.StatusHandler)
	mux.HandleFunc("/api/logs", api.LogsHandler)
	mux.HandleFunc("/api/notifications", api.NotificationsHandler)
	mux.HandleFunc("/api/cry", api.CryHandler)
	mux.HandleFunc("/api/detect-status", api.DetectStatusHandler)
	mux.Handle("/", http.FileServer(http.Dir("./server/web")))

	addr := ":80"
	log.Printf("Serving on http://0.0.0.0%s\n", addr)

	if err := http.ListenAndServe(addr, privateNetworkMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
