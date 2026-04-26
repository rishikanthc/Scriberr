package engineprovider

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appconfig "scriberr/internal/config"

	speechengine "scriberr-engine/speech/engine"
)

func TestRealEngineJFKTranscription(t *testing.T) {
	requireRealEngineIntegration(t)
	requireFFmpeg(t)

	audioPath := requireJFKFixture(t)
	cacheDir := realEngineCacheDir(t)
	provider := newRealTestProvider(t, cacheDir, true)
	start := time.Now()

	result, err := provider.Transcribe(context.Background(), TranscriptionRequest{
		JobID:     "real-jfk",
		UserID:    1,
		AudioPath: audioPath,
		ModelID:   DefaultTranscriptionModel,
		Language:  "en",
		Task:      "transcribe",
	})
	if err != nil {
		skipIfExternalRealEngineDependency(t, err)
		t.Fatalf("real JFK transcription failed: %v", err)
	}
	elapsed := time.Since(start)
	t.Logf("real JFK transcription completed in %s with %d words and %d chars", elapsed, len(result.Words), len(result.Text))

	if strings.TrimSpace(result.Text) == "" {
		t.Fatalf("real JFK transcription returned empty text")
	}
	if result.Words == nil {
		t.Fatalf("real JFK transcription returned nil words")
	}
	if result.EngineID != DefaultProviderID {
		t.Fatalf("EngineID = %q, want %q", result.EngineID, DefaultProviderID)
	}
	if strings.Contains(result.Text, audioPath) || strings.Contains(result.Text, cacheDir) {
		t.Fatalf("transcript text leaked local path")
	}
}

func TestRealEngineAutoDownloadDisabledMissingModelIsSanitized(t *testing.T) {
	requireRealEngineIntegration(t)
	requireFFmpeg(t)

	audioPath := requireJFKFixture(t)
	cacheDir := t.TempDir()
	provider := newRealTestProvider(t, cacheDir, false)

	_, err := provider.Transcribe(context.Background(), TranscriptionRequest{
		JobID:     "real-jfk-cache-disabled",
		UserID:    1,
		AudioPath: audioPath,
		ModelID:   DefaultTranscriptionModel,
		Language:  "en",
	})
	if err == nil {
		t.Fatalf("expected missing model error with auto-download disabled")
	}
	msg := err.Error()
	if strings.Contains(msg, audioPath) || strings.Contains(msg, cacheDir) {
		t.Fatalf("missing-model error leaked local path: %q", msg)
	}
	if !errors.Is(err, speechengine.ErrModelUnavailable) && !strings.Contains(strings.ToLower(msg), "model") {
		t.Fatalf("expected model-unavailable style error, got: %v", err)
	}
}

func BenchmarkRealEngineJFKTranscription(b *testing.B) {
	if os.Getenv("SCRIBERR_ENGINE_ITEST") != "1" {
		b.Skip("set SCRIBERR_ENGINE_ITEST=1 to run real engine benchmark")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		b.Skipf("ffmpeg unavailable: %v", err)
	}
	audioPath := filepath.Join(projectRootForRealEngineTest(b), "test-audio", "jfk.wav")
	if _, err := os.Stat(audioPath); err != nil {
		b.Skipf("JFK fixture unavailable: %v", err)
	}
	provider := newRealTestProvider(b, realEngineCacheDir(b), true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		result, err := provider.Transcribe(context.Background(), TranscriptionRequest{
			JobID:     "bench-real-jfk",
			UserID:    1,
			AudioPath: audioPath,
			ModelID:   DefaultTranscriptionModel,
			Language:  "en",
		})
		if err != nil {
			skipIfExternalRealEngineDependency(b, err)
			b.Fatalf("real JFK transcription failed: %v", err)
		}
		b.Logf("iteration %d completed in %s with %d words and %d chars", i+1, time.Since(start), len(result.Words), len(result.Text))
	}
}

type realEngineTB interface {
	Helper()
	Fatalf(format string, args ...any)
	Skipf(format string, args ...any)
	Cleanup(func())
}

func requireRealEngineIntegration(t realEngineTB) {
	t.Helper()
	if os.Getenv("SCRIBERR_ENGINE_ITEST") != "1" {
		t.Skipf("set SCRIBERR_ENGINE_ITEST=1 to run real engine integration tests")
	}
}

func requireFFmpeg(t realEngineTB) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skipf("ffmpeg unavailable: %v", err)
	}
}

func requireJFKFixture(t realEngineTB) string {
	t.Helper()
	audioPath := filepath.Join(projectRootForRealEngineTest(t), "test-audio", "jfk.wav")
	if _, err := os.Stat(audioPath); err != nil {
		t.Skipf("JFK fixture unavailable: %v", err)
	}
	return audioPath
}

func newRealTestProvider(t realEngineTB, cacheDir string, autoDownload bool) *LocalProvider {
	t.Helper()
	provider, err := NewLocalProvider(appconfig.EngineConfig{
		CacheDir:     cacheDir,
		Provider:     "cpu",
		Threads:      0,
		MaxLoaded:    1,
		AutoDownload: autoDownload,
	})
	if err != nil {
		skipIfExternalRealEngineDependency(t, err)
		t.Fatalf("create real local provider: %v", err)
	}
	t.Cleanup(func() {
		if err := provider.Close(); err != nil {
			t.Fatalf("close real local provider: %v", err)
		}
	})
	return provider
}

func realEngineCacheDir(t realEngineTB) string {
	t.Helper()
	if cacheDir := strings.TrimSpace(os.Getenv("SCRIBERR_ENGINE_ITEST_CACHE_DIR")); cacheDir != "" {
		return cacheDir
	}
	return filepath.Join(projectRootForRealEngineTest(t), "data", "models")
}

func projectRootForRealEngineTest(t realEngineTB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find project root from %q", dir)
		}
		dir = parent
	}
}

func skipIfExternalRealEngineDependency(t realEngineTB, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := strings.ToLower(err.Error())
	externalMarkers := []string{
		"download",
		"network",
		"no such host",
		"connection refused",
		"connection reset",
		"timeout",
		"ffmpeg",
		"cuda",
		"sherpa",
		"dylib",
		"shared object",
	}
	for _, marker := range externalMarkers {
		if strings.Contains(msg, marker) {
			t.Skipf("real engine external dependency unavailable: %v", err)
		}
	}
}
