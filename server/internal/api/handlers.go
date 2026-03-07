package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"babymonitor/server/internal/state"
)

var services = []string{"mediamtx", "stream.service", "monitor-http.service", "detect.service"}

// logFiles maps service names to their log file paths (relative to home dir).
// Services not listed here log to journald instead.
var logFiles = map[string]string{
	"stream.service":       "monitor/ffmpeg.log",
	"monitor-http.service": "monitor/monitor.log",
	"detect.service":       "monitor/detect.log",
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	result := make([]ServiceStatus, len(services))
	for i, svc := range services {
		err := exec.Command("systemctl", "is-active", "--quiet", svc).Run()
		result[i] = ServiceStatus{Name: svc, Active: err == nil}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	result := make([]ServiceLog, 0, len(services))

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
		result = append(result, ServiceLog{Name: svc, Content: content})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func NotificationsHandler(w http.ResponseWriter, r *http.Request) {
	var enabled bool
	if r.Method == http.MethodPost {
		enabled = state.ToggleNotifications()
	} else {
		enabled = state.GetNotificationsEnabled()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NotificationsResponse{Enabled: enabled})
}

func CryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req CryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		state.SetCry(req.Confidence)
		w.WriteHeader(http.StatusOK)
		return
	}

	cry := state.GetCry()
	w.Header().Set("Content-Type", "application/json")
	if cry.Time.IsZero() {
		json.NewEncoder(w).Encode(CryResponse{})
		return
	}
	secsAgo := int(time.Since(cry.Time).Seconds())
	json.NewEncoder(w).Encode(CryResponse{
		DetectedAt: &cry.Time,
		SecondsAgo: &secsAgo,
		Confidence: &cry.Score,
	})
}

func DetectStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req DetectStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		state.SetDetectError(req.Error)
		w.WriteHeader(http.StatusOK)
		return
	}

	det := state.GetDetectError()
	w.Header().Set("Content-Type", "application/json")
	if det.Msg == "" {
		json.NewEncoder(w).Encode(DetectStatusResponse{})
		return
	}
	secsAgo := int(time.Since(det.Time).Seconds())
	json.NewEncoder(w).Encode(DetectStatusResponse{
		Error:      &det.Msg,
		SecondsAgo: &secsAgo,
	})
}
