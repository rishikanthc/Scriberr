package tests

import (
	"encoding/json"
	"net/http"

	"scriberr/internal/api"
	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
)

func (suite *APIHandlerTestSuite) TestSummarize() {
	// Setup
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Summary Test Transcription")
	job.Status = models.StatusCompleted
	transcript := "This is a transcript about lots of things."
	job.Transcript = &transcript
	suite.helper.DB.Save(job)

	req := api.SummarizeRequest{
		Model:           "gpt-3.5-turbo",
		Content:         "This is the content to summarize",
		TranscriptionID: job.ID,
	}

	resp := suite.makeAuthenticatedRequest("POST", "/api/v1/summarize/", req, true)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Check headers for streaming
	assert.Equal(suite.T(), "text/plain; charset=utf-8", resp.Header().Get("Content-Type"))
	assert.Equal(suite.T(), "chunked", resp.Header().Get("Transfer-Encoding"))

	// Check response body
	body := resp.Body.String()
	// Mock server returns "This is a test response..." in chunks
	assert.Contains(suite.T(), body, "This")

	// Verify summary saved
	var summary models.Summary
	err := suite.helper.DB.Where("transcription_id = ?", job.ID).First(&summary).Error
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), summary.Content)
}

func (suite *APIHandlerTestSuite) TestGetSummaryForTranscription() {
	// Setup
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Summary Retrieval Test")

	// Case 1: No summary
	resp := suite.makeAuthenticatedRequest("GET", "/api/v1/transcription/"+job.ID+"/summary", nil, true)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)
	var summaryResp models.Summary
	json.Unmarshal(resp.Body.Bytes(), &summaryResp)
	assert.Empty(suite.T(), summaryResp.Content)

	// Case 2: Saved summary
	summary := models.Summary{
		TranscriptionID: job.ID,
		Model:           "gpt-4",
		Content:         "Stored summary content",
	}
	suite.helper.DB.Create(&summary)

	resp = suite.makeAuthenticatedRequest("GET", "/api/v1/transcription/"+job.ID+"/summary", nil, true)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	json.Unmarshal(resp.Body.Bytes(), &summaryResp)
	assert.Equal(suite.T(), "Stored summary content", summaryResp.Content)
	assert.Equal(suite.T(), "gpt-4", summaryResp.Model)
}
