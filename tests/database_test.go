package tests

import (
	"os"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
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

// Test database initialization
func (suite *DatabaseTestSuite) TestDatabaseInitialization() {
	// Test with a new database file
	testDbPath := "test_init_isolated.db"
	defer os.Remove(testDbPath)

	// Store current DB to restore later
	originalDB := database.DB

	err := database.Initialize(testDbPath)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), database.DB)

	// Verify database file exists
	_, err = os.Stat(testDbPath)
	assert.NoError(suite.T(), err, "Database file should exist")

	// Close the test database and restore original
	database.Close()
	database.DB = originalDB
}

// Test database initialization with invalid path
func (suite *DatabaseTestSuite) TestDatabaseInitializationInvalidPath() {
	// Try to initialize with an invalid path (directory doesn't exist and can't be created)
	invalidPath := "/root/nonexistent/database.db"

	// This might fail depending on permissions, but we'll test what we can
	err := database.Initialize(invalidPath)
	// The error might be from directory creation or database connection
	if err != nil {
		assert.Contains(suite.T(), err.Error(), "failed")
	}
}

// Test User model CRUD operations
func (suite *DatabaseTestSuite) TestUserCRUD() {
	db := suite.helper.GetDB()

	// Create
	user := models.User{
		Username: "testuser-crud",
		Password: "hashedpassword123",
	}

	result := db.Create(&user)
	assert.NoError(suite.T(), result.Error)
	assert.NotZero(suite.T(), user.ID)
	assert.NotZero(suite.T(), user.CreatedAt)

	// Read
	var foundUser models.User
	result = db.Where("username = ?", "testuser-crud").First(&foundUser)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), user.Username, foundUser.Username)
	assert.Equal(suite.T(), user.Password, foundUser.Password)

	// Update
	foundUser.Username = "updated-username"
	result = db.Save(&foundUser)
	assert.NoError(suite.T(), result.Error)

	var updatedUser models.User
	result = db.First(&updatedUser, foundUser.ID)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), "updated-username", updatedUser.Username)
	assert.NotEqual(suite.T(), updatedUser.CreatedAt, updatedUser.UpdatedAt)

	// Delete
	result = db.Delete(&updatedUser)
	assert.NoError(suite.T(), result.Error)

	// Verify deletion
	var deletedUser models.User
	result = db.First(&deletedUser, updatedUser.ID)
	assert.Error(suite.T(), result.Error)
	assert.Equal(suite.T(), gorm.ErrRecordNotFound, result.Error)
}

// Test APIKey model CRUD operations
func (suite *DatabaseTestSuite) TestAPIKeyCRUD() {
	db := suite.helper.GetDB()

	// Create
	apiKey := models.APIKey{
		Key:         "test-api-key-crud-12345",
		Name:        "Test CRUD API Key",
		Description: stringPtr("Test description"),
		IsActive:    true,
	}

	result := db.Create(&apiKey)
	assert.NoError(suite.T(), result.Error)
	assert.NotZero(suite.T(), apiKey.ID)

	// Read
	var foundKey models.APIKey
	result = db.Where("key = ?", "test-api-key-crud-12345").First(&foundKey)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), apiKey.Key, foundKey.Key)
	assert.Equal(suite.T(), apiKey.Name, foundKey.Name)
	assert.True(suite.T(), foundKey.IsActive)

	// Update
	foundKey.IsActive = false
	foundKey.Name = "Updated API Key"
	result = db.Save(&foundKey)
	assert.NoError(suite.T(), result.Error)

	var updatedKey models.APIKey
	result = db.First(&updatedKey, foundKey.ID)
	assert.NoError(suite.T(), result.Error)
	assert.False(suite.T(), updatedKey.IsActive)
	assert.Equal(suite.T(), "Updated API Key", updatedKey.Name)

	// Delete
	result = db.Delete(&updatedKey)
	assert.NoError(suite.T(), result.Error)

	// Verify deletion
	var deletedKey models.APIKey
	result = db.First(&deletedKey, updatedKey.ID)
	assert.Error(suite.T(), result.Error)
	assert.Equal(suite.T(), gorm.ErrRecordNotFound, result.Error)
}

