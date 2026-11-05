package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

func TestParakeetPyannoteUsesHFTokenFromEnv(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model_family":"nvidia_parakeet","diarize":true,"diarize_model":"pyannote"}`

	start := func() int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		(&Handler{}).getValidatedTranscriptionParams(c, &models.TranscriptionJob{}, "job")
		return w.Code
	}

	t.Setenv("HF_TOKEN", "")
	if code := start(); code != http.StatusBadRequest {
		t.Fatalf("without a token: code %d, want 400", code)
	}

	t.Setenv("HF_TOKEN", "hf_x")
	if code := start(); code == http.StatusBadRequest {
		t.Fatal("with HF_TOKEN set: got 400")
	}
}
