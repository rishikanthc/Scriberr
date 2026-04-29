package annotations

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/orchestrator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type recordingPublisher struct {
	events []Event
}

func (p *recordingPublisher) PublishAnnotationEvent(_ context.Context, event Event) {
	p.events = append(p.events, event)
}

type mapAccessPolicy struct {
	transcriptions map[string]*models.TranscriptionJob
	allowed        map[uint]map[string]bool
}

func (p mapAccessPolicy) FindAccessibleTranscription(_ context.Context, userID uint, transcriptionID string) (*models.TranscriptionJob, error) {
	if p.allowed[userID][transcriptionID] {
		return p.transcriptions[transcriptionID], nil
	}
	return nil, gorm.ErrRecordNotFound
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
	require.NotNil(t, created.Color)
	require.Len(t, created.Entries, 1)
	assert.Nil(t, created.Content)
	assert.Equal(t, "follow up", created.Entries[0].Content)
	assert.Equal(t, "yellow", *created.Color)
	assert.Equal(t, "quoted text", created.Quote)
	assert.Equal(t, models.AnnotationStatusActive, created.Status)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, "annotation.created", publisher.events[0].Name)
	assert.Equal(t, PublicAnnotationID(created.ID), publisher.events[0].AnnotationID)
	assert.Equal(t, models.AnnotationStatusActive, publisher.events[0].Status)

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
	require.Len(t, items[0].Entries, 1)
	assert.Equal(t, "follow up", items[0].Entries[0].Content)

	updatedQuote := "updated quote"
	updated, err := service.UpdateAnnotation(context.Background(), UpdateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		AnnotationID:    PublicAnnotationID(created.ID),
		Quote:           &updatedQuote,
		Anchor:          &Anchor{StartMS: 1300, EndMS: 2600},
	})
	require.NoError(t, err)
	assert.Nil(t, updated.Content)
	assert.Equal(t, "updated quote", updated.Quote)
	assert.Equal(t, int64(1300), updated.AnchorStartMS)

	got, err := service.GetAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID))
	require.NoError(t, err)
	assert.Equal(t, updated.ID, got.ID)
	require.Len(t, got.Entries, 1)

	require.NoError(t, service.DeleteAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID)))
	_, err = service.GetAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID))
	require.ErrorIs(t, err, ErrNotFound)

	require.Len(t, publisher.events, 3)
	assert.Equal(t, "annotation.deleted", publisher.events[2].Name)
}

func TestServiceCreateUpdateDeleteAnnotationEntries(t *testing.T) {
	service, publisher, user, job := openAnnotationServiceTestDB(t)
	content := "root note"
	created, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Quote:           "quoted text",
		Anchor:          Anchor{StartMS: 100, EndMS: 200},
	})
	require.NoError(t, err)
	require.Len(t, created.Entries, 1)

	reply, parent, err := service.CreateAnnotationEntry(context.Background(), CreateEntryRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		AnnotationID:    PublicAnnotationID(created.ID),
		Content:         " reply text ",
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, parent.ID)
	assert.Equal(t, created.ID, reply.AnnotationID)
	assert.Equal(t, "reply text", reply.Content)
	assert.Equal(t, "annotation.entry.created", publisher.events[len(publisher.events)-1].Name)
	assert.Equal(t, PublicAnnotationEntryID(reply.ID), publisher.events[len(publisher.events)-1].EntryID)

	updated, _, err := service.UpdateAnnotationEntry(context.Background(), UpdateEntryRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		AnnotationID:    PublicAnnotationID(created.ID),
		EntryID:         PublicAnnotationEntryID(reply.ID),
		Content:         " updated reply ",
	})
	require.NoError(t, err)
	assert.Equal(t, "updated reply", updated.Content)
	assert.Equal(t, "annotation.entry.updated", publisher.events[len(publisher.events)-1].Name)

	got, err := service.GetAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID))
	require.NoError(t, err)
	require.Len(t, got.Entries, 2)
	assert.Equal(t, "root note", got.Entries[0].Content)
	assert.Equal(t, "updated reply", got.Entries[1].Content)

	require.NoError(t, service.DeleteAnnotationEntry(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID), PublicAnnotationEntryID(reply.ID)))
	assert.Equal(t, "annotation.entry.deleted", publisher.events[len(publisher.events)-1].Name)
	got, err = service.GetAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(created.ID))
	require.NoError(t, err)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, "root note", got.Entries[0].Content)
}

