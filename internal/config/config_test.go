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
	t.Setenv("RECORDINGS_DIR", "")
	t.Setenv("RECORDING_MAX_CHUNK_BYTES", "")
	t.Setenv("RECORDING_MAX_DURATION", "")
	t.Setenv("RECORDING_SESSION_TTL", "")
	t.Setenv("RECORDING_FINALIZER_WORKERS", "")
	t.Setenv("RECORDING_FINALIZER_POLL_INTERVAL", "")
	t.Setenv("RECORDING_FINALIZER_LEASE_TIMEOUT", "")
	t.Setenv("RECORDING_ALLOWED_MIME_TYPES", "")
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
	if cfg.Recordings.Dir != "data/recordings" {
		t.Fatalf("Recordings.Dir = %q, want data/recordings", cfg.Recordings.Dir)
	}
	if cfg.Recordings.MaxChunkBytes != 25<<20 {
		t.Fatalf("Recordings.MaxChunkBytes = %d, want %d", cfg.Recordings.MaxChunkBytes, int64(25<<20))
	}
	if cfg.Recordings.MaxDuration != 8*time.Hour {
		t.Fatalf("Recordings.MaxDuration = %s, want 8h", cfg.Recordings.MaxDuration)
	}
	if cfg.Recordings.SessionTTL != 12*time.Hour {
		t.Fatalf("Recordings.SessionTTL = %s, want 12h", cfg.Recordings.SessionTTL)
	}
	if cfg.Recordings.FinalizerWorkers != 1 {
		t.Fatalf("Recordings.FinalizerWorkers = %d, want 1", cfg.Recordings.FinalizerWorkers)
	}
	if cfg.Recordings.FinalizerPollInterval != 2*time.Second {
		t.Fatalf("Recordings.FinalizerPollInterval = %s, want 2s", cfg.Recordings.FinalizerPollInterval)
	}
	if cfg.Recordings.FinalizerLeaseTimeout != 10*time.Minute {
		t.Fatalf("Recordings.FinalizerLeaseTimeout = %s, want 10m", cfg.Recordings.FinalizerLeaseTimeout)
	}
	if got := strings.Join(cfg.Recordings.AllowedMimeTypes, ","); got != "audio/webm;codecs=opus,audio/webm" {
		t.Fatalf("Recordings.AllowedMimeTypes = %q", got)
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
	t.Setenv("RECORDINGS_DIR", "/tmp/scriberr-recordings")
	t.Setenv("RECORDING_MAX_CHUNK_BYTES", "1048576")
	t.Setenv("RECORDING_MAX_DURATION", "2h")
	t.Setenv("RECORDING_SESSION_TTL", "3h")
	t.Setenv("RECORDING_FINALIZER_WORKERS", "2")
	t.Setenv("RECORDING_FINALIZER_POLL_INTERVAL", "750ms")
	t.Setenv("RECORDING_FINALIZER_LEASE_TIMEOUT", "45s")
	t.Setenv("RECORDING_ALLOWED_MIME_TYPES", "audio/webm;codecs=opus, audio/ogg")

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
	if cfg.Recordings.Dir != "/tmp/scriberr-recordings" {
		t.Fatalf("Recordings.Dir = %q", cfg.Recordings.Dir)
	}
	if cfg.Recordings.MaxChunkBytes != 1048576 {
		t.Fatalf("Recordings.MaxChunkBytes = %d", cfg.Recordings.MaxChunkBytes)
	}
	if cfg.Recordings.MaxDuration != 2*time.Hour {
		t.Fatalf("Recordings.MaxDuration = %s", cfg.Recordings.MaxDuration)
	}
	if cfg.Recordings.SessionTTL != 3*time.Hour {
		t.Fatalf("Recordings.SessionTTL = %s", cfg.Recordings.SessionTTL)
	}
	if cfg.Recordings.FinalizerWorkers != 2 {
		t.Fatalf("Recordings.FinalizerWorkers = %d", cfg.Recordings.FinalizerWorkers)
	}
	if cfg.Recordings.FinalizerPollInterval != 750*time.Millisecond {
		t.Fatalf("Recordings.FinalizerPollInterval = %s", cfg.Recordings.FinalizerPollInterval)
	}
	if cfg.Recordings.FinalizerLeaseTimeout != 45*time.Second {
		t.Fatalf("Recordings.FinalizerLeaseTimeout = %s", cfg.Recordings.FinalizerLeaseTimeout)
	}
	if got := strings.Join(cfg.Recordings.AllowedMimeTypes, ","); got != "audio/webm;codecs=opus,audio/ogg" {
		t.Fatalf("Recordings.AllowedMimeTypes = %q", got)
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
		{name: "recording max chunk bytes", env: "RECORDING_MAX_CHUNK_BYTES", val: "large"},
		{name: "recording max duration", env: "RECORDING_MAX_DURATION", val: "forever"},
		{name: "recording ttl", env: "RECORDING_SESSION_TTL", val: "soon"},
		{name: "recording finalizer workers", env: "RECORDING_FINALIZER_WORKERS", val: "many"},
		{name: "recording poll interval", env: "RECORDING_FINALIZER_POLL_INTERVAL", val: "often"},
		{name: "recording lease timeout", env: "RECORDING_FINALIZER_LEASE_TIMEOUT", val: "eventually"},
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

func TestLoadWithErrorRejectsInvalidRecordingValues(t *testing.T) {
	tests := []struct {
		name string
		env  string
		val  string
	}{
		{name: "zero chunk bytes", env: "RECORDING_MAX_CHUNK_BYTES", val: "0"},
		{name: "zero workers", env: "RECORDING_FINALIZER_WORKERS", val: "0"},
		{name: "empty mime list", env: "RECORDING_ALLOWED_MIME_TYPES", val: " , "},
		{name: "video mime", env: "RECORDING_ALLOWED_MIME_TYPES", val: "video/webm"},
		{name: "newline mime", env: "RECORDING_ALLOWED_MIME_TYPES", val: "audio/webm\nx"},
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
