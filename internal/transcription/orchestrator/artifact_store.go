package orchestrator

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"scriberr/internal/models"
)

type TranscriptStore interface {
	SaveTranscriptJSON(ctx context.Context, jobID string, transcriptJSON []byte) (string, error)
}

type LocalTranscriptStore struct {
	OutputDir string
}

func NewLocalTranscriptStore(outputDir string) *LocalTranscriptStore {
	return &LocalTranscriptStore{OutputDir: strings.TrimSpace(outputDir)}
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
		return "", fmt.Errorf("transcript output directory is required")
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

type LocalExecutionLogStore struct {
	RootDir string
}

func NewLocalExecutionLogStore(rootDir string) *LocalExecutionLogStore {
	return &LocalExecutionLogStore{RootDir: strings.TrimSpace(rootDir)}
}

func (s *LocalExecutionLogStore) ReadExecutionLog(ctx context.Context, execution models.TranscriptionJobExecution) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if execution.LogsPath == nil || strings.TrimSpace(*execution.LogsPath) == "" {
		return "", fs.ErrNotExist
	}
	path, err := s.safePath(*execution.LogsPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *LocalExecutionLogStore) safePath(raw string) (string, error) {
	root := strings.TrimSpace(s.RootDir)
	if root == "" {
		return "", fmt.Errorf("execution log root directory is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("execution log path is outside transcript storage")
	}
	return pathAbs, nil
}

func safeArtifactJobID(jobID string) (string, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" || jobID == "." || filepath.Clean(jobID) != jobID || strings.ContainsAny(jobID, `/\`) {
		return "", fmt.Errorf("transcript artifact job id is invalid")
	}
	return jobID, nil
}
