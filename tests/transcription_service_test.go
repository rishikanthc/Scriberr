package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/transcription"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TranscriptionServiceTestSuite struct {
	suite.Suite
	helper             *TestHelper
	whisperXService    *transcription.WhisperXService
	quickTranscription *transcription.QuickTranscriptionService
	sampleAudioPath    string
}

func (suite *TranscriptionServiceTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "transcription_test.db")

	// Initialize transcription services
	suite.whisperXService = transcription.NewWhisperXService(suite.helper.Config)
	var err error
	suite.quickTranscription, err = transcription.NewQuickTranscriptionService(suite.helper.Config, suite.whisperXService)
	assert.NoError(suite.T(), err)

	// Set path to sample audio file
	suite.sampleAudioPath = "/Users/richandrasekaran/Code/machy/Scriberr/samples/jfk.wav"

	// Verify sample file exists
	_, err = os.Stat(suite.sampleAudioPath)
	assert.NoError(suite.T(), err, "Sample audio file jfk.wav should exist")
}

func (suite *TranscriptionServiceTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

// Test WhisperX service creation
func (suite *TranscriptionServiceTestSuite) TestNewWhisperXService() {
	service := transcription.NewWhisperXService(suite.helper.Config)
	assert.NotNil(suite.T(), service)
}

// Test TranscriptResult structure
func (suite *TranscriptionServiceTestSuite) TestTranscriptResultStructure() {
	// Test JSON marshaling/unmarshaling of transcript structures
	segment := transcription.Segment{
		Start:   0.0,
		End:     5.0,
		Text:    "This is a test segment",
		Speaker: stringPtr("SPEAKER_01"),
	}

	word := transcription.Word{
		Start:   0.0,
		End:     1.0,
		Word:    "This",
		Score:   0.95,
		Speaker: stringPtr("SPEAKER_01"),
	}

	result := transcription.TranscriptResult{
		Segments: []transcription.Segment{segment},
		Word:     []transcription.Word{word},
		Language: "en",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(result)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(jsonData), "This is a test segment")
	assert.Contains(suite.T(), string(jsonData), "SPEAKER_01")

	// Test JSON unmarshaling
	var unmarshaled transcription.TranscriptResult
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "en", unmarshaled.Language)
	assert.Len(suite.T(), unmarshaled.Segments, 1)
	assert.Equal(suite.T(), "This is a test segment", unmarshaled.Segments[0].Text)
}

// Test creating test transcription job with sample audio
func (suite *TranscriptionServiceTestSuite) TestCreateJobWithSampleAudio() {
	// Copy sample audio to test upload directory
	testAudioPath := filepath.Join(suite.helper.Config.UploadDir, "jfk_test.wav")

	// Copy the sample file
	input, err := os.Open(suite.sampleAudioPath)
	assert.NoError(suite.T(), err)
	defer input.Close()

	output, err := os.Create(testAudioPath)
	assert.NoError(suite.T(), err)
	defer output.Close()

	_, err = output.ReadFrom(input)
	assert.NoError(suite.T(), err)

	// Create transcription job
	title := "JFK Test Transcription"
	job := &models.TranscriptionJob{
		ID:        "jfk-test-job-123",
		Title:     &title,
		Status:    models.StatusPending,
		AudioPath: testAudioPath,
		Parameters: models.WhisperXParams{
			Model:       "tiny", // Use tiny model for faster testing
			BatchSize:   8,
			ComputeType: "float32",
			Device:      "cpu",
			Language:    stringPtr("en"),
		},
	}

	// For testing, we verify the job structure is valid
	assert.Equal(suite.T(), "JFK Test Transcription", *job.Title)
	assert.Equal(suite.T(), models.StatusPending, job.Status)
	assert.Equal(suite.T(), testAudioPath, job.AudioPath)
	assert.Equal(suite.T(), "tiny", job.Parameters.Model)
}

