package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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

// Handler holds shared dependencies for all API handlers.
type Handler struct {
	state  *state.State
	logger *log.Logger
}

// New returns a Handler wired to the given State and logger.
func New(s *state.State, l *log.Logger) *Handler {
	return &Handler{state: s, logger: l}
}

// StatusHandler returns the systemd active state of each monitored service.
// Checks run in parallel to minimise latency.
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	result := make([]ServiceStatus, len(services))
	var wg sync.WaitGroup
	for i, svc := range services {
		wg.Add(1)
		go func(i int, svc string) {
			defer wg.Done()
			err := exec.Command("systemctl", "is-active", "--quiet", svc).Run()
			result[i] = ServiceStatus{Name: svc, Active: err == nil}
		}(i, svc)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Printf("api: status encode: %v", err)
	}
}

// LogsHandler returns the last 50 lines of each service log (file or journald).
func (h *Handler) LogsHandler(w http.ResponseWriter, r *http.Request) {
	home, err := os.UserHomeDir()
	if err != nil {
		h.logger.Printf("api: logs: home dir: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	result := make([]ServiceLog, 0, len(services))

	for _, svc := range services {
		var content string
		if rel, ok := logFiles[svc]; ok {
			path := filepath.Join(home, rel)
			out, err := tailFile(path, 50)
			if err != nil {
				content = fmt.Sprintf("(log unavailable: %s)", path)
			} else {
				content = out
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
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Printf("api: logs encode: %v", err)
	}
}

// tailFile returns the last n lines of the file at path.
// It reads up to 64 KB from the end of the file to avoid loading large files in full.
func tailFile(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // read-only file; close error is non-actionable

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := fi.Size()
	if size == 0 {
		return "", nil
	}

	// Read up to 64 KB from the end — sufficient for 50 typical log lines.
	const maxRead = 64 * 1024
	start := size - maxRead
	if start < 0 {
		start = 0
	}
	buf := make([]byte, size-start)
	nr, err := f.ReadAt(buf, start)
	// io.EOF is expected when the file was written to between Stat and ReadAt
	// (the file grew, so we hit the end of the original size window). Treat as
	// a successful partial read.
	if err != nil && err != io.EOF {
		return "", err
	}
	buf = buf[:nr]

	// Split into lines, keep the last n.
	lines := bytes.Split(buf, []byte("\n"))
	// Drop trailing empty entry produced by a final newline.
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return string(bytes.Join(lines, []byte("\n"))) + "\n", nil
}

// GetNotificationsHandler returns the current notification enabled state.
func (h *Handler) GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(NotificationsResponse{Enabled: h.state.GetNotificationsEnabled()}); err != nil {
		h.logger.Printf("api: notifications encode: %v", err)
	}
}

// ToggleNotificationsHandler flips the notification enabled state and returns the new value.
func (h *Handler) ToggleNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(NotificationsResponse{Enabled: h.state.ToggleNotifications()}); err != nil {
		h.logger.Printf("api: notifications encode: %v", err)
	}
}

// GetCryHandler returns the most recent cry detection event.
func (h *Handler) GetCryHandler(w http.ResponseWriter, r *http.Request) {
	cry := h.state.GetCry()
	w.Header().Set("Content-Type", "application/json")
	if cry.Time.IsZero() {
		if err := json.NewEncoder(w).Encode(CryResponse{}); err != nil {
			h.logger.Printf("api: cry encode: %v", err)
		}
		return
	}
	secsAgo := int(time.Since(cry.Time).Seconds())
	if err := json.NewEncoder(w).Encode(CryResponse{
		DetectedAt: &cry.Time,
		SecondsAgo: &secsAgo,
		Confidence: &cry.Score,
	}); err != nil {
		h.logger.Printf("api: cry encode: %v", err)
	}
}

// RecordCryHandler records a new cry detection event.
func (h *Handler) RecordCryHandler(w http.ResponseWriter, r *http.Request) {
	var req CryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	h.state.SetCry(req.Confidence)
	w.WriteHeader(http.StatusOK)
}

// GetFartHandler returns the most recent fart detection event.
func (h *Handler) GetFartHandler(w http.ResponseWriter, r *http.Request) {
	fart := h.state.GetFart()
	w.Header().Set("Content-Type", "application/json")
	if fart.Time.IsZero() {
		if err := json.NewEncoder(w).Encode(FartResponse{}); err != nil {
			h.logger.Printf("api: fart encode: %v", err)
		}
		return
	}
	secsAgo := int(time.Since(fart.Time).Seconds())
	if err := json.NewEncoder(w).Encode(FartResponse{
		DetectedAt: &fart.Time,
		SecondsAgo: &secsAgo,
		Confidence: &fart.Score,
		Wetness:    &fart.Wetness,
		IsWet:      &fart.IsWet,
	}); err != nil {
		h.logger.Printf("api: fart encode: %v", err)
	}
}

// RecordFartHandler records a new fart detection event.
func (h *Handler) RecordFartHandler(w http.ResponseWriter, r *http.Request) {
	var req FartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	h.state.SetFart(req.Confidence, req.Wetness, req.IsWet)
	w.WriteHeader(http.StatusOK)
}

// EventsHandler returns the audit log of recent detection events.
func (h *Handler) EventsHandler(w http.ResponseWriter, r *http.Request) {
	events, err := h.state.GetAuditLog()
	if err != nil {
		h.logger.Printf("api: events: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		h.logger.Printf("api: events encode: %v", err)
	}
}

// GetDetectStatusHandler returns the current detector error state.
func (h *Handler) GetDetectStatusHandler(w http.ResponseWriter, r *http.Request) {
	det := h.state.GetDetectError()
	w.Header().Set("Content-Type", "application/json")
	if det.Msg == "" {
		if err := json.NewEncoder(w).Encode(DetectStatusResponse{}); err != nil {
			h.logger.Printf("api: detect-status encode: %v", err)
		}
		return
	}
	secsAgo := int(time.Since(det.Time).Seconds())
	if err := json.NewEncoder(w).Encode(DetectStatusResponse{
		Error:      &det.Msg,
		SecondsAgo: &secsAgo,
	}); err != nil {
		h.logger.Printf("api: detect-status encode: %v", err)
	}
}

// SetDetectStatusHandler updates the detector error state.
func (h *Handler) SetDetectStatusHandler(w http.ResponseWriter, r *http.Request) {
	var req DetectStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	h.state.SetDetectError(req.Error)
	w.WriteHeader(http.StatusOK)
}
