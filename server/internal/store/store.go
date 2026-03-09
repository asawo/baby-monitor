package store

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.etcd.io/bbolt"
)

var (
	bucketState = []byte("state")
	bucketAudit = []byte("audit")

	keyNotifications = []byte("notifications_enabled")
	keyLastCry       = []byte("last_cry")
	keyLastFart      = []byte("last_fart")
)

const maxAuditEntries = 50

// DB wraps a BoltDB instance with typed accessors for monitor state.
type DB struct {
	db     *bbolt.DB
	logger *log.Logger
}

// New opens (or creates) the BoltDB file at path and initializes buckets.
// Uses a 1s timeout to avoid hanging if the file is already locked.
func New(path string, log *log.Logger) (*DB, error) {
	bdb, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	err = bdb.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucketState); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(bucketAudit); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		_ = bdb.Close()
		return nil, fmt.Errorf("init buckets: %w", err)
	}
	return &DB{db: bdb, logger: log}, nil
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}

// GetNotificationsEnabled returns the persisted value, defaulting to true if not yet set.
func (d *DB) GetNotificationsEnabled() (bool, error) {
	var result bool
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(bucketState).Get(keyNotifications)
		if v == nil {
			result = true // match the in-memory default
			return nil
		}
		return json.Unmarshal(v, &result)
	})
	return result, err
}

// SetNotificationsEnabled persists the notifications toggle state.
func (d *DB) SetNotificationsEnabled(v bool) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return d.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketState).Put(keyNotifications, data)
	})
}

// CryRecord is the persisted shape of a cry detection event.
type CryRecord struct {
	Time  time.Time `json:"time"`
	Score float64   `json:"score"`
}

// GetCry returns the last persisted cry record, or a zero-value CryRecord if none.
func (d *DB) GetCry() (CryRecord, error) {
	var r CryRecord
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(bucketState).Get(keyLastCry)
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &r)
	})
	return r, err
}

// SetCry persists a cry event and appends an audit entry.
func (d *DB) SetCry(r CryRecord) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if err := tx.Bucket(bucketState).Put(keyLastCry, data); err != nil {
			return err
		}
		return appendAudit(tx, AuditEvent{
			Type:  "cry",
			Time:  r.Time,
			Score: r.Score,
		})
	})
}

// FartRecord is the persisted shape of a fart detection event.
type FartRecord struct {
	Time    time.Time `json:"time"`
	Score   float64   `json:"score"`
	Wetness float64   `json:"wetness"`
	IsWet   bool      `json:"is_wet"`
}

// GetFart returns the last persisted fart record, or a zero-value FartRecord if none.
func (d *DB) GetFart() (FartRecord, error) {
	var r FartRecord
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(bucketState).Get(keyLastFart)
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &r)
	})
	return r, err
}

// SetFart persists a fart event and appends an audit entry.
func (d *DB) SetFart(r FartRecord) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if err := tx.Bucket(bucketState).Put(keyLastFart, data); err != nil {
			return err
		}
		return appendAudit(tx, AuditEvent{
			Type:    "fart",
			Time:    r.Time,
			Score:   r.Score,
			Wetness: r.Wetness,
			IsWet:   r.IsWet,
		})
	})
}

// AuditEvent is one entry in the rolling audit log.
type AuditEvent struct {
	Type    string    `json:"type"`
	Time    time.Time `json:"time"`
	Score   float64   `json:"score"`
	Wetness float64   `json:"wetness,omitempty"`
	IsWet   bool      `json:"is_wet,omitempty"`
}

// GetAuditLog returns up to maxAuditEntries most recent audit events in chronological order.
func (d *DB) GetAuditLog() ([]AuditEvent, error) {
	var events []AuditEvent
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAudit)
		c := b.Cursor()
		// Seek to last entry and walk backwards to collect up to maxAuditEntries.
		raw := make([][]byte, 0, maxAuditEntries)
		for k, v := c.Last(); k != nil && len(raw) < maxAuditEntries; k, v = c.Prev() {
			cp := make([]byte, len(v))
			copy(cp, v)
			raw = append(raw, cp)
		}
		// Reverse to restore chronological order.
		for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
			raw[i], raw[j] = raw[j], raw[i]
		}
		for _, v := range raw {
			var e AuditEvent
			if err := json.Unmarshal(v, &e); err != nil {
				d.logger.Printf("store: audit unmarshal: %v", err)
				continue
			}
			events = append(events, e)
		}
		return nil
	})
	return events, err
}

// appendAudit adds an event to the audit bucket and prunes to maxAuditEntries.
// Must be called inside an Update transaction.
func appendAudit(tx *bbolt.Tx, e AuditEvent) error {
	b := tx.Bucket(bucketAudit)
	seq, err := b.NextSequence()
	if err != nil {
		return err
	}
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, seq)
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if err := b.Put(key, data); err != nil {
		return err
	}
	// Prune oldest entry if over limit. Stats().KeyN reflects the count before
	// the current transaction commits, so the actual count after Put is KeyN+1;
	// compare with >= to catch the moment we hit the limit.
	if b.Stats().KeyN >= maxAuditEntries {
		k, _ := b.Cursor().First()
		if k != nil {
			return b.Delete(k)
		}
	}
	return nil
}
