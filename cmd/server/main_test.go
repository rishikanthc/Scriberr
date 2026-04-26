package main

import (
	"os"
	"strings"
	"testing"
)

func TestServerMainDoesNotReferenceLegacyPythonStartup(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"internal/queue",
		"internal/transcription/adapters",
		"internal/transcription/registry",
		"NewUnifiedJobProcessor",
		"NewQuickTranscriptionService",
		"InitEmbeddedPythonEnv",
		"registerAdapters",
		"WhisperXEnv",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("server main still references legacy startup symbol %q", forbidden)
		}
	}
}
