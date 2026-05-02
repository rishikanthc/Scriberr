package tags

import (
	"context"
	"path/filepath"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingPublisher struct {
	events []Event
}

func (p *recordingPublisher) PublishTagEvent(_ context.Context, event Event) {
	p.events = append(p.events, event)
}

func openTagServiceTestDB(t *testing.T) (*Service, *recordingPublisher, models.User, models.TranscriptionJob) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	user := models.User{Username: "tag-service-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	title := "tag service transcript"
	sourceID := "source-file"
	job := models.TranscriptionJob{
		ID:             "tag-service-job",
		UserID:         user.ID,
		Title:          &title,
		Status:         models.StatusCompleted,
		AudioPath:      filepath.Join(t.TempDir(), "audio.wav"),
		SourceFileName: "audio.wav",
		SourceFileHash: &sourceID,
	}
	require.NoError(t, db.Create(&job).Error)

	service := NewService(repository.NewTagRepository(db), repository.NewJobRepository(db))
	publisher := &recordingPublisher{}
	service.SetEventPublisher(publisher)
	return service, publisher, user, job
}

func TestServiceCreateListUpdateDeleteTag(t *testing.T) {
	service, publisher, user, _ := openTagServiceTestDB(t)
	color := " #E87539 "
	description := " customer calls "

	created, err := service.CreateTag(context.Background(), CreateRequest{
		UserID:      user.ID,
		Name:        " Client   Call ",
		Color:       &color,
		Description: &description,
	})
	require.NoError(t, err)
	assert.Equal(t, "Client Call", created.Name)
	assert.Equal(t, "client call", created.NormalizedName)
	require.NotNil(t, created.Color)
	assert.Equal(t, "#E87539", *created.Color)
	require.NotNil(t, created.Description)
	assert.Equal(t, "customer calls", *created.Description)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, "tag.created", publisher.events[0].Name)
	assert.Equal(t, PublicTagID(created.ID), publisher.events[0].TagID)

	_, err = service.CreateTag(context.Background(), CreateRequest{UserID: user.ID, Name: "client call"})
	require.ErrorIs(t, err, ErrConflict)

	items, count, err := service.ListTags(context.Background(), ListRequest{UserID: user.ID, Search: "client", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	require.Len(t, items, 1)
	assert.Equal(t, created.ID, items[0].ID)

	updatedName := "Customer Call"
	clearColor := ""
	updated, err := service.UpdateTag(context.Background(), UpdateRequest{
		UserID: user.ID,
		TagID:  PublicTagID(created.ID),
		Name:   &updatedName,
		Color:  &clearColor,
	})
	require.NoError(t, err)
	assert.Equal(t, "Customer Call", updated.Name)
	assert.Equal(t, "customer call", updated.NormalizedName)
	assert.Nil(t, updated.Color)

	got, err := service.GetTag(context.Background(), user.ID, PublicTagID(created.ID))
	require.NoError(t, err)
	assert.Equal(t, updated.ID, got.ID)

	require.NoError(t, service.DeleteTag(context.Background(), user.ID, PublicTagID(created.ID)))
	_, err = service.GetTag(context.Background(), user.ID, PublicTagID(created.ID))
	require.ErrorIs(t, err, ErrNotFound)

	require.Len(t, publisher.events, 3)
	assert.Equal(t, "tag.deleted", publisher.events[2].Name)
}

func TestServiceAssignReplaceRemoveTranscriptionTags(t *testing.T) {
	service, publisher, user, job := openTagServiceTestDB(t)
	first, err := service.CreateTag(context.Background(), CreateRequest{UserID: user.ID, Name: "Research"})
	require.NoError(t, err)
	second, err := service.CreateTag(context.Background(), CreateRequest{UserID: user.ID, Name: "Meeting"})
	require.NoError(t, err)

	items, err := service.AddTagToTranscription(context.Background(), TranscriptionTagRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		TagID:           PublicTagID(first.ID),
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, first.ID, items[0].ID)

	items, err = service.ReplaceTranscriptionTags(context.Background(), ReplaceTranscriptionTagsRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		TagIDs:          []string{PublicTagID(first.ID), PublicTagID(second.ID)},
	})
	require.NoError(t, err)
	require.Len(t, items, 2)

	ids, err := service.TranscriptionIDsByTags(context.Background(), FilterRequest{
		UserID:   user.ID,
		TagRefs:  []string{"research", PublicTagID(second.ID)},
		MatchAll: true,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{job.ID}, ids)

	require.NoError(t, service.RemoveTagFromTranscription(context.Background(), TranscriptionTagRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		TagID:           PublicTagID(first.ID),
	}))
	items, err = service.ListTranscriptionTags(context.Background(), user.ID, "tr_"+job.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, second.ID, items[0].ID)

	assert.Equal(t, "transcription.tags.updated", publisher.events[len(publisher.events)-1].Name)
	assert.Equal(t, "tr_"+job.ID, publisher.events[len(publisher.events)-1].TranscriptionID)
}

func TestServiceRejectsInvalidTagRequests(t *testing.T) {
	service, _, user, job := openTagServiceTestDB(t)

	_, err := service.CreateTag(context.Background(), CreateRequest{UserID: user.ID, Name: ""})
	require.ErrorIs(t, err, ErrValidation)

	badColor := "orange"
	_, err = service.CreateTag(context.Background(), CreateRequest{UserID: user.ID, Name: "Bad color", Color: &badColor})
	require.ErrorIs(t, err, ErrValidation)

	_, err = service.GetTag(context.Background(), user.ID, "missing")
	require.ErrorIs(t, err, ErrValidation)

	_, err = service.ListTranscriptionTags(context.Background(), user.ID, job.ID)
	require.ErrorIs(t, err, ErrValidation)

	_, err = service.ReplaceTranscriptionTags(context.Background(), ReplaceTranscriptionTagsRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		TagIDs:          []string{"tag_missing"},
	})
	require.ErrorIs(t, err, ErrNotFound)
}
