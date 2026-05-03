package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultOutputDir = "data/transcripts"

type TranscriptStore interface {
	SaveTranscriptJSON(ctx context.Context, jobID string, transcriptJSON []byte) (string, error)
}

type LocalTranscriptStore struct {
	OutputDir string
}

func NewLocalTranscriptStore(outputDir string) *LocalTranscriptStore {
	return &LocalTranscriptStore{OutputDir: outputDir}
}

func (s *LocalTranscriptStore) SaveTranscriptJSON(ctx context.Context, jobID string, transcriptJSON []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cleanJobID, err := safeArtifactJobID(jobID)
	if err != nil {
		return "", err
	}
	outputDir := s.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	jobDir := filepath.Join(outputDir, cleanJobID)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		return "", err
	}
	outputPath := filepath.Join(jobDir, "transcript.json")
	if err := os.WriteFile(outputPath, transcriptJSON, 0o600); err != nil {
		return "", err
	}
	return outputPath, nil
}

func safeArtifactJobID(jobID string) (string, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" || jobID == "." || filepath.Clean(jobID) != jobID || strings.ContainsAny(jobID, `/\`) {
		return "", fmt.Errorf("transcript artifact job id is invalid")
	}
	return jobID, nil
}
