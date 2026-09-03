package repository

import (
	"context"
	"testing"

	"scriberr/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestJobRepository(t *testing.T) (JobRepository, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.TranscriptionJob{}))

	return NewJobRepository(db), db
}

// ListWithParams used to build the ORDER BY clause by concatenating the
// caller-supplied sortBy and sortOrder values straight into the query.
// A sortBy value that isn't one of the allowed column names must not reach
// the query at all.
func TestListWithParams_RejectsUnknownSortColumn(t *testing.T) {
	repo, _ := newTestJobRepository(t)
	ctx := context.Background()

	// A payload that would be a syntax error (or worse, if it parsed) if it
	// ever reached the query as a raw ORDER BY fragment.
	maliciousSortBy := "id; DROP TABLE transcription_jobs; --"

	_, _, err := repo.ListWithParams(ctx, 0, 10, maliciousSortBy, "desc", "", nil)
	require.NoError(t, err, "an unrecognized sortBy should fall back to the default sort, not error or reach the query")
}

func TestListWithParams_AllowsKnownSortColumn(t *testing.T) {
	repo, db := newTestJobRepository(t)
	ctx := context.Background()

	title1 := "b-job"
	title2 := "a-job"
	require.NoError(t, db.Create(&models.TranscriptionJob{ID: "1", Title: &title1, AudioPath: "a.mp3"}).Error)
	require.NoError(t, db.Create(&models.TranscriptionJob{ID: "2", Title: &title2, AudioPath: "b.mp3"}).Error)

	jobs, count, err := repo.ListWithParams(ctx, 0, 10, "title", "asc", "", nil)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Len(t, jobs, 2)
	require.Equal(t, "a-job", *jobs[0].Title)
	require.Equal(t, "b-job", *jobs[1].Title)
}
