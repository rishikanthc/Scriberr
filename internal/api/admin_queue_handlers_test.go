package api

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/require"
)

func TestAdminQueueStatsAreGlobalWithUserBreakdown(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)
	adminID := currentTestUserID(t, "admin")
	member := models.User{Username: "queue-member", Password: "pw", Role: "user"}
	require.NoError(t, repository.NewUserRepository(database.DB).Create(context.Background(), &member))
	now := time.Now().Truncate(time.Millisecond)
	createAdminQueueTestJob(t, adminID, "admin-queued", models.StatusPending, now)
	createAdminQueueTestJob(t, member.ID, "member-processing", models.StatusProcessing, now)
	createAdminQueueTestJob(t, member.ID, "member-failed", models.StatusFailed, now)

	resp, body := s.request(t, http.MethodGet, "/api/v1/admin/queue", nil, adminToken, "")

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, float64(1), body["queued"])
	require.Equal(t, float64(1), body["processing"])
	require.Equal(t, float64(1), body["failed"])
	items := body["by_user"].([]any)
	require.Len(t, items, 2)
	byUsername := map[string]map[string]any{}
	for _, item := range items {
		row := item.(map[string]any)
		byUsername[row["username"].(string)] = row
	}
	require.Equal(t, float64(1), byUsername["admin"]["queued"])
	require.Equal(t, float64(1), byUsername["queue-member"]["processing"])
	require.Equal(t, float64(1), byUsername["queue-member"]["failed"])
}

func TestUserQueueStatsRemainScoped(t *testing.T) {
	s := newAuthTestServer(t)
	registerForFileTests(t, s)
	adminID := currentTestUserID(t, "admin")
	member := models.User{Username: "scoped-queue-member", Password: "pw", Role: "user"}
	require.NoError(t, repository.NewUserRepository(database.DB).Create(context.Background(), &member))
	now := time.Now().Truncate(time.Millisecond)
	createAdminQueueTestJob(t, adminID, "scoped-admin-queued", models.StatusPending, now)
	createAdminQueueTestJob(t, member.ID, "scoped-member-queued", models.StatusPending, now)

	stats, err := s.handler.transcriptions.Stats(context.Background(), adminID)

	require.NoError(t, err)
	require.Equal(t, int64(1), stats.Queued)
}

func createAdminQueueTestJob(t *testing.T, userID uint, id string, status models.JobStatus, now time.Time) {
	t.Helper()
	title := id
	sourceID := "file-" + id
	job := models.TranscriptionJob{
		ID:             id,
		UserID:         userID,
		Title:          &title,
		Status:         status,
		AudioPath:      filepath.Join(t.TempDir(), id+".wav"),
		SourceFileName: id + ".wav",
		SourceFileHash: &sourceID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, database.DB.Create(&job).Error)
}
