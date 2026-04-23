package tests

import (
	"os"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DatabaseTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *DatabaseTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "database_test.db")
}

func (suite *DatabaseTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

func (suite *DatabaseTestSuite) SetupTest() {
	suite.helper.ResetDB(suite.T())
}

func (suite *DatabaseTestSuite) TestDatabaseInitialization() {
	testDbPath := "test_init_isolated.db"
	defer os.Remove(testDbPath)

	originalDB := database.DB
	err := database.Initialize(testDbPath)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), database.DB)

	_, err = os.Stat(testDbPath)
	assert.NoError(suite.T(), err)

	database.Close()
	database.DB = originalDB
}

func (suite *DatabaseTestSuite) TestUserSettingsRoundTrip() {
	db := suite.helper.GetDB()
	profileID := "profile-123"
	user := models.User{
		Username:                 "settings-user",
		Password:                 "hashedpassword123",
		DefaultProfileID:         &profileID,
		AutoTranscriptionEnabled: true,
		SummaryDefaultModel:      "gpt-4o-mini",
	}
	require.NoError(suite.T(), db.Create(&user).Error)

	var found models.User
	require.NoError(suite.T(), db.First(&found, user.ID).Error)
	assert.True(suite.T(), found.AutoTranscriptionEnabled)
	require.NotNil(suite.T(), found.DefaultProfileID)
	assert.Equal(suite.T(), profileID, *found.DefaultProfileID)
	assert.Equal(suite.T(), "gpt-4o-mini", found.SummaryDefaultModel)
}

func (suite *DatabaseTestSuite) TestTranscriptionJobPersistsCompatibilityFields() {
	db := suite.helper.GetDB()
	title := "Persisted Job"
	transcript := `{"text":"hello"}`
	summary := "cached summary"
	merged := "/tmp/merged.wav"
	individual := `{"Speaker 1":"hello"}`

	job := models.TranscriptionJob{
		ID:                    "job-compat-1",
		Title:                 &title,
		Status:                models.StatusCompleted,
		AudioPath:             "/tmp/input.wav",
		Transcript:            &transcript,
		Summary:               &summary,
		IsMultiTrack:          true,
		MergedAudioPath:       &merged,
		MergeStatus:           "completed",
		IndividualTranscripts: &individual,
		Parameters: models.WhisperXParams{
			Model:       "base",
			ModelFamily: "whisper",
			Device:      "cpu",
			ComputeType: "float32",
			Diarize:     true,
		},
	}

	require.NoError(suite.T(), db.Create(&job).Error)

	var found models.TranscriptionJob
	require.NoError(suite.T(), db.First(&found, "id = ?", job.ID).Error)
	require.NotNil(suite.T(), found.Transcript)
	assert.Equal(suite.T(), transcript, *found.Transcript)
	require.NotNil(suite.T(), found.Summary)
	assert.Equal(suite.T(), summary, *found.Summary)
	assert.True(suite.T(), found.IsMultiTrack)
	require.NotNil(suite.T(), found.MergedAudioPath)
	assert.Equal(suite.T(), merged, *found.MergedAudioPath)
	assert.Equal(suite.T(), "base", found.Parameters.Model)
	assert.True(suite.T(), found.Parameters.Diarize)
	require.NotNil(suite.T(), found.CompletedAt)
}

func (suite *DatabaseTestSuite) TestAPIKeyRepositoryUsesHashedStorage() {
	db := suite.helper.GetDB()
	repo := repository.NewAPIKeyRepository(db)

	rawKey := "test-api-key-crud-12345"
	apiKey := models.APIKey{
		UserID:      suite.helper.TestUser.ID,
		Key:         rawKey,
		KeyPrefix:   rawKey[:8],
		KeyHash:     sha256Hex(rawKey),
		Name:        "Test CRUD API Key",
		Description: stringPtr("Test description"),
		IsActive:    true,
	}

	require.NoError(suite.T(), db.Create(&apiKey).Error)

	found, err := repo.FindByKey(suite.T().Context(), rawKey)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), apiKey.Name, found.Name)
	assert.Equal(suite.T(), "Test description", *found.Description)
	assert.True(suite.T(), found.IsActive)

	require.NoError(suite.T(), repo.Revoke(suite.T().Context(), apiKey.ID))

	var reloaded models.APIKey
	require.NoError(suite.T(), db.First(&reloaded, apiKey.ID).Error)
	assert.NotNil(suite.T(), reloaded.RevokedAt)
}

func (suite *DatabaseTestSuite) TestNoteCompatibilityMetadataRoundTrip() {
	db := suite.helper.GetDB()
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job for Note")

	note := models.Note{
		ID:              "test-note-crud-123",
		TranscriptionID: job.ID,
		StartWordIndex:  0,
		EndWordIndex:    5,
		StartTime:       1.25,
		EndTime:         2.50,
		Quote:           "Test quote text",
		Content:         "Test note content",
	}

	require.NoError(suite.T(), db.Create(&note).Error)

	var found models.Note
	require.NoError(suite.T(), db.First(&found, "id = ?", note.ID).Error)
	assert.Equal(suite.T(), int64(1250), found.StartMS)
	assert.Equal(suite.T(), int64(2500), found.EndMS)
	assert.Equal(suite.T(), note.Quote, found.Quote)
	assert.Equal(suite.T(), note.StartWordIndex, found.StartWordIndex)
	assert.Equal(suite.T(), note.EndWordIndex, found.EndWordIndex)
}

func (suite *DatabaseTestSuite) TestRefreshTokenRevocationSemantics() {
	db := suite.helper.GetDB()
	repo := repository.NewRefreshTokenRepository(db)

	token := &models.RefreshToken{
		UserID:    suite.helper.TestUser.ID,
		Hashed:    sha256Hex("refresh-token"),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(suite.T(), repo.Create(suite.T().Context(), token))

	found, err := repo.FindByHash(suite.T().Context(), token.Hashed)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), found.Revoked)

	require.NoError(suite.T(), repo.Revoke(suite.T().Context(), token.ID))
	found, err = repo.FindByHash(suite.T().Context(), token.Hashed)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), found.Revoked)
}

func (suite *DatabaseTestSuite) TestDatabaseConstraints() {
	db := suite.helper.GetDB()
	title := "Constraint job"
	job := models.TranscriptionJob{Title: &title, Status: models.StatusUploaded, AudioPath: "/tmp/audio.wav"}
	require.NoError(suite.T(), db.Create(&job).Error)

	mapping := models.SpeakerMapping{TranscriptionJobID: job.ID, OriginalSpeaker: "speaker_00", CustomName: "Alice"}
	require.NoError(suite.T(), db.Create(&mapping).Error)
	duplicate := models.SpeakerMapping{TranscriptionJobID: job.ID, OriginalSpeaker: "speaker_00", CustomName: "Bob"}
	assert.Error(suite.T(), db.Create(&duplicate).Error)

	badNote := models.Note{ID: "bad-note", TranscriptionID: "missing-job", Content: "bad"}
	assert.Error(suite.T(), db.Create(&badNote).Error)
}

func (suite *DatabaseTestSuite) TestDatabaseClose() {
	assert.NotPanics(suite.T(), func() {
		_ = database.Close
	})
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