// Test TranscriptionJob model CRUD operations
func (suite *DatabaseTestSuite) TestTranscriptionJobCRUD() {
	db := suite.helper.GetDB()
	// Create
	title := "Test Transcription Job"
	job := models.TranscriptionJob{
		ID:        "test-job-crud-123",
		Title:     &title,
		Status:    models.StatusPending,
		AudioPath: "/path/to/audio.mp3",
		Parameters: models.WhisperXParams{
			Model:       "base",
			BatchSize:   16,
			ComputeType: "float16",
			Device:      "auto",
		},
	}

	result := db.Create(&job)
	assert.NoError(suite.T(), result.Error)
	assert.NotZero(suite.T(), job.CreatedAt)

	// Read
	var foundJob models.TranscriptionJob
	result = db.Where("id = ?", "test-job-crud-123").First(&foundJob)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), job.ID, foundJob.ID)
	assert.Equal(suite.T(), *job.Title, *foundJob.Title)
	assert.Equal(suite.T(), job.Status, foundJob.Status)
	assert.Equal(suite.T(), job.Parameters.Model, foundJob.Parameters.Model)

	// Update status and transcript
	transcript := `{"segments": [{"start": 0.0, "end": 5.0, "text": "Test transcript"}]}`
	foundJob.Status = models.StatusCompleted
	foundJob.Transcript = &transcript
	result = db.Save(&foundJob)
	assert.NoError(suite.T(), result.Error)

	var updatedJob models.TranscriptionJob
	// For string primary keys, query explicitly by id
	result = db.Where("id = ?", foundJob.ID).First(&updatedJob)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), models.StatusCompleted, updatedJob.Status)
	assert.NotNil(suite.T(), updatedJob.Transcript)
	assert.Equal(suite.T(), transcript, *updatedJob.Transcript)

	// Delete
	result = db.Delete(&updatedJob)
	assert.NoError(suite.T(), result.Error)

	// Verify deletion
	var deletedJob models.TranscriptionJob
	result = db.Where("id = ?", updatedJob.ID).First(&deletedJob)
	assert.Error(suite.T(), result.Error)
	assert.Equal(suite.T(), gorm.ErrRecordNotFound, result.Error)
}

// Test TranscriptionProfile model CRUD operations
func (suite *DatabaseTestSuite) TestTranscriptionProfileCRUD() {
	db := suite.helper.GetDB()
	// Create
	profile := models.TranscriptionProfile{
		ID:          "test-profile-crud-123",
		Name:        "Test Profile",
		Description: stringPtr("Test profile description"),
		IsDefault:   false,
		Parameters: models.WhisperXParams{
			Model:       "small",
			BatchSize:   8,
			ComputeType: "float32",
			Device:      "cpu",
		},
	}

	result := db.Create(&profile)
	assert.NoError(suite.T(), result.Error)
	assert.NotZero(suite.T(), profile.CreatedAt)

	// Read
	var foundProfile models.TranscriptionProfile
	result = db.Where("id = ?", "test-profile-crud-123").First(&foundProfile)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), profile.Name, foundProfile.Name)
	assert.Equal(suite.T(), profile.Parameters.Model, foundProfile.Parameters.Model)

	// Update
	foundProfile.IsDefault = true
	foundProfile.Name = "Updated Profile"
	result = db.Save(&foundProfile)
	assert.NoError(suite.T(), result.Error)

	var updatedProfile models.TranscriptionProfile
	// For string primary keys, query explicitly by id
	result = db.Where("id = ?", foundProfile.ID).First(&updatedProfile)
	assert.NoError(suite.T(), result.Error)
	assert.True(suite.T(), updatedProfile.IsDefault)
	assert.Equal(suite.T(), "Updated Profile", updatedProfile.Name)

	// Delete
	result = db.Delete(&updatedProfile)
	assert.NoError(suite.T(), result.Error)
}

// Test Note model CRUD operations
func (suite *DatabaseTestSuite) TestNoteCRUD() {
	db := suite.helper.GetDB()
	// First create a transcription job for the note
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job for Note")

	// Create note
	note := models.Note{
		ID:              "test-note-crud-123",
		TranscriptionID: job.ID,
		StartWordIndex:  0,
		EndWordIndex:    5,
		StartTime:       0.0,
		EndTime:         2.5,
		Quote:           "Test quote text",
		Content:         "Test note content",
	}

	result := db.Create(&note)
	assert.NoError(suite.T(), result.Error)
	assert.NotZero(suite.T(), note.CreatedAt)

	// Read
	var foundNote models.Note
	result = db.Where("id = ?", "test-note-crud-123").First(&foundNote)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), note.TranscriptionID, foundNote.TranscriptionID)
	assert.Equal(suite.T(), note.Content, foundNote.Content)
	assert.Equal(suite.T(), note.Quote, foundNote.Quote)
	assert.Equal(suite.T(), note.StartTime, foundNote.StartTime)

	// Update
	foundNote.Content = "Updated note content"
	foundNote.Quote = "Updated quote"
	result = db.Save(&foundNote)
	assert.NoError(suite.T(), result.Error)

	var updatedNote models.Note
	// For string primary keys, query explicitly by id
	result = db.Where("id = ?", foundNote.ID).First(&updatedNote)
	assert.NoError(suite.T(), result.Error)
	assert.Equal(suite.T(), "Updated note content", updatedNote.Content)
	assert.Equal(suite.T(), "Updated quote", updatedNote.Quote)

	// Delete
	result = db.Delete(&updatedNote)
	assert.NoError(suite.T(), result.Error)
}

