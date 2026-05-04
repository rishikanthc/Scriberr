package preprocess

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ProviderSampleRate = 16000
	ProviderChannels   = 1
	ProviderFormat     = "wav"
)

type Request struct {
	JobID          string
	SourcePath     string
	SourceFileHash string
}

type Artifact struct {
	Path         string
	ProviderPath string
	SampleRate   int
	Channels     int
	Format       string
}

type Preprocessor interface {
	Prepare(ctx context.Context, req Request) (Artifact, error)
}

type Config struct {
	Dir               string
	ProviderMountRoot string
	FFmpegPath        string
}

type LocalPreprocessor struct {
	cfg Config
}

func NewLocalPreprocessor(cfg Config) *LocalPreprocessor {
	return &LocalPreprocessor{cfg: Config{
		Dir:               strings.TrimSpace(cfg.Dir),
		ProviderMountRoot: strings.TrimRight(strings.TrimSpace(cfg.ProviderMountRoot), "/"),
		FFmpegPath:        strings.TrimSpace(cfg.FFmpegPath),
	}}
}

func (p *LocalPreprocessor) Prepare(ctx context.Context, req Request) (Artifact, error) {
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if strings.TrimSpace(p.cfg.Dir) == "" {
		return Artifact{}, fmt.Errorf("normalized audio directory is required")
	}
	if strings.TrimSpace(req.SourcePath) == "" {
		return Artifact{}, fmt.Errorf("source audio path is required")
	}
	name, err := artifactName(req)
	if err != nil {
		return Artifact{}, err
	}
	root := filepath.Clean(p.cfg.Dir)
	path := filepath.Join(root, name)
	if !pathWithin(root, path) {
		return Artifact{}, fmt.Errorf("normalized audio path escapes storage root")
	}
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return Artifact{}, fmt.Errorf("check normalized audio artifact: %w", err)
		}
		if err := convertArtifact(ctx, req.SourcePath, path, p.cfg.FFmpegPath); err != nil {
			return Artifact{}, err
		}
	}
	return Artifact{
		Path:         path,
		ProviderPath: providerVisiblePath(p.cfg.ProviderMountRoot, name, path),
		SampleRate:   ProviderSampleRate,
		Channels:     ProviderChannels,
		Format:       ProviderFormat,
	}, nil
}

type PassthroughPreprocessor struct{}

func (PassthroughPreprocessor) Prepare(ctx context.Context, req Request) (Artifact, error) {
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if strings.TrimSpace(req.SourcePath) == "" {
		return Artifact{}, fmt.Errorf("source audio path is required")
	}
	return Artifact{
		Path:         req.SourcePath,
		ProviderPath: req.SourcePath,
		SampleRate:   ProviderSampleRate,
		Channels:     ProviderChannels,
		Format:       ProviderFormat,
	}, nil
}

func artifactName(req Request) (string, error) {
	id := strings.TrimSpace(req.SourceFileHash)
	if id == "" {
		id = strings.TrimSpace(req.JobID)
	}
	if id == "" {
		return "", fmt.Errorf("normalized audio artifact id is required")
	}
	if strings.Contains(id, "/") || strings.Contains(id, `\`) || id == "." || id == ".." {
		return "", fmt.Errorf("normalized audio artifact id is invalid")
	}
	return id + "." + ProviderFormat, nil
}

func providerVisiblePath(mountRoot, name, fallback string) string {
	if mountRoot == "" {
		return fallback
	}
	return mountRoot + "/" + name
}

func convertArtifact(ctx context.Context, sourcePath, destPath, ffmpegPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("prepare normalized audio directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(destPath), ".audio-*.tmp")
	if err != nil {
		return fmt.Errorf("create normalized audio temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close normalized audio temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	ffmpeg := strings.TrimSpace(ffmpegPath)
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	if _, err := exec.LookPath(ffmpeg); err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-i", sourcePath,
		"-vn",
		"-ar", fmt.Sprintf("%d", ProviderSampleRate),
		"-ac", fmt.Sprintf("%d", ProviderChannels),
		"-f", ProviderFormat,
		tmpPath,
	}
	cmd := exec.CommandContext(ctx, ffmpeg, args...) // #nosec G204
	var stderr bytes.Buffer
	cmd.Stdout = nil
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("ffmpeg normalize audio failed: %w", err)
		}
		return fmt.Errorf("ffmpeg normalize audio failed: %w: %s", err, message)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("commit normalized audio artifact: %w", err)
	}
	return nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}
