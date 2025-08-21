package tests

import (
	"encoding/json"
	"testing"
	"time"

	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestTranscriptionJob_BeforeCreate(t *testing.T) {
	job := &models.TranscriptionJob{
		Title:     stringPtr("Test Job"),
		AudioPath: "/path/to/audio.mp3",
		Status:    models.StatusPending,
	}
	
	// ID should be empty initially
	assert.Empty(t, job.ID)
	
	// Simulate GORM BeforeCreate hook
	err := job.BeforeCreate(nil)
	assert.NoError(t, err)
	
	// ID should be generated
	assert.NotEmpty(t, job.ID)
	assert.Len(t, job.ID, 36) // UUID length
}

func TestAPIKey_BeforeCreate(t *testing.T) {
	apiKey := &models.APIKey{
		Name:        "Test API Key",
		Description: stringPtr("Test description"),
		IsActive:    true,
	}
	
	// Key should be empty initially
	assert.Empty(t, apiKey.Key)
	
	// Simulate GORM BeforeCreate hook
	err := apiKey.BeforeCreate(nil)
	assert.NoError(t, err)
	
	// Key should be generated
	assert.NotEmpty(t, apiKey.Key)
	assert.Len(t, apiKey.Key, 36) // UUID length
}

func TestJobStatus_Values(t *testing.T) {
	assert.Equal(t, "pending", string(models.StatusPending))
	assert.Equal(t, "processing", string(models.StatusProcessing))
	assert.Equal(t, "completed", string(models.StatusCompleted))
	assert.Equal(t, "failed", string(models.StatusFailed))
}

func TestWhisperXParams_Defaults(t *testing.T) {
	params := models.WhisperXParams{
		Model:       "base",
		BatchSize:   16,
		ComputeType: "float16",
		Device:      "auto",
		VadFilter:   false,
		VadOnset:    0.500,
		VadOffset:   0.363,
	}
	
	assert.Equal(t, "base", params.Model)
	assert.Equal(t, 16, params.BatchSize)
	assert.Equal(t, "float16", params.ComputeType)
	assert.Equal(t, "auto", params.Device)
	assert.False(t, params.VadFilter)
	assert.Equal(t, 0.500, params.VadOnset)
	assert.Equal(t, 0.363, params.VadOffset)
}

func TestTranscriptionJob_JSON_Serialization(t *testing.T) {
	transcript := `{"segments": [{"start": 0.0, "end": 5.0, "text": "Test"}], "language": "en"}`
	
	job := models.TranscriptionJob{
		ID:         "test-job-123",
		Title:      stringPtr("Test Job"),
		Status:     models.StatusCompleted,
		AudioPath:  "/path/to/audio.mp3",
		Transcript: &transcript,
		Diarization: true,
		Summary:    stringPtr("Test summary"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Parameters: models.WhisperXParams{
			Model:       "base",
			Language:    stringPtr("en"),
			BatchSize:   16,
			ComputeType: "float16",
			Device:      "auto",
			VadFilter:   true,
			VadOnset:    0.500,
			VadOffset:   0.363,
			MinSpeakers: intPtr(2),
			MaxSpeakers: intPtr(5),
		},
	}
	
	// Test JSON serialization
	jsonData, err := json.Marshal(job)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	
	// Test JSON deserialization
	var deserializedJob models.TranscriptionJob
	err = json.Unmarshal(jsonData, &deserializedJob)
	assert.NoError(t, err)
	
	assert.Equal(t, job.ID, deserializedJob.ID)
	assert.Equal(t, job.Status, deserializedJob.Status)
	assert.Equal(t, job.AudioPath, deserializedJob.AudioPath)
	assert.Equal(t, job.Diarization, deserializedJob.Diarization)
	assert.Equal(t, *job.Title, *deserializedJob.Title)
	assert.Equal(t, *job.Summary, *deserializedJob.Summary)
	assert.Equal(t, *job.Transcript, *deserializedJob.Transcript)
	assert.Equal(t, job.Parameters.Model, deserializedJob.Parameters.Model)
	assert.Equal(t, *job.Parameters.Language, *deserializedJob.Parameters.Language)
	assert.Equal(t, *job.Parameters.MinSpeakers, *deserializedJob.Parameters.MinSpeakers)
	assert.Equal(t, *job.Parameters.MaxSpeakers, *deserializedJob.Parameters.MaxSpeakers)
}

func TestUser_JSON_Serialization(t *testing.T) {
	user := models.User{
		ID:       1,
		Username: "testuser",
		Password: "hashedpassword", // Should not appear in JSON
	}
	
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)
	
	// Check that password is not included in JSON
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)
	
	assert.Equal(t, float64(1), jsonMap["id"])
	assert.Equal(t, "testuser", jsonMap["username"])
	assert.NotContains(t, jsonMap, "password")
}

func intPtr(i int) *int {
	return &i
}