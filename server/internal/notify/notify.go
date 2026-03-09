// Package notify provides systemd sd_notify integration.
// All methods are no-ops when the relevant config values are empty/zero,
// so the binary behaves correctly both on the Pi (under systemd) and on a dev machine.
package notify

import (
	"log"
	"net"
	"time"
)

// Notifier sends sd_notify messages to systemd.
type Notifier struct {
	socketPath   string
	watchdogUsec int64
}

// New returns a Notifier configured with the given socket path and watchdog interval.
// Pass the values from config.Config.NotifySocket and config.Config.WatchdogUsec.
func New(socketPath string, watchdogUsec int64) *Notifier {
	return &Notifier{
		socketPath:   socketPath,
		watchdogUsec: watchdogUsec,
	}
}

// Notify sends a state string to the systemd notification socket.
// It is a no-op if socketPath was empty at construction time.
// Handles both filesystem and abstract (@ prefix) socket paths.
func (n *Notifier) Notify(state string) error {
	if n.socketPath == "" {
		return nil
	}
	socketPath := n.socketPath
	// Abstract namespace sockets are prefixed with '@'; net requires '\x00' instead.
	if socketPath[0] == '@' {
		socketPath = "\x00" + socketPath[1:]
	}
	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: socketPath, Net: "unixgram"})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_, err = conn.Write([]byte(state))
	return err
}

// StartWatchdog sends WATCHDOG=1 at half the watchdogUsec interval in a background goroutine.
// It is a no-op if watchdogUsec was zero at construction time.
func (n *Notifier) StartWatchdog() {
	if n.watchdogUsec <= 0 {
		return
	}
	interval := time.Duration(n.watchdogUsec/2) * time.Microsecond
	if interval < time.Second {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := n.Notify("WATCHDOG=1"); err != nil {
				log.Printf("notify: watchdog: %v", err)
			}
		}
	}()
}
