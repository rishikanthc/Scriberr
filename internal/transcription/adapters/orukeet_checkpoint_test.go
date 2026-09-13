package adapters

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"

	"scriberr/internal/transcription/interfaces"
)

func TestOrukeetCheckpointInstall(t *testing.T) {
	data := []byte("test checkpoint")
	var requests atomic.Int32
	var corrupt atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if corrupt.Load() {
			_, _ = w.Write([]byte("bad checkpoint!"))
			return
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	model := nemoCheckpoint{Filename: "test.nemo", URL: server.URL, SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Bytes: int64(len(data))}
	directory := t.TempDir()
	if err := downloadNemoCheckpoint(context.Background(), directory, model); err != nil {
		t.Fatal(err)
	}
	if err := downloadNemoCheckpoint(context.Background(), directory, model); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatal("cached checkpoint used the network")
	}

	// A failed replacement must leave the existing file intact and clean staging.
	previous := []byte("existing damaged checkpoint")
	if err := os.WriteFile(filepath.Join(directory, model.Filename), previous, 0600); err != nil {
		t.Fatal(err)
	}
	corrupt.Store(true)
	if err := downloadNemoCheckpoint(context.Background(), directory, model); err == nil {
		t.Fatal("accepted corrupt download")
	}
	got, err := os.ReadFile(filepath.Join(directory, model.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(previous) {
		t.Fatal("failed download replaced existing file")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
	corrupt.Store(false)
	if err := downloadNemoCheckpoint(context.Background(), directory, model); err != nil {
		t.Fatal(err)
	}
	if err := verifyNemoCheckpoint(context.Background(), filepath.Join(directory, model.Filename), model); err != nil {
		t.Fatal(err)
	}
}

func TestOrukeetCancelledDownload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	directory := t.TempDir()
	if err := downloadNemoCheckpoint(ctx, directory, orukeetCheckpoint); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
	entries, _ := os.ReadDir(directory)
	if len(entries) != 0 {
		t.Fatal("cancelled request created files")
	}
}

func TestOrukeetRegistrationAndArguments(t *testing.T) {
	for _, makeAdapter := range []func(string) *ParakeetAdapter{NewParakeetAdapter, NewOrukeetAdapter} {
		adapter := makeAdapter(t.TempDir())
		for _, build := range []func(interfaces.AudioInput, map[string]interface{}, string) ([]string, error){adapter.buildParakeetArgs, adapter.buildBufferedArgs} {
			args, err := build(interfaces.AudioInput{FilePath: "audio.wav"}, map[string]interface{}{}, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			i := slices.Index(args, "--model-file")
			if i < 0 || args[i+1] != adapter.model.Filename {
				t.Fatalf("wrong checkpoint: %v", args)
			}
		}
	}
	adapter := NewOrukeetAdapter(filepath.Join(t.TempDir(), "new-env"))
	if err := adapter.PrepareEnvironment(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(adapter.envPath)
	if len(entries) != 0 {
		t.Fatal("registration downloaded an optional checkpoint")
	}
	if !adapter.IsReady(context.Background()) {
		t.Fatal("optional adapter unavailable")
	}
	capabilities := adapter.GetCapabilities()
	if capabilities.ModelID != "orukeet" || len(capabilities.SupportedLanguages) != 25 || capabilities.Metadata["license"] != "CC-BY-SA-4.0" {
		t.Fatalf("wrong capabilities: %+v", capabilities)
	}
	if !slices.Equal(adapter.GetSupportedModels(), []string{"orukeet-v0.1.0"}) {
		t.Fatal("incorrect model identity")
	}
	if _, err := adapter.Transcribe(context.Background(), interfaces.AudioInput{}, nil, interfaces.ProcessingContext{}); err == nil {
		t.Fatal("invalid audio was accepted")
	}
	entries, _ = os.ReadDir(adapter.envPath)
	if len(entries) != 0 {
		t.Fatal("invalid audio triggered setup or download")
	}
}

func TestOrukeetCancelledWhileWaitingForPreparation(t *testing.T) {
	adapter := NewOrukeetAdapter(t.TempDir())
	adapter.prepareLock <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := adapter.prepareOrukeet(ctx); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}
