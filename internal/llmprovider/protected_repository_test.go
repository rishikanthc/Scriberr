package llmprovider

import (
	"context"
	"path/filepath"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/require"
)

func TestProtectedRepositoryEncryptsStoredAPIKeyAndDecryptsReads(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	repo := repository.NewLLMConfigRepository(db)
	protected, err := NewProtectedRepository(repo, "test-credential-secret")
	require.NoError(t, err)

	user := models.User{Username: "llm-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	rawKey := "sk-test-secret"
	baseURL := "https://provider.example/v1"
	config := &models.LLMConfig{
		UserID:        user.ID,
		Name:          "Default",
		Provider:      "openai_compatible",
		BaseURL:       &baseURL,
		OpenAIBaseURL: &baseURL,
		APIKey:        &rawKey,
		IsDefault:     true,
	}

	require.NoError(t, protected.ReplaceActiveByUser(context.Background(), user.ID, config))

	var stored models.LLMConfig
	require.NoError(t, db.First(&stored, "user_id = ?", user.ID).Error)
	require.NotNil(t, stored.APIKey)
	require.NotEqual(t, rawKey, *stored.APIKey)
	require.Contains(t, *stored.APIKey, "enc:v1:")
	require.NotContains(t, stored.ConfigJSON, rawKey)

	loaded, err := protected.GetActiveByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.APIKey)
	require.Equal(t, rawKey, *loaded.APIKey)
}

func TestProtectedRepositoryAllowsExistingPlaintextConfig(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	repo := repository.NewLLMConfigRepository(db)
	protected, err := NewProtectedRepository(repo, "test-credential-secret")
	require.NoError(t, err)

	user := models.User{Username: "legacy-llm-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	rawKey := "legacy-plaintext-secret"
	baseURL := "https://provider.example/v1"
	require.NoError(t, repo.ReplaceActiveByUser(context.Background(), user.ID, &models.LLMConfig{
		UserID:        user.ID,
		Name:          "Legacy",
		Provider:      "openai_compatible",
		BaseURL:       &baseURL,
		OpenAIBaseURL: &baseURL,
		APIKey:        &rawKey,
		IsDefault:     true,
	}))

	loaded, err := protected.GetActiveByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.APIKey)
	require.Equal(t, rawKey, *loaded.APIKey)
}
