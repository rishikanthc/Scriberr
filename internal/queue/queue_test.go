package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetOptimalWorkerCount_DefaultBehavior verifies that without QUEUE_WORKERS,
// the function returns positive CPU-based defaults with min <= max.
func TestGetOptimalWorkerCount_DefaultBehavior(t *testing.T) {
	t.Setenv("QUEUE_WORKERS", "")

	min, max := getOptimalWorkerCount()

	assert.Positive(t, min, "min workers should be positive")
	assert.Positive(t, max, "max workers should be positive")
	assert.LessOrEqual(t, min, max, "min should be <= max")
}

// TestGetOptimalWorkerCount_RespectsEnvVar verifies that QUEUE_WORKERS env var
// pins both min and max to the exact value specified.
func TestGetOptimalWorkerCount_RespectsEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     int
	}{
		{"single worker", "1", 1},
		{"four workers", "4", 4},
		{"ten workers", "10", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("QUEUE_WORKERS", tt.envValue)

			min, max := getOptimalWorkerCount()

			assert.Equal(t, tt.want, min, "min workers")
			assert.Equal(t, tt.want, max, "max workers")
		})
	}
}

// TestGetOptimalWorkerCount_IgnoresInvalidEnvVar verifies that non-numeric,
// zero, and negative QUEUE_WORKERS values are ignored, falling back to
// CPU-based defaults rather than crashing or using bad values.
func TestGetOptimalWorkerCount_IgnoresInvalidEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{"non-numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("QUEUE_WORKERS", tt.envValue)

			min, max := getOptimalWorkerCount()

			assert.Positive(t, min, "should fall back to positive CPU-based default")
			assert.Positive(t, max, "should fall back to positive CPU-based default")
		})
	}
}

// TestNewTaskQueue_DefaultWorkerCount verifies that without QUEUE_WORKERS,
// the legacy parameter (2) is used as default, preserving existing behavior.
func TestNewTaskQueue_DefaultWorkerCount(t *testing.T) {
	t.Setenv("QUEUE_WORKERS", "")

	tq := NewTaskQueue(2, nil, nil)
	defer tq.cancel()

	assert.Equal(t, 2, tq.minWorkers, "default minWorkers should match legacy parameter")
	assert.Equal(t, 2, tq.maxWorkers, "default maxWorkers should match legacy parameter")
}

// TestNewTaskQueue_EnvOverridesLegacy verifies that QUEUE_WORKERS takes
// precedence over the hardcoded legacy parameter.
func TestNewTaskQueue_EnvOverridesLegacy(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		legacyWorkers int
		want          int
	}{
		{"env=1 overrides legacy=2", "1", 2, 1},
		{"env=4 overrides legacy=2", "4", 2, 4},
		{"env=3 overrides legacy=2", "3", 2, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("QUEUE_WORKERS", tt.envValue)

			tq := NewTaskQueue(tt.legacyWorkers, nil, nil)
			defer tq.cancel()

			assert.Equal(t, tt.want, tq.minWorkers, "QUEUE_WORKERS should override legacy minWorkers")
			assert.Equal(t, tt.want, tq.maxWorkers, "QUEUE_WORKERS should override legacy maxWorkers")
		})
	}
}

// TestNewTaskQueue_AutoScaleDisabledWithFixedWorkers verifies that
// auto-scaling is disabled when QUEUE_WORKERS sets a fixed count.
func TestNewTaskQueue_AutoScaleDisabledWithFixedWorkers(t *testing.T) {
	t.Setenv("QUEUE_WORKERS", "3")

	tq := NewTaskQueue(2, nil, nil)
	defer tq.cancel()

	assert.False(t, tq.autoScale, "auto-scaling should be disabled when QUEUE_WORKERS sets fixed count")
}
