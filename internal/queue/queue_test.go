package queue

import (
	"os"
	"testing"
)

func TestGetOptimalWorkerCount_DefaultBehavior(t *testing.T) {
	os.Unsetenv("QUEUE_WORKERS")

	min, max := getOptimalWorkerCount()

	// Without QUEUE_WORKERS, should return CPU-based values (non-zero)
	if min <= 0 || max <= 0 {
		t.Errorf("expected positive worker counts, got min=%d, max=%d", min, max)
	}
	if min > max {
		t.Errorf("min (%d) should be <= max (%d)", min, max)
	}
}

func TestGetOptimalWorkerCount_RespectsEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantMin  int
		wantMax  int
	}{
		{"single worker", "1", 1, 1},
		{"four workers", "4", 4, 4},
		{"ten workers", "10", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("QUEUE_WORKERS", tt.envValue)
			defer os.Unsetenv("QUEUE_WORKERS")

			min, max := getOptimalWorkerCount()
			if min != tt.wantMin || max != tt.wantMax {
				t.Errorf("QUEUE_WORKERS=%s: got min=%d, max=%d; want min=%d, max=%d",
					tt.envValue, min, max, tt.wantMin, tt.wantMax)
			}
		})
	}
}

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
			os.Setenv("QUEUE_WORKERS", tt.envValue)
			defer os.Unsetenv("QUEUE_WORKERS")

			min, max := getOptimalWorkerCount()
			// Should fall back to CPU-based defaults
			if min <= 0 || max <= 0 {
				t.Errorf("QUEUE_WORKERS=%s: expected positive defaults, got min=%d, max=%d",
					tt.envValue, min, max)
			}
		})
	}
}

// TestNewTaskQueue_DefaultWorkerCount verifies that without QUEUE_WORKERS,
// the legacy parameter (2) is used as default - preserving existing behavior.
func TestNewTaskQueue_DefaultWorkerCount(t *testing.T) {
	os.Unsetenv("QUEUE_WORKERS")

	tq := NewTaskQueue(2, nil, nil)
	defer tq.cancel()

	if tq.minWorkers != 2 {
		t.Errorf("without QUEUE_WORKERS, expected minWorkers=2, got %d", tq.minWorkers)
	}
	if tq.maxWorkers != 2 {
		t.Errorf("without QUEUE_WORKERS, expected maxWorkers=2, got %d", tq.maxWorkers)
	}
}

// TestNewTaskQueue_EnvOverridesLegacy verifies that QUEUE_WORKERS takes
// precedence over the hardcoded legacy parameter.
// This is the core bug test - currently QUEUE_WORKERS is ignored.
func TestNewTaskQueue_EnvOverridesLegacy(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		legacyWorkers int
		wantMin       int
		wantMax       int
	}{
		{"env=1 overrides legacy=2", "1", 2, 1, 1},
		{"env=4 overrides legacy=2", "4", 2, 4, 4},
		{"env=3 overrides legacy=2", "3", 2, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("QUEUE_WORKERS", tt.envValue)
			defer os.Unsetenv("QUEUE_WORKERS")

			tq := NewTaskQueue(tt.legacyWorkers, nil, nil)
			defer tq.cancel()

			if tq.minWorkers != tt.wantMin {
				t.Errorf("QUEUE_WORKERS=%s with legacy=%d: got minWorkers=%d, want %d",
					tt.envValue, tt.legacyWorkers, tq.minWorkers, tt.wantMin)
			}
			if tq.maxWorkers != tt.wantMax {
				t.Errorf("QUEUE_WORKERS=%s with legacy=%d: got maxWorkers=%d, want %d",
					tt.envValue, tt.legacyWorkers, tq.maxWorkers, tt.wantMax)
			}
		})
	}
}

// TestNewTaskQueue_AutoScaleDisabledWithFixedWorkers verifies that
// auto-scaling is disabled when QUEUE_WORKERS sets a fixed count.
func TestNewTaskQueue_AutoScaleDisabledWithFixedWorkers(t *testing.T) {
	os.Setenv("QUEUE_WORKERS", "3")
	defer os.Unsetenv("QUEUE_WORKERS")

	tq := NewTaskQueue(2, nil, nil)
	defer tq.cancel()

	if tq.autoScale {
		t.Error("auto-scaling should be disabled when QUEUE_WORKERS sets fixed worker count")
	}
}
