package mediaimport

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	filesdomain "scriberr/internal/files"
	"scriberr/internal/models"
)

type Repository interface {
	Create(ctx context.Context, job *models.TranscriptionJob) error
	UpdateProgress(ctx context.Context, jobID string, progress float64, stage string) error
	CompleteMediaImport(ctx context.Context, jobID, title, audioPath, sourceFileName string, durationMs *int64, completedAt time.Time) error
	FailMediaImport(ctx context.Context, jobID string, message string, failedAt time.Time) error
}

type EventPublisher interface {
	PublishFileEvent(ctx context.Context, name string, payload map[string]any)
}

type YouTubeImporter interface {
	Import(ctx context.Context, job YouTubeImportJob, onProgress ProgressFunc) (YouTubeImportResult, error)
}

type ProgressFunc func(progressPercent float64)

type YouTubeImportJob struct {
	URL        string
	OutputPath string
	Title      string
}

type YouTubeImportResult struct {
	Title      string
	Filename   string
	MimeType   string
	DurationMs *int64
}

type Service struct {
	repo      Repository
	importer  YouTubeImporter
	publisher EventPublisher
	ready     filesdomain.ReadyHandoff
	uploadDir string
	timeout   time.Duration
	asyncJobs *sync.WaitGroup
}

type ServiceOptions struct {
	Repository Repository
	Importer   YouTubeImporter
	Publisher  EventPublisher
	Ready      filesdomain.ReadyHandoff
	UploadDir  string
	Timeout    time.Duration
	AsyncJobs  *sync.WaitGroup
}

type ImportYouTubeCommand struct {
	UserID uint
	URL    string
	Title  string
}

func NewService(opts ServiceOptions) *Service {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Hour
	}
	importer := opts.Importer
	if importer == nil {
		importer = YTDLPImporter{}
	}
	return &Service{
		repo:      opts.Repository,
		importer:  importer,
		publisher: opts.Publisher,
		ready:     opts.Ready,
		uploadDir: opts.UploadDir,
		timeout:   timeout,
		asyncJobs: opts.AsyncJobs,
	}
}

func (s *Service) SetPublisher(publisher EventPublisher) {
	s.publisher = publisher
}

func (s *Service) SetAsyncJobs(asyncJobs *sync.WaitGroup) {
	s.asyncJobs = asyncJobs
}

func (s *Service) SetReadyHandoff(ready filesdomain.ReadyHandoff) {
	s.ready = ready
}

func (s *Service) SetImporter(importer YouTubeImporter) {
	if importer == nil {
		importer = YTDLPImporter{}
	}
	s.importer = importer
}

func (s *Service) ImportYouTube(ctx context.Context, cmd ImportYouTubeCommand) (*models.TranscriptionJob, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("media import service is not configured")
	}
	rawURL := strings.TrimSpace(cmd.URL)
	if !ValidYouTubeURL(rawURL) {
		return nil, ErrInvalidYouTubeURL
	}
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		title = "YouTube audio"
	}
	uploadDir := s.uploadDir
	if uploadDir == "" {
		uploadDir = filepath.Join(os.TempDir(), "scriberr-uploads")
	}
	jobID := randomID(16)
	storedName := jobID + ".mp3"
	storagePath := filepath.Join(uploadDir, storedName)

	job := models.TranscriptionJob{
		ID:             jobID,
		UserID:         cmd.UserID,
		Title:          &title,
		Status:         models.StatusProcessing,
		AudioPath:      storagePath,
		SourceFileName: "youtube:" + storedName,
		Progress:       0.01,
		ProgressStage:  "queued",
	}
	if err := s.repo.Create(ctx, &job); err != nil {
		return nil, err
	}
	s.publish(ctx, "file.processing", jobID, "youtube", "processing", 1)
	s.startDownload(jobID, rawURL, title, storagePath)
	return &job, nil
}

func (s *Service) startDownload(jobID, rawURL, title, storagePath string) {
	if s.asyncJobs != nil {
		s.asyncJobs.Add(1)
	}
	go func() {
		if s.asyncJobs != nil {
			defer s.asyncJobs.Done()
		}
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()

		result, err := s.importer.Import(ctx, YouTubeImportJob{
			URL:        rawURL,
			OutputPath: storagePath,
			Title:      title,
		}, func(progress float64) {
			progress = clamp(progress, 1, 99)
			_ = s.repo.UpdateProgress(ctx, jobID, progress/100, "downloading")
			s.publish(ctx, "file.processing", jobID, "youtube", "processing", progress)
		})
		if err != nil {
			_ = os.Remove(storagePath)
			_ = s.repo.FailMediaImport(ctx, jobID, "YouTube import failed", time.Now())
			s.publish(ctx, "file.failed", jobID, "youtube", string(models.StatusFailed), 0)
			return
		}

		finalTitle := strings.TrimSpace(result.Title)
		if finalTitle == "" {
			finalTitle = title
		}
		sourceFilename := result.Filename
		if result.Title != "" {
			sourceFilename = finalTitle + ".mp3"
		}
		sourceName := "youtube:" + safeFilename(sourceFilename)
		if sourceName == "youtube:" {
			sourceName = "youtube:" + safeFilename(finalTitle) + ".mp3"
		}
		if sourceName == "youtube:.mp3" {
			sourceName = "youtube:" + filepath.Base(storagePath)
		}
		if err := s.repo.CompleteMediaImport(ctx, jobID, finalTitle, storagePath, sourceName, result.DurationMs, time.Now()); err != nil {
			_ = s.repo.FailMediaImport(ctx, jobID, "YouTube import failed", time.Now())
			s.publish(ctx, "file.failed", jobID, "youtube", string(models.StatusFailed), 0)
			return
		}
		if s.ready != nil {
			_ = s.ready.FileReady(ctx, filesdomain.ReadyEvent{FileID: jobID, Kind: "youtube", Status: "ready"})
		} else {
			s.publish(ctx, "file.ready", jobID, "youtube", "ready", 100)
		}
	}()
}

