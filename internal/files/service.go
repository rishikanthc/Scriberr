package files

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"
)

type Repository interface {
	Create(ctx context.Context, job *models.TranscriptionJob) error
	FindFileByIDForUser(ctx context.Context, id string, userID uint) (*models.TranscriptionJob, error)
	ListFilesByUser(ctx context.Context, userID uint, opts ListOptions) ([]models.TranscriptionJob, error)
	UpdateFileTitle(ctx context.Context, id string, userID uint, title string) error
	DeleteFile(ctx context.Context, id string, userID uint) error
	CompleteMediaImport(ctx context.Context, jobID, title, audioPath, sourceFileName string, durationMs *int64, completedAt time.Time) error
	FailMediaImport(ctx context.Context, jobID string, message string, failedAt time.Time) error
}

type EventPublisher interface {
	PublishFileEvent(ctx context.Context, name string, payload map[string]any)
}

type MediaExtractor interface {
	ExtractAudio(ctx context.Context, inputPath, outputPath string) error
}

type Service struct {
	repo      Repository
	events    EventPublisher
	uploadDir string
	extractor MediaExtractor
	asyncJobs *sync.WaitGroup
	timeout   time.Duration
}

type Config struct {
	UploadDir string
	Timeout   time.Duration
}

type ListOptions = repository.FileListOptions

type ListCursor = repository.FileListCursor

type UploadCommand struct {
	UserID      uint
	Filename    string
	ContentType string
	Title       string
	Body        io.Reader
}

type UploadResult struct {
	Job      *models.TranscriptionJob
	MimeType string
	Kind     string
}

var (
	ErrNotConfigured        = errors.New("file service is not configured")
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrNotFound             = errors.New("file not found")
)

func NewService(repo Repository, cfg Config) *Service {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Hour
	}
	return &Service{
		repo:      repo,
		uploadDir: cfg.UploadDir,
		extractor: ffmpegExtractor{},
		timeout:   timeout,
	}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) SetMediaExtractor(extractor MediaExtractor) {
	if extractor == nil {
		extractor = ffmpegExtractor{}
	}
	s.extractor = extractor
}

func (s *Service) SetAsyncJobs(wg *sync.WaitGroup) {
	s.asyncJobs = wg
}

func (s *Service) Upload(ctx context.Context, cmd UploadCommand) (*UploadResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNotConfigured
	}
	sourceName := safeFilename(cmd.Filename)
	if sourceName == "" {
		sourceName = randomHex(16)
	}
	mimeType := MediaType(cmd.ContentType, sourceName)
	kind := FileKind(mimeType)
	if kind == "" {
		return nil, ErrUnsupportedMediaType
	}
	uploadDir := s.uploadDir
	if uploadDir == "" {
		uploadDir = filepath.Join(os.TempDir(), "scriberr-uploads")
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, err
	}

	jobID := randomHex(16)
	storedName := jobID + filepath.Ext(sourceName)
	storagePath := filepath.Join(uploadDir, storedName)
	if err := writeNewFile(storagePath, cmd.Body); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		title = strings.TrimSuffix(sourceName, filepath.Ext(sourceName))
	}
	if kind == "video" {
		result, err := s.createVideoImport(ctx, jobID, cmd.UserID, title, sourceName, storagePath, uploadDir)
		if err != nil {
			_ = os.Remove(storagePath)
			return nil, err
		}
		return &UploadResult{Job: result, MimeType: mimeType, Kind: kind}, nil
	}

	job := &models.TranscriptionJob{
		ID:             jobID,
		UserID:         cmd.UserID,
		Title:          &title,
		Status:         models.StatusUploaded,
		AudioPath:      storagePath,
		SourceFileName: sourceName,
	}
	if err := s.repo.Create(ctx, job); err != nil {
		_ = os.Remove(storagePath)
		return nil, err
	}
	s.publish(ctx, "file.ready", job.ID, "audio", "ready")
	return &UploadResult{Job: job, MimeType: mimeType, Kind: kind}, nil
}

func (s *Service) List(ctx context.Context, userID uint, opts ListOptions) ([]models.TranscriptionJob, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNotConfigured
	}
	return s.repo.ListFilesByUser(ctx, userID, opts)
}

func (s *Service) Get(ctx context.Context, userID uint, id string) (*models.TranscriptionJob, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNotConfigured
	}
	job, err := s.repo.FindFileByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	return job, nil
}

