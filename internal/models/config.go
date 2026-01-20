package models

import (
	"strconv"
)

// Config represents a key-value configuration entry
type Config struct {
	Key   string
	Value string
}

// Configuration keys
const (
	ConfigKeyMaxDailyEffort = "max_daily_effort"
)

// Default configuration values
const (
	DefaultMaxDailyEffort = 10
)

// AsInt returns the config value as an integer
// Returns 0 and an error if the value cannot be parsed
func (c *Config) AsInt() (int, error) {
	return strconv.Atoi(c.Value)
}

// AsString returns the config value as a string
func (c *Config) AsString() string {
	return c.Value
}
