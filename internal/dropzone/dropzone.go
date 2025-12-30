package dropzone

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
)

// TaskQueue interface for enqueueing transcription jobs
type TaskQueue interface {
	EnqueueJob(jobID string) error
}

// Service manages the dropzone file monitoring
type Service struct {
	config       *config.Config
	watcher      *fsnotify.Watcher
	dropzonePath string
	taskQueue    TaskQueue
	jobRepo      repository.JobRepository
	userRepo     repository.UserRepository
}

// NewService creates a new dropzone service
func NewService(cfg *config.Config, taskQueue TaskQueue, jobRepo repository.JobRepository, userRepo repository.UserRepository) *Service {
	return &Service{
		config:       cfg,
		taskQueue:    taskQueue,
		dropzonePath: filepath.Join("data", "dropzone"),
		jobRepo:      jobRepo,
		userRepo:     userRepo,
	}
}

// Start initializes the dropzone directory and starts file monitoring
func (s *Service) Start() error {
	log.Printf("Starting dropzone service...")

	// Create dropzone directory if it doesn't exist
	if err := os.MkdirAll(s.dropzonePath, 0755); err != nil {
		return fmt.Errorf("failed to create dropzone directory: %v", err)
	}

	log.Printf("Dropzone directory created/verified at: %s", s.dropzonePath)

	// Initialize file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %v", err)
	}
	s.watcher = watcher

	// Add dropzone directory and all subdirectories to watcher recursively
	if err := s.addDirectoryRecursively(s.dropzonePath); err != nil {
		s.watcher.Close()
		return fmt.Errorf("failed to add directories to watcher: %v", err)
	}

	// Process existing files recursively on startup
	if err := s.processExistingFiles(); err != nil {
		log.Printf("Warning: failed to process some existing files: %v", err)
	}

	// Start monitoring in a goroutine
	go s.watchFiles()

	log.Printf("Dropzone service started, monitoring recursively: %s", s.dropzonePath)
	return nil
}

// Stop stops the dropzone service
func (s *Service) Stop() error {
	if s.watcher != nil {
		log.Printf("Stopping dropzone service...")
		return s.watcher.Close()
	}
	return nil
}

// addDirectoryRecursively adds a directory and all its subdirectories to the watcher
func (s *Service) addDirectoryRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: error accessing path %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Only add directories to the watcher
		if info.IsDir() {
			if err := s.watcher.Add(path); err != nil {
				log.Printf("Warning: failed to watch directory %s: %v", path, err)
				return nil // Continue despite individual directory failures
			}
			log.Printf("Added directory to watcher: %s", path)
		}

		return nil
	})
}

// processExistingFiles processes all existing audio files in the dropzone on startup
func (s *Service) processExistingFiles() error {
	return filepath.Walk(s.dropzonePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: error accessing path %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Only process files, not directories
		if !info.IsDir() {
			filename := filepath.Base(path)
			if s.isAudioFile(filename) {
				log.Printf("Processing existing audio file: %s", path)
				s.processFile(path)
			}
		}

		return nil
	})
}

// watchFiles monitors the dropzone directory for new files
func (s *Service) watchFiles() {
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Handle creation events for both files and directories
			if event.Op&fsnotify.Create == fsnotify.Create {
				// Check if the created item is a directory
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					log.Printf("Detected new directory in dropzone: %s", event.Name)
					// Add the new directory to the watcher recursively
					if err := s.addDirectoryRecursively(event.Name); err != nil {
						log.Printf("Failed to watch new directory %s: %v", event.Name, err)
					}
				} else {
					log.Printf("Detected new file in dropzone: %s", event.Name)
					s.processFile(event.Name)
				}
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Dropzone watcher error: %v", err)
		}
	}
}