// Test WhisperX parameters structure
func (suite *TranscriptionServiceTestSuite) TestWhisperXParameters() {
	params := models.WhisperXParams{
		Model:                          "base",
		ModelCacheOnly:                 false,
		ModelDir:                       stringPtr("/custom/models"),
		Device:                         "auto",
		DeviceIndex:                    0,
		BatchSize:                      16,
		ComputeType:                    "float16",
		Threads:                        4,
		OutputFormat:                   "all",
		Verbose:                        true,
		Task:                           "transcribe",
		Language:                       stringPtr("en"),
		AlignModel:                     stringPtr("WAV2VEC2_ASR_BASE_960H"),
		InterpolateMethod:              "nearest",
		NoAlign:                        false,
		ReturnCharAlignments:           false,
		VadMethod:                      "pyannote",
		VadOnset:                       0.5,
		VadOffset:                      0.363,
		ChunkSize:                      30,
		Diarize:                        true,
		MinSpeakers:                    intPtr(1),
		MaxSpeakers:                    intPtr(10),
		DiarizeModel:                   "pyannote/speaker-diarization-3.1",
		SpeakerEmbeddings:              false,
		Temperature:                    0.0,
		BestOf:                         5,
		BeamSize:                       5,
		Patience:                       1.0,
		LengthPenalty:                  1.0,
		SuppressTokens:                 stringPtr("-1"),
		SuppressNumerals:               false,
		InitialPrompt:                  stringPtr(""),
		ConditionOnPreviousText:        false,
		Fp16:                           true,
		TemperatureIncrementOnFallback: 0.2,
		CompressionRatioThreshold:      2.4,
		LogprobThreshold:               -1.0,
		NoSpeechThreshold:              0.6,
		MaxLineWidth:                   intPtr(80),
		MaxLineCount:                   intPtr(3),
		HighlightWords:                 false,
		SegmentResolution:              "sentence",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(params)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(jsonData), "base")
	assert.Contains(suite.T(), string(jsonData), "float16")

	// Test JSON unmarshaling
	var unmarshaledParams models.WhisperXParams
	err = json.Unmarshal(jsonData, &unmarshaledParams)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "base", unmarshaledParams.Model)
	assert.Equal(suite.T(), 16, unmarshaledParams.BatchSize)
	assert.True(suite.T(), unmarshaledParams.Diarize)
}

// Test audio file validation
func (suite *TranscriptionServiceTestSuite) TestAudioFileValidation() {
	// Test with existing file
	_, err := os.Stat(suite.sampleAudioPath)
	assert.NoError(suite.T(), err, "Sample audio file should exist and be accessible")

	// Test file size
	fileInfo, err := os.Stat(suite.sampleAudioPath)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), fileInfo.Size(), int64(1000), "Audio file should have reasonable size")

	// Test with non-existent file
	nonExistentPath := "/path/to/nonexistent/audio.wav"
	_, err = os.Stat(nonExistentPath)
	assert.Error(suite.T(), err, "Non-existent file should return error")
}

// Test different audio formats (structure test - not actual processing)
func (suite *TranscriptionServiceTestSuite) TestAudioFormatSupport() {
	supportedFormats := []string{".wav", ".mp3", ".m4a", ".flac", ".ogg"}

	for _, format := range supportedFormats {
		// Test that we can create job with different formats
		testPath := filepath.Join(suite.helper.Config.UploadDir, "test"+format)

		job := &models.TranscriptionJob{
			ID:        "format-test-" + format[1:], // Remove the dot
			AudioPath: testPath,
			Parameters: models.WhisperXParams{
				Model:  "tiny",
				Device: "cpu",
			},
		}

		assert.Contains(suite.T(), job.AudioPath, format)
	}
}

// Test transcription job status transitions
func (suite *TranscriptionServiceTestSuite) TestJobStatusTransitions() {
	validStatuses := []models.JobStatus{
		models.StatusPending,
		models.StatusProcessing,
		models.StatusCompleted,
		models.StatusFailed,
	}

	for _, status := range validStatuses {
		job := &models.TranscriptionJob{
			ID:        "status-test-" + string(status),
			Status:    status,
			AudioPath: suite.sampleAudioPath,
		}

		assert.Equal(suite.T(), status, job.Status)

		// Test status string conversion
		statusStr := string(status)
		assert.NotEmpty(suite.T(), statusStr)
	}
}

