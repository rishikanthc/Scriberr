package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/database"
)

func TestBuildConstructsRouterWithoutStartingListener(t *testing.T) {
	_ = database.Close()
	t.Cleanup(func() { _ = database.Close() })

	root := t.TempDir()
	cfg := testConfig(root)

	application, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := application.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	rec := httptest.NewRecorder()
	application.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ready status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode ready response: %v", err)
	}
	if body["status"] != "ready" || body["database"] != "ok" {
		t.Fatalf("unexpected ready response: %#v", body)
	}
}

func TestServerUsesConfiguredAddressAndRouter(t *testing.T) {
	application := &App{
		Config: &config.Config{Host: "127.0.0.1", Port: "18080"},
		Router: http.NewServeMux(),
	}

	server := application.Server()
	if server.Addr != "127.0.0.1:18080" {
		t.Fatalf("server address = %q", server.Addr)
	}
	if server.Handler != application.Router {
		t.Fatal("server handler does not use the constructed router")
	}
}

func testConfig(root string) *config.Config {
	return &config.Config{
		Host:           "127.0.0.1",
		Port:           "0",
		Environment:    "test",
		AllowedOrigins: []string{"http://localhost:5173"},
		DatabasePath:   filepath.Join(root, "data", "scriberr.db"),
		JWTSecret:      "test-secret",
		UploadDir:      filepath.Join(root, "uploads"),
		TranscriptsDir: filepath.Join(root, "transcripts"),
		TempDir:        filepath.Join(root, "temp"),
		Recordings: config.RecordingConfig{
			Dir:                   filepath.Join(root, "recordings"),
			MaxChunkBytes:         1 << 20,
			MaxSessionBytes:       1 << 24,
			MaxDuration:           time.Hour,
			SessionTTL:            time.Hour,
			FinalizerWorkers:      1,
			FinalizerPollInterval: time.Second,
			FinalizerLeaseTimeout: time.Minute,
			CleanupInterval:       time.Minute,
			FailedRetention:       time.Hour,
			AllowedMimeTypes:      []string{"audio/webm"},
		},
		Engine: config.EngineConfig{
			CacheDir:     filepath.Join(root, "models"),
			Provider:     "auto",
			MaxLoaded:    1,
			AutoDownload: false,
		},
		Worker: config.WorkerConfig{
			Workers:      1,
			PollInterval: time.Second,
			LeaseTimeout: time.Minute,
		},
	}
}