// isAudioFile checks if the file is a valid audio file based on extension
func (s *Service) isAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	audioExtensions := []string{
		".mp3", ".wav", ".flac", ".m4a", ".aac", ".ogg",
		".wma", ".mp4", ".avi", ".mov", ".mkv", ".webm",
	}

	for _, validExt := range audioExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

// processFile handles a newly detected file in the dropzone
func (s *Service) processFile(filePath string) {
	// Small delay to ensure file is fully written
	time.Sleep(500 * time.Millisecond)

	filename := filepath.Base(filePath)

	// Check if it's an audio file
	if !s.isAudioFile(filename) {
		log.Printf("Skipping non-audio file: %s", filename)
		return
	}

	// Check if file exists and is accessible
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error accessing file %s: %v", filePath, err)
		return
	}

	// Skip if it's a directory
	if fileInfo.IsDir() {
		return
	}

	log.Printf("Processing audio file: %s", filename)

	// Upload the file using the same logic as the API handler
	if err := s.uploadFile(filePath, filename); err != nil {
		log.Printf("Failed to upload file %s: %v", filename, err)
		return
	}

	// Delete the original file from dropzone after successful upload
	// Retry a few times in case of file locks
	var deleteErr error
	for i := 0; i < 5; i++ {
		deleteErr = os.Remove(filePath)
		if deleteErr == nil {
			break
		}
		// If it's a permission error or similar, wait and retry
		time.Sleep(500 * time.Millisecond)
	}

	if deleteErr != nil {
		log.Printf("Warning: Failed to delete file from dropzone %s after retries: %v", filePath, deleteErr)
	} else {
		log.Printf("Successfully processed and removed file: %s", filename)
	}
}

// uploadFile uploads the file using the existing pipeline logic
func (s *Service) uploadFile(sourcePath, originalFilename string) error {
	// Create upload directory
	uploadDir := s.config.UploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %v", err)
	}

	// Generate unique filename
	jobID := uuid.New().String()
	ext := filepath.Ext(originalFilename)
	filename := fmt.Sprintf("%s%s", jobID, ext)
	destPath := filepath.Join(uploadDir, filename)

	// Copy file from dropzone to upload directory
	if err := s.copyFile(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	// Create job record with "uploaded" status
	job := models.TranscriptionJob{
		ID:        jobID,
		AudioPath: destPath,
		Status:    models.StatusUploaded,
		Title:     &originalFilename, // Use original filename as title
	}

	// Save to database
	if err := s.jobRepo.Create(context.Background(), &job); err != nil {
		os.Remove(destPath) // Clean up file on database error
		return fmt.Errorf("failed to create job record: %v", err)
	}

	// Check if auto-transcription is enabled
	if s.isAutoTranscriptionEnabled() {
		// Multi-track files should never be auto-transcribed
		if job.IsMultiTrack {
			log.Printf("Skipping auto-transcription for multi-track job %s", jobID)
		} else {
			log.Printf("Auto-transcription enabled, enqueueing job %s", jobID)

			// Update job status to pending before enqueueing
			job.Status = models.StatusPending
			if err := s.jobRepo.Update(context.Background(), &job); err != nil {
				log.Printf("Warning: Failed to update job status to pending: %v", err)
			}

			// Enqueue the job for transcription
			if err := s.taskQueue.EnqueueJob(jobID); err != nil {
				log.Printf("Failed to enqueue job %s for transcription: %v", jobID, err)
			} else {
				log.Printf("Job %s enqueued for auto-transcription", jobID)
			}
		}
	}

	log.Printf("Successfully uploaded file %s as job %s", originalFilename, jobID)
	return nil
}

// isAutoTranscriptionEnabled checks if auto-transcription is enabled for any user
func (s *Service) isAutoTranscriptionEnabled() bool {
	count, err := s.userRepo.CountWithAutoTranscription(context.Background())
	if err != nil {
		log.Printf("Error checking auto-transcription settings: %v", err)
		return false
	}

	return count > 0
}

// copyFile copies a file from source to destination
func (s *Service) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