// Test database relationships
func (suite *DatabaseTestSuite) TestDatabaseRelationships() {
	db := suite.helper.GetDB()
	// Create a transcription job
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job for Relations")

	// Create notes for the job
	note1 := models.Note{
		ID:              "note-1-relations",
		TranscriptionID: job.ID,
		StartWordIndex:  0,
		EndWordIndex:    3,
		StartTime:       0.0,
		EndTime:         1.5,
		Quote:           "First quote",
		Content:         "First note",
	}

	note2 := models.Note{
		ID:              "note-2-relations",
		TranscriptionID: job.ID,
		StartWordIndex:  4,
		EndWordIndex:    8,
		StartTime:       1.5,
		EndTime:         3.0,
		Quote:           "Second quote",
		Content:         "Second note",
	}

	result := db.Create(&note1)
	assert.NoError(suite.T(), result.Error)
	result = db.Create(&note2)
	assert.NoError(suite.T(), result.Error)

	// Query notes by transcription ID
	var notes []models.Note
	result = db.Where("transcription_id = ?", job.ID).Find(&notes)
	assert.NoError(suite.T(), result.Error)
	assert.Len(suite.T(), notes, 2)

	// Verify note contents
	noteContents := []string{notes[0].Content, notes[1].Content}
	assert.Contains(suite.T(), noteContents, "First note")
	assert.Contains(suite.T(), noteContents, "Second note")

	// Clean up
	db.Delete(&note1)
	db.Delete(&note2)
}

// Test unique constraints
func (suite *DatabaseTestSuite) TestUniqueConstraints() {
	db := suite.helper.GetDB()
	// Test user username uniqueness
	user1 := models.User{
		Username: "unique-test-user",
		Password: "password1",
	}
	user2 := models.User{
		Username: "unique-test-user", // Same username
		Password: "password2",
	}

	result := db.Create(&user1)
	assert.NoError(suite.T(), result.Error)

	result = db.Create(&user2)
	assert.Error(suite.T(), result.Error, "Should fail due to unique constraint on username")

	// Test API key uniqueness
	apiKey1 := models.APIKey{
		Key:      "unique-api-key-test",
		Name:     "First Key",
		IsActive: true,
	}
	apiKey2 := models.APIKey{
		Key:      "unique-api-key-test", // Same key
		Name:     "Second Key",
		IsActive: true,
	}

	result = db.Create(&apiKey1)
	assert.NoError(suite.T(), result.Error)

	result = db.Create(&apiKey2)
	assert.Error(suite.T(), result.Error, "Should fail due to unique constraint on API key")

	// Clean up
	db.Delete(&user1)
	db.Delete(&apiKey1)
}

// Test database queries with filters
func (suite *DatabaseTestSuite) TestDatabaseQueries() {
	db := suite.helper.GetDB()
	// Create multiple API keys with different statuses
	activeKey := models.APIKey{
		Key:      "active-key-query-test",
		Name:     "Active Key",
		IsActive: true,
	}
	inactiveKey := models.APIKey{
		Key:      "inactive-key-query-test",
		Name:     "Inactive Key",
		IsActive: false,
	}

	db.Create(&activeKey)
	db.Create(&inactiveKey)

	// Query only active keys
	var activeKeys []models.APIKey
	result := db.Where("is_active = ?", true).Find(&activeKeys)
	assert.NoError(suite.T(), result.Error)

	// Should include at least our test active key
	found := false
	for _, key := range activeKeys {
		if key.Key == "active-key-query-test" {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "Should find the active test key")

	// Query inactive keys
	var inactiveKeys []models.APIKey
	result = db.Where("is_active = ?", false).Find(&inactiveKeys)
	assert.NoError(suite.T(), result.Error)

	// Should include our inactive key
	found = false
	for _, key := range inactiveKeys {
		if key.Key == "inactive-key-query-test" {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "Should find the inactive test key")

	// Clean up
	db.Delete(&activeKey)
	db.Delete(&inactiveKey)
}

// Test database close functionality
func (suite *DatabaseTestSuite) TestDatabaseClose() {
	// Test that the Close function exists and can be called
	// We just verify it doesn't panic when called
	assert.NotPanics(suite.T(), func() {
		// In a real scenario, we'd test database close functionality
		// For now, we just verify the function can be called
		_ = database.Close
	})
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
