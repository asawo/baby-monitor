package state

import (
	"log"
	"sync"
	"time"

	"babymonitor/server/internal/store"
)

// State holds all in-memory detection state and writes through to a backing store on mutation.
type State struct {
	mu                   sync.Mutex
	db                   *store.DB
	notificationsEnabled bool
	lastCryTime          time.Time
	lastCryScore         float64
	lastFartTime         time.Time
	lastFartScore        float64
	lastFartWetness      float64
	lastFartIsWet        bool
	detectErrMsg         string
	detectErrTime        time.Time
}

// New creates a State, loading any previously persisted values from db.
func New(db *store.DB) *State {
	s := &State{
		db:                   db,
		notificationsEnabled: true,
	}

	if enabled, err := db.GetNotificationsEnabled(); err != nil {
		log.Printf("state: load notifications: %v", err)
	} else {
		s.notificationsEnabled = enabled
	}

	if cry, err := db.GetCry(); err != nil {
		log.Printf("state: load cry: %v", err)
	} else if !cry.Time.IsZero() {
		s.lastCryTime = cry.Time
		s.lastCryScore = cry.Score
	}

	if fart, err := db.GetFart(); err != nil {
		log.Printf("state: load fart: %v", err)
	} else if !fart.Time.IsZero() {
		s.lastFartTime = fart.Time
		s.lastFartScore = fart.Score
		s.lastFartWetness = fart.Wetness
		s.lastFartIsWet = fart.IsWet
	}

	return s
}

// GetNotificationsEnabled returns whether push notifications are currently enabled.
func (s *State) GetNotificationsEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.notificationsEnabled
}

// ToggleNotifications flips the notification enabled state and returns the new value.
func (s *State) ToggleNotifications() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notificationsEnabled = !s.notificationsEnabled
	if err := s.db.SetNotificationsEnabled(s.notificationsEnabled); err != nil {
		log.Printf("state: persist notifications: %v", err)
	}
	return s.notificationsEnabled
}

// SetCry records a cry detection event with the given confidence score.
func (s *State) SetCry(confidence float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCryTime = time.Now()
	s.lastCryScore = confidence
	if err := s.db.SetCry(store.CryRecord{Time: s.lastCryTime, Score: s.lastCryScore}); err != nil {
		log.Printf("state: persist cry: %v", err)
	}
}

// CryState holds the timestamp and confidence score of the most recent cry detection.
type CryState struct {
	Time  time.Time
	Score float64
}

// GetCry returns a snapshot of the most recent cry detection state.
func (s *State) GetCry() CryState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return CryState{Time: s.lastCryTime, Score: s.lastCryScore}
}

// SetFart records a fart detection event with the given confidence score, wetness score, and wet classification.
func (s *State) SetFart(confidence float64, wetness float64, isWet bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastFartTime = time.Now()
	s.lastFartScore = confidence
	s.lastFartWetness = wetness
	s.lastFartIsWet = isWet
	if err := s.db.SetFart(store.FartRecord{
		Time:    s.lastFartTime,
		Score:   s.lastFartScore,
		Wetness: s.lastFartWetness,
		IsWet:   s.lastFartIsWet,
	}); err != nil {
		log.Printf("state: persist fart: %v", err)
	}
}

// FartState holds the timestamp, confidence score, wetness score, and wet classification of the most recent fart detection.
type FartState struct {
	Time    time.Time
	Score   float64
	Wetness float64
	IsWet   bool
}

// GetFart returns a snapshot of the most recent fart detection state.
func (s *State) GetFart() FartState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return FartState{Time: s.lastFartTime, Score: s.lastFartScore, Wetness: s.lastFartWetness, IsWet: s.lastFartIsWet}
}

// SetDetectError records a detector error message. Pass an empty string to clear the error.
func (s *State) SetDetectError(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detectErrMsg = msg
	if msg != "" {
		s.detectErrTime = time.Now()
	} else {
		s.detectErrTime = time.Time{}
	}
}

// DetectErrorState holds the most recent detector error message and when it occurred.
type DetectErrorState struct {
	Msg  string
	Time time.Time
}

// GetDetectError returns a snapshot of the most recent detector error state.
func (s *State) GetDetectError() DetectErrorState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return DetectErrorState{Msg: s.detectErrMsg, Time: s.detectErrTime}
}
