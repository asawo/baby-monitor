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
	"stream.service":       "monitor/logs/ffmpeg.log",
	"monitor-http.service": "monitor/logs/monitor.log",
	"detect.service":       "monitor/logs/detect.log",
}

// Handlers holds shared dependencies for all API handlers.
type Handlers struct {
	state *state.State
}

// New returns a Handlers wired to the given State.
func New(s *state.State) *Handlers {
	return &Handlers{state: s}
}

// StatusHandler returns the systemd active state of each monitored service.
func (h *Handlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	result := make([]ServiceStatus, len(services))
	for i, svc := range services {
		err := exec.Command("systemctl", "is-active", "--quiet", svc).Run()
		result[i] = ServiceStatus{Name: svc, Active: err == nil}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// LogsHandler returns the last 50 lines of each service log (file or journald).
func (h *Handlers) LogsHandler(w http.ResponseWriter, r *http.Request) {
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
			out, err := exec.Command("journalctl", "--no-pager", "-n", "50", "--output=short-iso", "-u", svc).Output()
			if err != nil {
				content = fmt.Sprintf("(journalctl error: %v)", err)
			} else {
				content = string(out)
			}
		}
		result = append(result, ServiceLog{Name: svc, Content: content})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// NotificationsHandler returns the current notification state (GET) or toggles it (POST).
func (h *Handlers) NotificationsHandler(w http.ResponseWriter, r *http.Request) {
	var enabled bool
	if r.Method == http.MethodPost {
		enabled = h.state.ToggleNotifications()
	} else {
		enabled = h.state.GetNotificationsEnabled()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(NotificationsResponse{Enabled: enabled})
}

// CryHandler returns the most recent cry detection event (GET) or records a new one (POST).
func (h *Handlers) CryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req CryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		h.state.SetCry(req.Confidence)
		w.WriteHeader(http.StatusOK)
		return
	}

	cry := h.state.GetCry()
	w.Header().Set("Content-Type", "application/json")
	if cry.Time.IsZero() {
		_ = json.NewEncoder(w).Encode(CryResponse{})
		return
	}
	secsAgo := int(time.Since(cry.Time).Seconds())
	_ = json.NewEncoder(w).Encode(CryResponse{
		DetectedAt: &cry.Time,
		SecondsAgo: &secsAgo,
		Confidence: &cry.Score,
	})
}

// FartHandler returns the most recent fart detection event (GET) or records a new one (POST).
func (h *Handlers) FartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req FartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		h.state.SetFart(req.Confidence, req.Wetness, req.IsWet)
		w.WriteHeader(http.StatusOK)
		return
	}

	fart := h.state.GetFart()
	w.Header().Set("Content-Type", "application/json")
	if fart.Time.IsZero() {
		_ = json.NewEncoder(w).Encode(FartResponse{})
		return
	}
	secsAgo := int(time.Since(fart.Time).Seconds())
	_ = json.NewEncoder(w).Encode(FartResponse{
		DetectedAt: &fart.Time,
		SecondsAgo: &secsAgo,
		Confidence: &fart.Score,
		Wetness:    &fart.Wetness,
		IsWet:      &fart.IsWet,
	})
}

// DetectStatusHandler returns the current detector error state (GET) or updates it (POST).
func (h *Handlers) DetectStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req DetectStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		h.state.SetDetectError(req.Error)
		w.WriteHeader(http.StatusOK)
		return
	}

	det := h.state.GetDetectError()
	w.Header().Set("Content-Type", "application/json")
	if det.Msg == "" {
		_ = json.NewEncoder(w).Encode(DetectStatusResponse{})
		return
	}
	secsAgo := int(time.Since(det.Time).Seconds())
	_ = json.NewEncoder(w).Encode(DetectStatusResponse{
		Error:      &det.Msg,
		SecondsAgo: &secsAgo,
	})
}
