package annotations

import (
	"context"
	"errors"
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

func (p *recordingPublisher) PublishAnnotationEvent(_ context.Context, event Event) {
	p.events = append(p.events, event)
}

func openAnnotationServiceTestDB(t *testing.T) (*Service, *recordingPublisher, models.User, models.TranscriptionJob) {
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

	user := models.User{Username: "annotation-service-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	title := "annotation service transcript"
	sourceID := "source-file"
	job := models.TranscriptionJob{
		ID:             "annotation-service-job",
		UserID:         user.ID,
		Title:          &title,
		Status:         models.StatusCompleted,
		AudioPath:      filepath.Join(t.TempDir(), "audio.wav"),
		SourceFileName: "audio.wav",
		SourceFileHash: &sourceID,
	}
	require.NoError(t, db.Create(&job).Error)

	service := NewService(repository.NewAnnotationRepository(db), repository.NewJobRepository(db))
	publisher := &recordingPublisher{}
	service.SetEventPublisher(publisher)
	return service, publisher, user, job
}

func TestServiceCreateListUpdateDeleteAnnotation(t *testing.T) {
	service, publisher, user, job := openAnnotationServiceTestDB(t)
	content := " follow up "
	color := " yellow "
	startWord := 4
	endWord := 7

	created, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Color:           &color,
		Quote:           " quoted text ",
		Anchor: Anchor{
			StartMS:   1200,
			EndMS:     2400,
			StartWord: &startWord,
			EndWord:   &endWord,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotNil(t, created.Content)
	require.NotNil(t, created.Color)
	assert.Equal(t, "follow up", *created.Content)
	assert.Equal(t, "yellow", *created.Color)
	assert.Equal(t, "quoted text", created.Quote)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, "annotation.created", publisher.events[0].Name)
	assert.Equal(t, PublicAnnotationID(created.ID), publisher.events[0].AnnotationID)

	kind := models.AnnotationKindNote
	items, count, err := service.ListAnnotations(context.Background(), ListRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            &kind,
		Limit:           10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	require.Len(t, items, 1)
	assert.Equal(t, created.ID, items[0].ID)

	updatedContent := "updated note"
	updatedQuote := "updated quote"
	updated, err := service.UpdateAnnotation(context.Background(), UpdateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		AnnotationID:    PublicAnnotationID(created.ID),
		Content:         &updatedContent,
		Quote:           &updatedQuote,
		Anchor:          &Anchor{StartMS: 1300, EndMS: 2600},
	})
	require.NoError(t, err)
	require.NotNil(t, updated.Content)
	assert.Equal(t, "updated note", *updated.Content)
	assert.Equal(t, "updated quote", updated.Quote)
	assert.Equal(t, int64(1300), updated.AnchorStartMS)

	got, err := service.GetAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID))
	require.NoError(t, err)
	assert.Equal(t, updated.ID, got.ID)

	require.NoError(t, service.DeleteAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID)))
	_, err = service.GetAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID))
	require.ErrorIs(t, err, ErrNotFound)

	require.Len(t, publisher.events, 3)
	assert.Equal(t, "annotation.deleted", publisher.events[2].Name)
}

func TestServiceRejectsInvalidIDsAndAnchors(t *testing.T) {
	service, _, user, job := openAnnotationServiceTestDB(t)
	content := "note"

	_, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Quote:           "quote",
		Anchor:          Anchor{StartMS: 100, EndMS: 200},
	})
	require.ErrorIs(t, err, ErrValidation)

	_, err = service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKind("bookmark"),
		Content:         &content,
		Quote:           "quote",
		Anchor:          Anchor{StartMS: 100, EndMS: 200},
	})
	require.ErrorIs(t, err, ErrValidation)

	_, err = service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Quote:           "quote",
		Anchor:          Anchor{StartMS: 300, EndMS: 200},
	})
	require.ErrorIs(t, err, ErrValidation)

	emptyContent := " "
	_, err = service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &emptyContent,
		Quote:           "quote",
		Anchor:          Anchor{StartMS: 100, EndMS: 200},
	})
	require.ErrorIs(t, err, ErrValidation)
}

func TestServiceScopesAnnotationsByTranscriptionOwnership(t *testing.T) {
	service, _, user, job := openAnnotationServiceTestDB(t)
	otherUserID := user.ID + 100
	content := "note"

	_, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          otherUserID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Quote:           "quote",
		Anchor:          Anchor{StartMS: 100, EndMS: 200},
	})
	require.ErrorIs(t, err, ErrNotFound)

	created, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Quote:           "quote",
		Anchor:          Anchor{StartMS: 100, EndMS: 200},
	})
	require.NoError(t, err)

	_, err = service.GetAnnotation(context.Background(), otherUserID, "tr_"+job.ID, PublicAnnotationID(created.ID))
	require.True(t, errors.Is(err, ErrNotFound), "got %v", err)
}
