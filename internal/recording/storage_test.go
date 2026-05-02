package recording

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoragePathsStayUnderRoot(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}

	chunkPath, err := storage.ChunkPath("session-1", 7, "audio/webm;codecs=opus")
	if err != nil {
		t.Fatalf("ChunkPath returned error: %v", err)
	}
	assertUnderRoot(t, storage.Root(), chunkPath)
	if filepath.Base(chunkPath) != "000007.webm" {
		t.Fatalf("chunk path = %q, want 000007.webm suffix", chunkPath)
	}

	rawPath, err := storage.RawPath("session-1")
	if err != nil {
		t.Fatalf("RawPath returned error: %v", err)
	}
	assertUnderRoot(t, storage.Root(), rawPath)

	finalPath, err := storage.FinalPath("session-1", "audio/flac")
	if err != nil {
		t.Fatalf("FinalPath returned error: %v", err)
	}
	assertUnderRoot(t, storage.Root(), finalPath)
	if filepath.Base(finalPath) != "final.flac" {
		t.Fatalf("final path = %q, want final.flac suffix", finalPath)
	}
}

func TestStorageRejectsTraversalIDs(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}

	for _, sessionID := range []string{"../escape", "nested/path", `nested\path`, ".", "..", ""} {
		t.Run(sessionID, func(t *testing.T) {
			if _, err := storage.ChunkPath(sessionID, 0, "audio/webm"); err == nil {
				t.Fatalf("ChunkPath accepted invalid session id %q", sessionID)
			}
		})
	}
}

func TestWriteChunkUsesAtomicPathAndRestrictivePermissions(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}

	path, size, err := storage.WriteChunk(context.Background(), "session-1", 0, "audio/webm;codecs=opus", strings.NewReader("chunk data"))
	if err != nil {
		t.Fatalf("WriteChunk returned error: %v", err)
	}
	if size != int64(len("chunk data")) {
		t.Fatalf("size = %d", size)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "chunk data" {
		t.Fatalf("chunk data = %q", string(data))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("chunk file mode = %o, want 600", info.Mode().Perm())
	}
}

func TestWriteChunkRejectsDuplicateWithoutLeavingTempFiles(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}

	if _, _, err := storage.WriteChunk(context.Background(), "session-1", 0, "audio/webm", strings.NewReader("first")); err != nil {
		t.Fatalf("first WriteChunk returned error: %v", err)
	}
	_, _, err = storage.WriteChunk(context.Background(), "session-1", 0, "audio/webm", strings.NewReader("second"))
	if !errors.Is(err, ErrArtifactExists) {
		t.Fatalf("duplicate WriteChunk err = %v, want ErrArtifactExists", err)
	}

	chunkPath, err := storage.ChunkPath("session-1", 0, "audio/webm")
	if err != nil {
		t.Fatalf("ChunkPath returned error: %v", err)
	}
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("duplicate write changed chunk data to %q", string(data))
	}
	chunkDir, err := storage.ChunkDir("session-1")
	if err != nil {
		t.Fatalf("ChunkDir returned error: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(chunkDir, ".*.tmp"))
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("duplicate write left temp files: %v", matches)
	}
}

func TestRemoveTemporaryArtifactsKeepsFinalAudio(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}
	if _, _, err := storage.WriteChunk(context.Background(), "session-1", 0, "audio/webm", strings.NewReader("chunk")); err != nil {
		t.Fatalf("WriteChunk returned error: %v", err)
	}
	rawPath, err := storage.RawPath("session-1")
	if err != nil {
		t.Fatalf("RawPath returned error: %v", err)
	}
	if err := os.WriteFile(rawPath, []byte("raw"), 0o600); err != nil {
		t.Fatalf("WriteFile raw returned error: %v", err)
	}
	finalPath, err := storage.FinalPath("session-1", "audio/webm")
	if err != nil {
		t.Fatalf("FinalPath returned error: %v", err)
	}
	if err := os.WriteFile(finalPath, []byte("final"), 0o600); err != nil {
		t.Fatalf("WriteFile final returned error: %v", err)
	}

	if err := storage.RemoveTemporaryArtifacts("session-1"); err != nil {
		t.Fatalf("RemoveTemporaryArtifacts returned error: %v", err)
	}

	chunkDir, err := storage.ChunkDir("session-1")
	if err != nil {
		t.Fatalf("ChunkDir returned error: %v", err)
	}
	if _, err := os.Stat(chunkDir); !os.IsNotExist(err) {
		t.Fatalf("chunk dir still exists or unexpected error: %v", err)
	}
	if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
		t.Fatalf("raw artifact still exists or unexpected error: %v", err)
	}
	if data, err := os.ReadFile(finalPath); err != nil || string(data) != "final" {
		t.Fatalf("final artifact missing or changed: data=%q err=%v", string(data), err)
	}
}

func TestWriteChunkHonorsCanceledContext(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	path, _, err := storage.WriteChunk(ctx, "session-1", 0, "audio/webm", strings.NewReader("chunk"))
	if err == nil {
		t.Fatal("WriteChunk returned nil error for canceled context")
	}
	if path != "" {
		t.Fatalf("path = %q, want empty on canceled write", path)
	}
	chunkDir, chunkErr := storage.ChunkDir("session-1")
	if chunkErr != nil {
		t.Fatalf("ChunkDir returned error: %v", chunkErr)
	}
	if _, statErr := os.Stat(chunkDir); !os.IsNotExist(statErr) {
		t.Fatalf("chunk dir exists after canceled write or unexpected error: %v", statErr)
	}
}

func assertUnderRoot(t *testing.T, root string, path string) {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("Rel returned error: %v", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		t.Fatalf("path %q is outside root %q", path, root)
	}
}
