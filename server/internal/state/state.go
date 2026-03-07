package state

import (
	"sync"
	"time"
)

var (
	mu                   sync.Mutex
	notificationsEnabled = true
	lastCryTime          time.Time
	lastCryScore         float64
	detectErrMsg         string
	detectErrTime        time.Time
)

// GetNotificationsEnabled returns whether push notifications are currently enabled.
func GetNotificationsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return notificationsEnabled
}

// ToggleNotifications flips the notification enabled state and returns the new value.
func ToggleNotifications() bool {
	mu.Lock()
	defer mu.Unlock()
	notificationsEnabled = !notificationsEnabled
	return notificationsEnabled
}

// SetCry records a cry detection event with the given confidence score.
func SetCry(confidence float64) {
	mu.Lock()
	defer mu.Unlock()
	lastCryTime = time.Now()
	lastCryScore = confidence
}

// CryState holds the timestamp and confidence score of the most recent cry detection.
type CryState struct {
	Time  time.Time
	Score float64
}

// GetCry returns a snapshot of the most recent cry detection state.
func GetCry() CryState {
	mu.Lock()
	defer mu.Unlock()
	return CryState{Time: lastCryTime, Score: lastCryScore}
}

// SetDetectError records a detector error message. Pass an empty string to clear the error.
func SetDetectError(msg string) {
	mu.Lock()
	defer mu.Unlock()
	detectErrMsg = msg
	if msg != "" {
		detectErrTime = time.Now()
	} else {
		detectErrTime = time.Time{}
	}
}

// DetectErrorState holds the most recent detector error message and when it occurred.
type DetectErrorState struct {
	Msg  string
	Time time.Time
}

// GetDetectError returns a snapshot of the most recent detector error state.
func GetDetectError() DetectErrorState {
	mu.Lock()
	defer mu.Unlock()
	return DetectErrorState{Msg: detectErrMsg, Time: detectErrTime}
}
