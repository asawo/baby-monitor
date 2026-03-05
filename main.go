package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var services = []string{"mediamtx", "stream.service", "monitor-http.service", "detect.service"}

// logFiles maps service names to their log file paths (relative to home dir).
// Services not listed here log to journald instead.
var logFiles = map[string]string{
	"stream.service":      "monitor/ffmpeg.log",
	"monitor-http.service": "monitor/monitor.log",
	"detect.service":      "monitor/detect.log",
}

type serviceStatus struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

var (
	mu                   sync.Mutex
	notificationsEnabled = true
	lastCryTime          time.Time
	lastCryScore         float64
	detectErrMsg         string
	detectErrTime        time.Time
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	result := make([]serviceStatus, len(services))
	for i, svc := range services {
		err := exec.Command("systemctl", "is-active", "--quiet", svc).Run()
		result[i] = serviceStatus{Name: svc, Active: err == nil}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type serviceLog struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	result := make([]serviceLog, 0, len(services))

	for _, svc := range services {
		var content string
		if rel, ok := logFiles[svc]; ok {
			path := filepath.Join(home, rel)
			out, err := exec.Command("tail", "-n", "50", path).Output()
			if err != nil {
				content = fmt.Sprintf("(log unavailable: %s)", path)
			} else {
				content = string(out)
			}
		} else {
			out, err := exec.Command("journalctl", "--no-pager", "-n", "50", "--output=short", "-u", svc).Output()
			if err != nil {
				content = fmt.Sprintf("(journalctl error: %v)", err)
			} else {
				content = string(out)
			}
		}
		result = append(result, serviceLog{Name: svc, Content: content})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func notificationsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if r.Method == http.MethodPost {
		notificationsEnabled = !notificationsEnabled
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"enabled": notificationsEnabled})
}

func cryHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if r.Method == http.MethodPost {
		var body struct {
			Confidence float64 `json:"confidence"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		lastCryTime = time.Now()
		lastCryScore = body.Confidence
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if lastCryTime.IsZero() {
		json.NewEncoder(w).Encode(map[string]interface{}{"detected_at": nil})
		return
	}
	secsAgo := int(time.Since(lastCryTime).Seconds())
	json.NewEncoder(w).Encode(map[string]interface{}{
		"detected_at": lastCryTime,
		"seconds_ago": secsAgo,
		"confidence":  lastCryScore,
	})
}

func detectStatusHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if r.Method == http.MethodPost {
		var body struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		detectErrMsg = body.Error
		if body.Error != "" {
			detectErrTime = time.Now()
		} else {
			detectErrTime = time.Time{}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if detectErrMsg == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": nil})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":       detectErrMsg,
		"seconds_ago": int(time.Since(detectErrTime).Seconds()),
	})
}

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
	mux.HandleFunc("/api/status", statusHandler)
	mux.HandleFunc("/api/logs", logsHandler)
	mux.HandleFunc("/api/notifications", notificationsHandler)
	mux.HandleFunc("/api/cry", cryHandler)
	mux.HandleFunc("/api/detect-status", detectStatusHandler)
	mux.Handle("/", http.FileServer(http.Dir("./public")))

	addr := ":80"
	log.Printf("Serving on http://0.0.0.0%s\n", addr)

	if err := http.ListenAndServe(addr, privateNetworkMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
