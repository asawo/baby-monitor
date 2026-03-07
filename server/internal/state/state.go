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

func GetNotificationsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return notificationsEnabled
}

func ToggleNotifications() bool {
	mu.Lock()
	defer mu.Unlock()
	notificationsEnabled = !notificationsEnabled
	return notificationsEnabled
}

func SetCry(confidence float64) {
	mu.Lock()
	defer mu.Unlock()
	lastCryTime = time.Now()
	lastCryScore = confidence
}

type CryState struct {
	Time  time.Time
	Score float64
}

func GetCry() CryState {
	mu.Lock()
	defer mu.Unlock()
	return CryState{Time: lastCryTime, Score: lastCryScore}
}

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

type DetectErrorState struct {
	Msg  string
	Time time.Time
}

func GetDetectError() DetectErrorState {
	mu.Lock()
	defer mu.Unlock()
	return DetectErrorState{Msg: detectErrMsg, Time: detectErrTime}
}
