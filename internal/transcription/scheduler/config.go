package scheduler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const SettingKey = "queue.scheduler"

type Policy string

const (
	PolicyPriority         Policy = "priority"
	PolicyFIFO             Policy = "fifo"
	PolicyWeightedDuration Policy = "weighted_duration"
	PolicyFairShare        Policy = "fair_share"
)

var ErrInvalidConfig = errors.New("invalid scheduler config")

type Config struct {
	Policy               Policy `json:"policy"`
	MaxConcurrentPerUser int    `json:"max_concurrent_per_user,omitempty"`
}

func DefaultConfig() Config {
	return Config{Policy: PolicyPriority}
}

func (c Config) Validate() error {
	if c.MaxConcurrentPerUser < 0 {
		return fmt.Errorf("%w: max_concurrent_per_user cannot be negative", ErrInvalidConfig)
	}
	switch c.Policy {
	case PolicyPriority, PolicyFIFO, PolicyWeightedDuration, PolicyFairShare:
		if c.Policy == PolicyFairShare && c.MaxConcurrentPerUser <= 0 {
			return fmt.Errorf("%w: fair_share requires max_concurrent_per_user", ErrInvalidConfig)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported policy %q", ErrInvalidConfig, c.Policy)
	}
}

func ParseJSON(raw string) (Config, error) {
	if raw == "" {
		return DefaultConfig(), nil
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Config{}, fmt.Errorf("%w: trailing scheduler config data", ErrInvalidConfig)
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func Marshal(config Config) (string, error) {
	if err := config.Validate(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