func (s *Service) publish(ctx context.Context, name, jobID, kind, status string, progress float64) {
	if s.publisher == nil {
		return
	}
	s.publisher.PublishFileEvent(ctx, name, map[string]any{
		"id":       "file_" + jobID,
		"kind":     kind,
		"status":   status,
		"progress": progress,
	})
}

var ErrInvalidYouTubeURL = fmt.Errorf("youtube url is invalid")

func ValidYouTubeURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be", "www.youtu.be", "youtube-nocookie.com", "www.youtube-nocookie.com":
		return true
	default:
		return false
	}
}

type YTDLPImporter struct{}

func (YTDLPImporter) Import(ctx context.Context, job YouTubeImportJob, onProgress ProgressFunc) (YouTubeImportResult, error) {
	if err := os.MkdirAll(filepath.Dir(job.OutputPath), 0755); err != nil {
		return YouTubeImportResult{}, fmt.Errorf("prepare output directory: %w", err)
	}

	outputTemplate := strings.TrimSuffix(job.OutputPath, filepath.Ext(job.OutputPath)) + ".%(ext)s"
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--newline",
		"--progress",
		"--print", "after_move:SCRIBERR_TITLE:%(title)s",
		"--output", outputTemplate,
		"--",
		job.URL,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return YouTubeImportResult{}, fmt.Errorf("prepare yt-dlp stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return YouTubeImportResult{}, fmt.Errorf("prepare yt-dlp stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return YouTubeImportResult{}, fmt.Errorf("start yt-dlp: %w", err)
	}

	var outputMu sync.Mutex
	lastLine := ""
	downloadedTitle := ""
	recordLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		if title, ok := strings.CutPrefix(line, "SCRIBERR_TITLE:"); ok {
			outputMu.Lock()
			downloadedTitle = strings.TrimSpace(title)
			outputMu.Unlock()
			return
		}
		outputMu.Lock()
		lastLine = line
		outputMu.Unlock()
		if progress, ok := parseYTDLPProgress(line); ok && onProgress != nil {
			onProgress(progress)
		}
	}

	var scanWG sync.WaitGroup
	scanWG.Add(2)
	go scanYTDLPOutput(stdout, recordLine, &scanWG)
	go scanYTDLPOutput(stderr, recordLine, &scanWG)
	waitErr := cmd.Wait()
	scanWG.Wait()
	if waitErr != nil {
		outputMu.Lock()
		message := sanitizeCommandOutput(lastLine)
		outputMu.Unlock()
		return YouTubeImportResult{}, fmt.Errorf("yt-dlp failed: %s", message)
	}

	matches, err := filepath.Glob(strings.TrimSuffix(job.OutputPath, filepath.Ext(job.OutputPath)) + ".*")
	if err != nil || len(matches) == 0 {
		return YouTubeImportResult{}, fmt.Errorf("downloaded file was not created")
	}
	downloadedPath := matches[0]
	if downloadedPath != job.OutputPath {
		if err := os.Rename(downloadedPath, job.OutputPath); err != nil {
			return YouTubeImportResult{}, fmt.Errorf("finalize downloaded file: %w", err)
		}
	}
	outputMu.Lock()
	title := downloadedTitle
	outputMu.Unlock()
	filenameTitle := title
	if filenameTitle == "" {
		filenameTitle = job.Title
	}
	return YouTubeImportResult{
		Title:    title,
		Filename: safeFilename(filenameTitle) + ".mp3",
		MimeType: "audio/mpeg",
	}, nil
}

func scanYTDLPOutput(reader io.Reader, recordLine func(string), wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		recordLine(scanner.Text())
	}
}

var progressPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)%`)

func parseYTDLPProgress(line string) (float64, bool) {
	match := progressPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, false
	}
	return clamp(value, 0, 100), true
}

func sanitizeCommandOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "download failed"
	}
	if len(output) > 160 {
		output = output[:160]
	}
	return output
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	return strings.TrimSpace(name)
}

func randomID(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
