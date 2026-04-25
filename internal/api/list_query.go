package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultListLimit = 50
	maxListLimit     = 100
)

type listQuery struct {
	Limit        int
	Sort         string
	SortColumn   string
	SortDesc     bool
	Cursor       *listCursor
	Query        string
	UpdatedAfter *time.Time
}

type listCursor struct {
	Sort  string `json:"sort"`
	Value string `json:"value"`
	ID    string `json:"id"`
}

func parseListQuery(c *gin.Context, allowedSorts map[string]string) (*listQuery, bool) {
	limit := defaultListLimit
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > maxListLimit {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit is invalid", stringPtr("limit"))
			return nil, false
		}
		limit = parsed
	}

	sortValue := strings.TrimSpace(c.DefaultQuery("sort", "-created_at"))
	sortColumn, sortDesc, ok := parseSort(sortValue, allowedSorts)
	if !ok {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "sort is invalid", stringPtr("sort"))
		return nil, false
	}

	var cursor *listCursor
	if rawCursor := strings.TrimSpace(c.Query("cursor")); rawCursor != "" {
		decoded, err := decodeListCursor(rawCursor)
		if err != nil || decoded.Sort != sortValue || decoded.ID == "" {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "cursor is invalid", stringPtr("cursor"))
			return nil, false
		}
		cursor = decoded
	}

	var updatedAfter *time.Time
	if rawUpdatedAfter := strings.TrimSpace(c.Query("updated_after")); rawUpdatedAfter != "" {
		parsed, err := time.Parse(time.RFC3339, rawUpdatedAfter)
		if err != nil {
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "updated_after is invalid", stringPtr("updated_after"))
			return nil, false
		}
		updatedAfter = &parsed
	}

	return &listQuery{
		Limit:        limit,
		Sort:         sortValue,
		SortColumn:   sortColumn,
		SortDesc:     sortDesc,
		Cursor:       cursor,
		Query:        strings.TrimSpace(c.Query("q")),
		UpdatedAfter: updatedAfter,
	}, true
}

func parseSort(raw string, allowed map[string]string) (string, bool, bool) {
	desc := strings.HasPrefix(raw, "-")
	key := strings.TrimPrefix(raw, "-")
	column, ok := allowed[key]
	return column, desc, ok
}

func applyListQuery(query *gorm.DB, opts *listQuery) *gorm.DB {
	if opts.Query != "" {
		query = query.Where("LOWER(COALESCE(title, '')) LIKE ?", "%"+strings.ToLower(opts.Query)+"%")
	}
	if opts.UpdatedAfter != nil {
		query = query.Where("updated_at > ?", *opts.UpdatedAfter)
	}
	if opts.Cursor != nil {
		query = applyCursor(query, opts)
	}
	direction := "ASC"
	comparisonDirection := "asc"
	if opts.SortDesc {
		direction = "DESC"
		comparisonDirection = "desc"
	}
	return query.Order(opts.SortColumn + " " + direction).Order("id " + comparisonDirection).Limit(opts.Limit + 1)
}

func applyCursor(query *gorm.DB, opts *listQuery) *gorm.DB {
	operator := ">"
	if opts.SortDesc {
		operator = "<"
	}
	switch opts.SortColumn {
	case "created_at", "updated_at":
		value, err := time.Parse(time.RFC3339Nano, opts.Cursor.Value)
		if err != nil {
			return query.Where("1 = 0")
		}
		return query.Where("("+opts.SortColumn+" "+operator+" ?) OR ("+opts.SortColumn+" = ? AND id "+operator+" ?)", value, value, opts.Cursor.ID)
	default:
		return query.Where("("+opts.SortColumn+" "+operator+" ?) OR ("+opts.SortColumn+" = ? AND id "+operator+" ?)", opts.Cursor.Value, opts.Cursor.Value, opts.Cursor.ID)
	}
}

func trimListPage(jobs []models.TranscriptionJob, opts *listQuery) ([]models.TranscriptionJob, any) {
	if len(jobs) <= opts.Limit {
		return jobs, nil
	}
	page := jobs[:opts.Limit]
	return page, encodeListCursor(listCursor{
		Sort:  opts.Sort,
		Value: cursorValue(page[len(page)-1], opts.SortColumn),
		ID:    page[len(page)-1].ID,
	})
}

func cursorValue(job models.TranscriptionJob, column string) string {
	switch column {
	case "created_at":
		return job.CreatedAt.Format(time.RFC3339Nano)
	case "updated_at":
		return job.UpdatedAt.Format(time.RFC3339Nano)
	case "title":
		if job.Title != nil {
			return *job.Title
		}
		return ""
	default:
		return ""
	}
}

func encodeListCursor(cursor listCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeListCursor(raw string) (*listCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var cursor listCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}

func allowedResourceSorts() map[string]string {
	return map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"title":      "title",
	}
}
