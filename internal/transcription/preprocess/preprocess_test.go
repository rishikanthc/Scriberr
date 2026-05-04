package preprocess

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocalPreprocessorReturnsProviderVisibleArtifact(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.wav")
	if err := os.WriteFile(source, []byte("fake audio"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	preprocessor := NewLocalPreprocessor(Config{
		Dir:               root,
		ProviderMountRoot: "/provider-input/audio",
		FFmpegPath:        fakeFFmpeg(t),
	})

	artifact, err := preprocessor.Prepare(context.Background(), Request{
		JobID:          "job-123",
		SourcePath:     source,
		SourceFileHash: "hash-abc",
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if artifact.Path == "" || artifact.ProviderPath == "" {
		t.Fatalf("artifact paths were not set: %#v", artifact)
	}
	if artifact.ProviderPath != "/provider-input/audio/hash-abc.wav" {
		t.Fatalf("ProviderPath = %q", artifact.ProviderPath)
	}
	if artifact.SampleRate != 16000 || artifact.Channels != 1 || artifact.Format != "wav" {
		t.Fatalf("unexpected audio contract: %#v", artifact)
	}
	if _, err := os.Stat(artifact.Path); err != nil {
		t.Fatalf("artifact does not exist: %v", err)
	}
	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(data) != "normalized audio\n" {
		t.Fatalf("artifact data = %q", string(data))
	}
}

func TestLocalPreprocessorReusesExistingArtifact(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.wav")
	if err := os.WriteFile(source, []byte("first"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	preprocessor := NewLocalPreprocessor(Config{Dir: root, ProviderMountRoot: "/provider-input/audio", FFmpegPath: fakeFFmpeg(t)})

	first, err := preprocessor.Prepare(context.Background(), Request{JobID: "job-123", SourcePath: source})
	if err != nil {
		t.Fatalf("first Prepare returned error: %v", err)
	}
	if err := os.WriteFile(source, []byte("second"), 0o600); err != nil {
		t.Fatalf("rewrite source: %v", err)
	}
	second, err := preprocessor.Prepare(context.Background(), Request{JobID: "job-123", SourcePath: source})
	if err != nil {
		t.Fatalf("second Prepare returned error: %v", err)
	}

	if first.Path != second.Path {
		t.Fatalf("artifact path changed: %q != %q", first.Path, second.Path)
	}
	data, err := os.ReadFile(second.Path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(data) != "normalized audio\n" {
		t.Fatalf("cached artifact was overwritten: %q", string(data))
	}
}

func TestLocalPreprocessorRejectsUnsafeIDs(t *testing.T) {
	preprocessor := NewLocalPreprocessor(Config{Dir: t.TempDir(), ProviderMountRoot: "/provider-input/audio", FFmpegPath: fakeFFmpeg(t)})

	_, err := preprocessor.Prepare(context.Background(), Request{
		JobID:      "../escape",
		SourcePath: filepath.Join(t.TempDir(), "source.wav"),
	})
	if err == nil {
		t.Fatal("Prepare returned nil error for unsafe job id")
	}
}

func TestPassthroughPreprocessorKeepsSourcePath(t *testing.T) {
	artifact, err := PassthroughPreprocessor{}.Prepare(context.Background(), Request{
		JobID:      "job-123",
		SourcePath: "/tmp/source.wav",
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if artifact.ProviderPath != "/tmp/source.wav" || artifact.Path != "/tmp/source.wav" {
		t.Fatalf("unexpected passthrough artifact: %#v", artifact)
	}
}

func fakeFFmpeg(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake ffmpeg shell script is unix-only")
	}
	path := filepath.Join(t.TempDir(), "ffmpeg")
	script := "#!/bin/sh\nset -eu\nout=\"\"\nfor arg in \"$@\"; do out=\"$arg\"; done\nprintf 'normalized audio\\n' > \"$out\"\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
	return path
}
