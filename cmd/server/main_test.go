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

func TestServerMainKeepsBackendCompositionInAppPackage(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"internal/account",
		"internal/annotations",
		"internal/api",
		"internal/auth",
		"internal/automation",
		"internal/chat",
		"internal/database",
		"internal/files",
		"internal/llmprovider",
		"internal/mediaimport",
		"internal/profile",
		"internal/recording",
		"internal/repository",
		"internal/summarization",
		"internal/tags",
		"internal/transcription",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("server main still owns backend composition import %q", forbidden)
		}
	}
}
