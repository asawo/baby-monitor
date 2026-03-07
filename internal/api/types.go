package api

import "time"

// ServiceStatus is the response shape for each entry in GET /api/status.
type ServiceStatus struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// ServiceLog is the response shape for each entry in GET /api/logs.
type ServiceLog struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// NotificationsResponse is the response for GET and POST /api/notifications.
type NotificationsResponse struct {
	Enabled bool `json:"enabled"`
}

// CryRequest is the request body for POST /api/cry.
type CryRequest struct {
	Confidence float64 `json:"confidence"`
}

// CryResponse is the response for GET /api/cry.
// DetectedAt is null when no cry has been recorded yet.
type CryResponse struct {
	DetectedAt *time.Time `json:"detected_at"`
	SecondsAgo *int       `json:"seconds_ago,omitempty"`
	Confidence *float64   `json:"confidence,omitempty"`
}

// DetectStatusRequest is the request body for POST /api/detect-status.
type DetectStatusRequest struct {
	Error string `json:"error"`
}

// DetectStatusResponse is the response for GET /api/detect-status.
// Error is null when the detector is healthy.
type DetectStatusResponse struct {
	Error      *string `json:"error"`
	SecondsAgo *int    `json:"seconds_ago,omitempty"`
}