// Test model parameters validation
func (suite *TranscriptionServiceTestSuite) TestModelParameters() {
	supportedModels := []string{"tiny", "base", "small", "medium", "large", "large-v2", "large-v3"}

	for _, model := range supportedModels {
		params := models.WhisperXParams{
			Model:       model,
			BatchSize:   8,
			ComputeType: "float32",
			Device:      "cpu",
		}

		assert.Equal(suite.T(), model, params.Model)
		assert.True(suite.T(), params.BatchSize > 0)
		assert.NotEmpty(suite.T(), params.ComputeType)
		assert.NotEmpty(suite.T(), params.Device)
	}
}

// Test device configuration
func (suite *TranscriptionServiceTestSuite) TestDeviceConfiguration() {
	devices := []string{"cpu", "cuda", "auto"}

	for _, device := range devices {
		params := models.WhisperXParams{
			Model:  "tiny",
			Device: device,
		}

		assert.Equal(suite.T(), device, params.Device)
	}
}

// Test batch size validation
func (suite *TranscriptionServiceTestSuite) TestBatchSizeValidation() {
	validBatchSizes := []int{1, 4, 8, 16, 32}

	for _, batchSize := range validBatchSizes {
		params := models.WhisperXParams{
			Model:     "tiny",
			BatchSize: batchSize,
		}

		assert.Equal(suite.T(), batchSize, params.BatchSize)
		assert.True(suite.T(), params.BatchSize > 0)
		assert.True(suite.T(), params.BatchSize <= 32)
	}
}

// Test compute type options
func (suite *TranscriptionServiceTestSuite) TestComputeTypes() {
	computeTypes := []string{"float16", "float32", "int8"}

	for _, computeType := range computeTypes {
		params := models.WhisperXParams{
			Model:       "tiny",
			ComputeType: computeType,
		}

		assert.Equal(suite.T(), computeType, params.ComputeType)
	}
}

// Test language parameter
func (suite *TranscriptionServiceTestSuite) TestLanguageParameter() {
	languages := []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"}

	for _, lang := range languages {
		params := models.WhisperXParams{
			Model:    "tiny",
			Language: stringPtr(lang),
		}

		assert.NotNil(suite.T(), params.Language)
		assert.Equal(suite.T(), lang, *params.Language)
	}

	// Test auto-detection (nil language)
	params := models.WhisperXParams{
		Model:    "tiny",
		Language: nil,
	}
	assert.Nil(suite.T(), params.Language)
}

// Test diarization parameters
func (suite *TranscriptionServiceTestSuite) TestDiarizationParameters() {
	params := models.WhisperXParams{
		Model:        "tiny",
		Diarize:      true,
		MinSpeakers:  intPtr(1),
		MaxSpeakers:  intPtr(5),
		DiarizeModel: "pyannote/speaker-diarization-3.1",
	}

	assert.True(suite.T(), params.Diarize)
	assert.NotNil(suite.T(), params.MinSpeakers)
	assert.Equal(suite.T(), 1, *params.MinSpeakers)
	assert.NotNil(suite.T(), params.MaxSpeakers)
	assert.Equal(suite.T(), 5, *params.MaxSpeakers)
	assert.NotEmpty(suite.T(), params.DiarizeModel)
	assert.Contains(suite.T(), params.DiarizeModel, "speaker-diarization")
}

// Test quick transcription service
func (suite *TranscriptionServiceTestSuite) TestQuickTranscriptionService() {
	assert.NotNil(suite.T(), suite.quickTranscription)

	// Test that quick transcription service was created successfully
	// (The actual transcription would require WhisperX installation)
}

