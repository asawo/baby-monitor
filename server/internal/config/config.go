// Package config loads application configuration from environment variables.
package config

import "github.com/kelseyhightower/envconfig"

// Config holds all environment-driven configuration for the monitor server.
type Config struct {
	// DBPath is the path to the BoltDB state file.
	DBPath string `envconfig:"DB_PATH" default:"./monitor.db"`

	// NotifySocket is the systemd NOTIFY_SOCKET path (set by systemd when Type=notify).
	NotifySocket string `envconfig:"NOTIFY_SOCKET"`

	// WatchdogUsec is the systemd WATCHDOG_USEC value in microseconds (set by systemd).
	WatchdogUsec int64 `envconfig:"WATCHDOG_USEC"`
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