func TestServiceRejectsInvalidAnnotationEntryCommands(t *testing.T) {
	service, _, user, job := openAnnotationServiceTestDB(t)
	highlight, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "highlight",
		Anchor:          Anchor{StartMS: 10, EndMS: 20},
	})
	require.NoError(t, err)

	_, _, err = service.CreateAnnotationEntry(context.Background(), CreateEntryRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		AnnotationID:    PublicAnnotationID(highlight.ID),
		Content:         "not allowed",
	})
	require.ErrorIs(t, err, ErrValidation)

	content := "root note"
	note, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Quote:           "note",
		Anchor:          Anchor{StartMS: 30, EndMS: 40},
	})
	require.NoError(t, err)

	_, _, err = service.CreateAnnotationEntry(context.Background(), CreateEntryRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		AnnotationID:    PublicAnnotationID(note.ID),
		Content:         " ",
	})
	require.ErrorIs(t, err, ErrValidation)

	_, _, err = service.CreateAnnotationEntry(context.Background(), CreateEntryRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_missing",
		AnnotationID:    PublicAnnotationID(note.ID),
		Content:         "wrong transcript",
	})
	require.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, service.DeleteAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(note.ID)))
	_, _, err = service.CreateAnnotationEntry(context.Background(), CreateEntryRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		AnnotationID:    PublicAnnotationID(note.ID),
		Content:         "deleted parent",
	})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceCreateHighlightReturnsExistingDuplicate(t *testing.T) {
	service, publisher, user, job := openAnnotationServiceTestDB(t)
	startChar := 0
	endChar := 12
	hash := HashAnchorText("quoted text")
	req := CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "quoted text",
		Anchor: Anchor{
			StartMS:   1200,
			EndMS:     2400,
			StartChar: &startChar,
			EndChar:   &endChar,
			TextHash:  &hash,
		},
	}

	first, err := service.CreateAnnotation(context.Background(), req)
	require.NoError(t, err)
	second, err := service.CreateAnnotation(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	kind := models.AnnotationKindHighlight
	items, count, err := service.ListAnnotations(context.Background(), ListRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            &kind,
		Limit:           10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	require.Len(t, items, 1)
	assert.Equal(t, first.ID, items[0].ID)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, "annotation.created", publisher.events[0].Name)
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

func TestServiceSupportsFutureSharedTranscriptPolicyWithoutLeakingAnnotations(t *testing.T) {
	service, _, owner, job := openAnnotationServiceTestDB(t)
	otherUserID := owner.ID + 1
	service.SetTranscriptionAccessPolicy(mapAccessPolicy{
		transcriptions: map[string]*models.TranscriptionJob{job.ID: &job},
		allowed: map[uint]map[string]bool{
			owner.ID:    {job.ID: true},
			otherUserID: {job.ID: true},
		},
	})

	ownerContent := "owner note"
	ownerAnnotation, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          owner.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &ownerContent,
		Quote:           "owner quote",
		Anchor:          Anchor{StartMS: 100, EndMS: 200},
	})
	require.NoError(t, err)

	otherContent := "collaborator note"
	otherAnnotation, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          otherUserID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &otherContent,
		Quote:           "collaborator quote",
		Anchor:          Anchor{StartMS: 300, EndMS: 400},
	})
	require.NoError(t, err)
	require.Equal(t, otherUserID, otherAnnotation.UserID)
	require.Equal(t, job.ID, otherAnnotation.TranscriptionID)

	ownerItems, _, err := service.ListAnnotations(context.Background(), ListRequest{
		UserID:          owner.ID,
		TranscriptionID: "tr_" + job.ID,
		Limit:           10,
	})
	require.NoError(t, err)
	require.Len(t, ownerItems, 1)
	assert.Equal(t, ownerAnnotation.ID, ownerItems[0].ID)

	otherItems, _, err := service.ListAnnotations(context.Background(), ListRequest{
		UserID:          otherUserID,
		TranscriptionID: "tr_" + job.ID,
		Limit:           10,
	})
	require.NoError(t, err)
	require.Len(t, otherItems, 1)
	assert.Equal(t, otherAnnotation.ID, otherItems[0].ID)

	_, err = service.GetAnnotation(context.Background(), owner.ID, "tr_"+job.ID, PublicAnnotationID(otherAnnotation.ID))
	require.ErrorIs(t, err, ErrNotFound)
	_, err = service.GetAnnotation(context.Background(), otherUserID, "tr_"+job.ID, PublicAnnotationID(ownerAnnotation.ID))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceMarksHashedAnchorsStaleAgainstCurrentTranscript(t *testing.T) {
	service, publisher, user, job := openAnnotationServiceTestDB(t)
	transcript := orchestrator.CanonicalTranscript{
		Text: "hello world from transcript",
		Words: []orchestrator.CanonicalWord{
			{Word: "hello"},
			{Word: "world"},
			{Word: "from"},
			{Word: "transcript"},
		},
	}
	transcriptJSON, err := json.Marshal(transcript)
	require.NoError(t, err)
	require.NoError(t, service.jobs.UpdateTranscript(context.Background(), job.ID, string(transcriptJSON)))

	startWord := 0
	endWord := 1
	matchingHash := HashAnchorText("hello world")
	matching, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "hello world",
		Anchor: Anchor{
			StartMS:   0,
			EndMS:     1000,
			StartWord: &startWord,
			EndWord:   &endWord,
			TextHash:  &matchingHash,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, models.AnnotationStatusActive, matching.Status)

	staleHash := HashAnchorText("different text")
	stale, err := service.CreateAnnotation(context.Background(), CreateRequest{
		UserID:          user.ID,
		TranscriptionID: "tr_" + job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "hello world",
		Anchor: Anchor{
			StartMS:   0,
			EndMS:     1000,
			StartWord: &startWord,
			EndWord:   &endWord,
			TextHash:  &staleHash,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, models.AnnotationStatusStale, stale.Status)

	updatedTranscript := orchestrator.CanonicalTranscript{
		Text:  "different text from transcript",
		Words: []orchestrator.CanonicalWord{{Word: "different"}, {Word: "text"}},
	}
	updatedJSON, err := json.Marshal(updatedTranscript)
	require.NoError(t, err)
	updatedTranscriptValue := string(updatedJSON)
	job.Transcript = &updatedTranscriptValue
	require.NoError(t, service.jobs.UpdateTranscript(context.Background(), job.ID, updatedTranscriptValue))
	require.NoError(t, service.EnqueueForTranscription(context.Background(), &job))

	refreshed, err := service.GetAnnotation(context.Background(), user.ID, "tr_"+job.ID, PublicAnnotationID(matching.ID))
	require.NoError(t, err)
	assert.Equal(t, models.AnnotationStatusStale, refreshed.Status)

	require.GreaterOrEqual(t, len(publisher.events), 3)
	assert.Equal(t, "annotation.updated", publisher.events[len(publisher.events)-1].Name)
	assert.Equal(t, models.AnnotationStatusStale, publisher.events[len(publisher.events)-1].Status)
}
