package config

import (
	"strings"
	"testing"
	"time"
)

func setConfigTestBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("APP_ENV", "")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("SPEECH_ENGINE_CACHE_DIR", "")
	t.Setenv("SPEECH_ENGINE_PROVIDER", "")
	t.Setenv("SPEECH_ENGINE_THREADS", "")
	t.Setenv("SPEECH_ENGINE_MAX_LOADED", "")
	t.Setenv("SPEECH_ENGINE_AUTO_DOWNLOAD", "")
	t.Setenv("TRANSCRIPTION_WORKERS", "")
	t.Setenv("TRANSCRIPTION_QUEUE_POLL_INTERVAL", "")
	t.Setenv("TRANSCRIPTION_LEASE_TIMEOUT", "")
}

func TestLoadWithErrorEngineAndWorkerDefaults(t *testing.T) {
	setConfigTestBaseEnv(t)

	cfg, err := LoadWithError()
	if err != nil {
		t.Fatalf("LoadWithError returned error: %v", err)
	}

	if cfg.Engine.CacheDir != "data/models" {
		t.Fatalf("Engine.CacheDir = %q, want data/models", cfg.Engine.CacheDir)
	}
	if cfg.Engine.Provider != "auto" {
		t.Fatalf("Engine.Provider = %q, want auto", cfg.Engine.Provider)
	}
	if cfg.Engine.Threads != 0 {
		t.Fatalf("Engine.Threads = %d, want 0", cfg.Engine.Threads)
	}
	if cfg.Engine.MaxLoaded != 2 {
		t.Fatalf("Engine.MaxLoaded = %d, want 2", cfg.Engine.MaxLoaded)
	}
	if !cfg.Engine.AutoDownload {
		t.Fatalf("Engine.AutoDownload = false, want true")
	}
	if cfg.Worker.Workers != 1 {
		t.Fatalf("Worker.Workers = %d, want 1", cfg.Worker.Workers)
	}
	if cfg.Worker.PollInterval != 2*time.Second {
		t.Fatalf("Worker.PollInterval = %s, want 2s", cfg.Worker.PollInterval)
	}
	if cfg.Worker.LeaseTimeout != 10*time.Minute {
		t.Fatalf("Worker.LeaseTimeout = %s, want 10m", cfg.Worker.LeaseTimeout)
	}
}

func TestLoadWithErrorEngineAndWorkerOverrides(t *testing.T) {
	setConfigTestBaseEnv(t)
	t.Setenv("SPEECH_ENGINE_CACHE_DIR", "/tmp/scriberr-models")
	t.Setenv("SPEECH_ENGINE_PROVIDER", "cuda")
	t.Setenv("SPEECH_ENGINE_THREADS", "4")
	t.Setenv("SPEECH_ENGINE_MAX_LOADED", "3")
	t.Setenv("SPEECH_ENGINE_AUTO_DOWNLOAD", "false")
	t.Setenv("TRANSCRIPTION_WORKERS", "2")
	t.Setenv("TRANSCRIPTION_QUEUE_POLL_INTERVAL", "500ms")
	t.Setenv("TRANSCRIPTION_LEASE_TIMEOUT", "30s")

	cfg, err := LoadWithError()
	if err != nil {
		t.Fatalf("LoadWithError returned error: %v", err)
	}

	if cfg.Engine.CacheDir != "/tmp/scriberr-models" {
		t.Fatalf("Engine.CacheDir = %q", cfg.Engine.CacheDir)
	}
	if cfg.Engine.Provider != "cuda" {
		t.Fatalf("Engine.Provider = %q", cfg.Engine.Provider)
	}
	if cfg.Engine.Threads != 4 {
		t.Fatalf("Engine.Threads = %d", cfg.Engine.Threads)
	}
	if cfg.Engine.MaxLoaded != 3 {
		t.Fatalf("Engine.MaxLoaded = %d", cfg.Engine.MaxLoaded)
	}
	if cfg.Engine.AutoDownload {
		t.Fatalf("Engine.AutoDownload = true, want false")
	}
	if cfg.Worker.Workers != 2 {
		t.Fatalf("Worker.Workers = %d", cfg.Worker.Workers)
	}
	if cfg.Worker.PollInterval != 500*time.Millisecond {
		t.Fatalf("Worker.PollInterval = %s", cfg.Worker.PollInterval)
	}
	if cfg.Worker.LeaseTimeout != 30*time.Second {
		t.Fatalf("Worker.LeaseTimeout = %s", cfg.Worker.LeaseTimeout)
	}
}

func TestLoadWithErrorRejectsInvalidEngineProvider(t *testing.T) {
	setConfigTestBaseEnv(t)
	t.Setenv("SPEECH_ENGINE_PROVIDER", "metal")

	_, err := LoadWithError()
	if err == nil {
		t.Fatal("LoadWithError returned nil error for invalid provider")
	}
	if !strings.Contains(err.Error(), "SPEECH_ENGINE_PROVIDER") {
		t.Fatalf("error %q does not mention SPEECH_ENGINE_PROVIDER", err.Error())
	}
}

func TestLoadWithErrorRejectsInvalidEngineAndWorkerValues(t *testing.T) {
	tests := []struct {
		name string
		env  string
		val  string
	}{
		{name: "engine threads", env: "SPEECH_ENGINE_THREADS", val: "fast"},
		{name: "engine max loaded", env: "SPEECH_ENGINE_MAX_LOADED", val: "many"},
		{name: "auto download", env: "SPEECH_ENGINE_AUTO_DOWNLOAD", val: "sometimes"},
		{name: "workers", env: "TRANSCRIPTION_WORKERS", val: "two"},
		{name: "poll interval", env: "TRANSCRIPTION_QUEUE_POLL_INTERVAL", val: "quickly"},
		{name: "lease timeout", env: "TRANSCRIPTION_LEASE_TIMEOUT", val: "later"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setConfigTestBaseEnv(t)
			t.Setenv(tt.env, tt.val)

			_, err := LoadWithError()
			if err == nil {
				t.Fatalf("LoadWithError returned nil error for %s=%q", tt.env, tt.val)
			}
			if !strings.Contains(err.Error(), tt.env) {
				t.Fatalf("error %q does not mention %s", err.Error(), tt.env)
			}
		})
	}
}