func (s *Service) UpdateTitle(ctx context.Context, userID uint, id string, title string) (*models.TranscriptionJob, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if err := s.repo.UpdateFileTitle(ctx, id, userID, title); err != nil {
		return nil, ErrNotFound
	}
	return s.Get(ctx, userID, id)
}

func (s *Service) Delete(ctx context.Context, userID uint, id string) error {
	if s == nil || s.repo == nil {
		return ErrNotConfigured
	}
	if err := s.repo.DeleteFile(ctx, id, userID); err != nil {
		return ErrNotFound
	}
	s.publish(ctx, "file.deleted", id, "", "")
	return nil
}

func (s *Service) OpenAudio(ctx context.Context, userID uint, id string) (*os.File, *models.TranscriptionJob, error) {
	job, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}
	file, err := os.Open(job.AudioPath)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	return file, job, nil
}

func (s *Service) createVideoImport(ctx context.Context, jobID string, userID uint, title, sourceName, videoPath, uploadDir string) (*models.TranscriptionJob, error) {
	job := &models.TranscriptionJob{
		ID:             jobID,
		UserID:         userID,
		Title:          &title,
		Status:         models.StatusProcessing,
		AudioPath:      videoPath,
		SourceFileName: sourceName,
	}
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}
	s.publish(ctx, "file.processing", job.ID, "video", string(models.StatusProcessing))
	extractedPath := filepath.Join(uploadDir, jobID+".mp3")
	extractedName := strings.TrimSuffix(sourceName, filepath.Ext(sourceName)) + ".mp3"
	s.startVideoAudioExtraction(job.ID, title, videoPath, extractedPath, extractedName)
	return job, nil
}

func (s *Service) startVideoAudioExtraction(jobID, title, videoPath, extractedPath, extractedFilename string) {
	if s.asyncJobs != nil {
		s.asyncJobs.Add(1)
	}
	go func() {
		if s.asyncJobs != nil {
			defer s.asyncJobs.Done()
		}
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		extractor := s.extractor
		if extractor == nil {
			extractor = ffmpegExtractor{}
		}
		if err := extractor.ExtractAudio(ctx, videoPath, extractedPath); err != nil {
			_ = os.Remove(extractedPath)
			_ = s.repo.FailMediaImport(ctx, jobID, "video audio extraction failed", time.Now())
			s.publish(ctx, "file.failed", jobID, "video", string(models.StatusFailed))
			return
		}
		safeExtractedName := safeFilename(extractedFilename)
		if safeExtractedName == "" {
			safeExtractedName = filepath.Base(extractedPath)
		}
		if err := s.repo.CompleteMediaImport(ctx, jobID, title, extractedPath, safeExtractedName, nil, time.Now()); err != nil {
			_ = s.repo.FailMediaImport(ctx, jobID, "video audio extraction failed", time.Now())
			s.publish(ctx, "file.failed", jobID, "video", string(models.StatusFailed))
			return
		}
		_ = os.Remove(videoPath)
		s.publish(ctx, "file.ready", jobID, "audio", "ready")
	}()
}

func (s *Service) publish(ctx context.Context, name, jobID, kind, status string) {
	if s.events == nil {
		return
	}
	payload := map[string]any{"id": "file_" + jobID}
	if kind != "" {
		payload["kind"] = kind
	}
	if status != "" {
		payload["status"] = status
	}
	s.events.PublishFileEvent(ctx, name, payload)
}

func writeNewFile(path string, source io.Reader) error {
	destination, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destination, source); err != nil {
		_ = destination.Close()
		return err
	}
	return destination.Close()
}

func MediaType(headerValue, filename string) string {
	cleanHeader := strings.ToLower(strings.TrimSpace(strings.Split(headerValue, ";")[0]))
	if strings.HasPrefix(cleanHeader, "audio/") || strings.HasPrefix(cleanHeader, "video/") {
		return cleanHeader
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		return "audio/wav"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".opus":
		return "audio/opus"
	case ".webm":
		return "video/webm"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	default:
		return cleanHeader
	}
}

func FileKind(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	default:
		return ""
	}
}

func safeFilename(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return strings.NewReplacer("/", "_", "\\", "_", "\x00", "").Replace(base)
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

type ffmpegExtractor struct{}

func (ffmpegExtractor) ExtractAudio(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", inputPath, "-vn", "-acodec", "libmp3lame", outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg extraction failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