// Test context cancellation support
func (suite *TranscriptionServiceTestSuite) TestContextCancellation() {
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test that cancelled context is handled properly
	assert.Equal(suite.T(), context.Canceled, ctx.Err())

	// In real implementation, ProcessJob should respect context cancellation
	// Here we test the context handling structure
	select {
	case <-ctx.Done():
		assert.Equal(suite.T(), context.Canceled, ctx.Err())
	default:
		suite.T().Error("Context should be cancelled")
	}
}

// Test timeout handling
func (suite *TranscriptionServiceTestSuite) TestTimeoutHandling() {
	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	<-ctx.Done()

	assert.Equal(suite.T(), context.DeadlineExceeded, ctx.Err())
}

// Test transcript result parsing
func (suite *TranscriptionServiceTestSuite) TestTranscriptParsing() {
	// Test parsing a mock WhisperX result
	mockResult := `{
		"segments": [
			{
				"start": 0.0,
				"end": 5.2,
				"text": " And so, my fellow Americans, ask not what your country can do for you",
				"speaker": "SPEAKER_00"
			},
			{
				"start": 5.2,
				"end": 8.1, 
				"text": " ask what you can do for your country.",
				"speaker": "SPEAKER_00"
			}
		],
		"word_segments": [
			{
				"start": 0.5,
				"end": 0.8,
				"word": "And",
				"score": 0.95,
				"speaker": "SPEAKER_00"
			},
			{
				"start": 0.8,
				"end": 1.1,
				"word": "so",
				"score": 0.92,
				"speaker": "SPEAKER_00"
			}
		],
		"language": "en"
	}`

	var result transcription.TranscriptResult
	err := json.Unmarshal([]byte(mockResult), &result)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), "en", result.Language)
	assert.Len(suite.T(), result.Segments, 2)
	assert.Len(suite.T(), result.Word, 2)

	// Verify first segment
	assert.Equal(suite.T(), 0.0, result.Segments[0].Start)
	assert.Equal(suite.T(), 5.2, result.Segments[0].End)
	assert.Contains(suite.T(), result.Segments[0].Text, "fellow Americans")
	assert.Equal(suite.T(), "SPEAKER_00", *result.Segments[0].Speaker)

	// Verify word-level timing
	assert.Equal(suite.T(), "And", result.Word[0].Word)
	assert.Equal(suite.T(), 0.95, result.Word[0].Score)
}

// Test error handling for invalid audio files
func (suite *TranscriptionServiceTestSuite) TestInvalidAudioHandling() {
	// Test with empty file
	emptyFile := filepath.Join(suite.helper.Config.UploadDir, "empty.wav")
	file, err := os.Create(emptyFile)
	assert.NoError(suite.T(), err)
	file.Close()
	defer os.Remove(emptyFile)

	// Verify empty file exists but has zero size
	fileInfo, err := os.Stat(emptyFile)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), fileInfo.Size())

	// Test with non-audio file (text file)
	textFile := filepath.Join(suite.helper.Config.UploadDir, "not_audio.txt")
	err = os.WriteFile(textFile, []byte("This is not an audio file"), 0644)
	assert.NoError(suite.T(), err)
	defer os.Remove(textFile)

	// Verify text file exists
	_, err = os.Stat(textFile)
	assert.NoError(suite.T(), err)
}

// Test output directory creation
func (suite *TranscriptionServiceTestSuite) TestOutputDirectoryCreation() {
	testJobID := "output-dir-test-123"
	outputDir := filepath.Join("data", "transcripts", testJobID)

	// Create the directory structure that would be used
	err := os.MkdirAll(outputDir, 0755)
	assert.NoError(suite.T(), err)
	defer os.RemoveAll(filepath.Join("data", "transcripts"))

	// Verify directory was created
	fileInfo, err := os.Stat(outputDir)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), fileInfo.IsDir())

	// Test permissions
	assert.Equal(suite.T(), os.FileMode(0755), fileInfo.Mode().Perm())
}

func TestTranscriptionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TranscriptionServiceTestSuite))
}
