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
	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
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

func TestBuildProviderRegistryUsesLocalProviderOnly(t *testing.T) {
	local := fakeProvider{id: engineprovider.DefaultProviderID}
	registry, providers, err := buildProviderRegistry(config.ASRConfig{
		LocalProviderEnabled: true,
		DefaultProvider:      engineprovider.DefaultProviderID,
	}, local)
	if err != nil {
		t.Fatalf("buildProviderRegistry returned error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("providers length = %d, want 1", len(providers))
	}
	if registry.DefaultProvider().ID() != engineprovider.DefaultProviderID {
		t.Fatalf("default provider = %q", registry.DefaultProvider().ID())
	}
}

func TestBuildProviderRegistryRejectsMissingLocalProvider(t *testing.T) {
	_, _, err := buildProviderRegistry(config.ASRConfig{
		LocalProviderEnabled: true,
		DefaultProvider:      engineprovider.DefaultProviderID,
	}, nil)
	if err == nil {
		t.Fatal("buildProviderRegistry returned nil error")
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
		ASR: config.ASRConfig{
			NormalizedAudioDir:     filepath.Join(root, "asr-normalized"),
			ProviderAudioMountRoot: "/provider-input/audio",
			LocalProviderEnabled:   true,
			DefaultProvider:        engineprovider.DefaultProviderID,
		},
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

type fakeProvider struct {
	id string
}

func (p fakeProvider) ID() string { return p.id }
func (p fakeProvider) Inspect(ctx context.Context) (*asrcontract.ProviderInfo, error) {
	return &asrcontract.ProviderInfo{Provider: asrcontract.ProviderIdentity{ID: p.id}}, nil
}
func (p fakeProvider) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	return nil, nil
}
func (p fakeProvider) Status(ctx context.Context) (*asrcontract.ProviderStatus, error) {
	return &asrcontract.ProviderStatus{State: asrcontract.ProviderStateIdle}, nil
}
func (p fakeProvider) LoadModel(ctx context.Context, req asrcontract.LoadModelRequest) error {
	return nil
}
func (p fakeProvider) UnloadModel(ctx context.Context, req asrcontract.UnloadModelRequest) error {
	return nil
}
func (p fakeProvider) LoadedModels(ctx context.Context) ([]asrcontract.LoadedModel, error) {
	return nil, nil
}
func (p fakeProvider) ExecuteTask(ctx context.Context, req engineprovider.TaskRequest) (*engineprovider.TaskResult, error) {
	return nil, nil
}
func (p fakeProvider) Close() error { return nil }
