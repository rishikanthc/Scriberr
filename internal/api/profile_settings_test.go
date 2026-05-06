package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func pipelineRequest(kindModel ...string) []map[string]any {
	steps := make([]map[string]any, 0, len(kindModel)/2)
	for i := 0; i+1 < len(kindModel); i += 2 {
		steps = append(steps, map[string]any{
			"kind":  kindModel[i],
			"model": kindModel[i+1],
		})
	}
	return steps
}

func TestProfileCRUDAndDefaultSelection(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name":        "Fast local",
		"description": "Fast local transcription",
		"is_default":  true,
		"options": map[string]any{
			"pipeline": []map[string]any{{
				"kind":  "transcription",
				"model": "whisper-base",
				"options": map[string]any{
					"sherpa.whisper.language": "en",
					"chunking.mode":           "vad",
					"runtime.num_threads":     2,
				},
			}},
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	firstID := body["id"].(string)
	require.True(t, strings.HasPrefix(firstID, "profile_"))
	require.Equal(t, true, body["is_default"])
	require.Equal(t, "Fast local", body["name"])
	options := body["options"].(map[string]any)
	pipeline := options["pipeline"].([]any)
	require.Len(t, pipeline, 1)
	step := pipeline[0].(map[string]any)
	require.Equal(t, "transcription", step["kind"])
	require.Equal(t, "whisper-base", step["model"])
	require.Equal(t, "whisper", step["model_family"])
	stepOptions := step["options"].(map[string]any)
	require.Equal(t, "en", stepOptions["sherpa.whisper.language"])
	require.Equal(t, "vad", stepOptions["chunking.mode"])
	require.Equal(t, float64(2), stepOptions["runtime.num_threads"])
	require.NotContains(t, options, "decoding_method")
	require.NotContains(t, options, "chunking_strategy")

	var storedProfile models.TranscriptionProfile
	require.NoError(t, database.DB.First(&storedProfile, "id = ?", strings.TrimPrefix(firstID, "profile_")).Error)
	require.Empty(t, storedProfile.Parameters.Model)
	require.Empty(t, storedProfile.Parameters.ChunkingStrategy)
	require.Len(t, storedProfile.Parameters.Pipeline, 1)
	require.Equal(t, models.ASRStepTranscription, storedProfile.Parameters.Pipeline[0].Kind)
	require.Equal(t, "vad", storedProfile.Parameters.Pipeline[0].Options["chunking.mode"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/settings", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, firstID, body["default_profile_id"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name":       "Accurate",
		"is_default": true,
		"options": map[string]any{
			"pipeline": pipelineRequest("transcription", "whisper-small", "diarization", "diarization-default"),
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	secondID := body["id"].(string)
	require.Equal(t, true, body["is_default"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/profiles", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 2)
	defaultCount := 0
	for _, raw := range items {
		item := raw.(map[string]any)
		if item["is_default"].(bool) {
			defaultCount++
			require.Equal(t, secondID, item["id"])
		}
	}
	require.Equal(t, 1, defaultCount)

	resp, body = s.request(t, http.MethodPost, "/api/v1/profiles/"+firstID+":set-default", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, firstID, body["id"])
	require.Equal(t, true, body["is_default"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/profiles/"+firstID, map[string]any{
		"name":        "Fast local renamed",
		"description": "Updated",
		"options": map[string]any{
			"pipeline": pipelineRequest("transcription", "parakeet-v2", "diarization", "diarization-default"),
		},
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "Fast local renamed", body["name"])
	updatedPipeline := body["options"].(map[string]any)["pipeline"].([]any)
	require.Equal(t, "parakeet-v2", updatedPipeline[0].(map[string]any)["model"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/profiles/"+firstID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["is_default"])

	resp, _ = s.request(t, http.MethodDelete, "/api/v1/profiles/"+secondID, nil, token, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/profiles/"+secondID, nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestProfileValidationAndAuth(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, _ := s.request(t, http.MethodGet, "/api/v1/profiles", nil, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, body := s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name": "Invalid",
		"options": map[string]any{
			"pipeline": pipelineRequest("transcription", "whisper-base"),
			"language": "english",
		},
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "options.language", errBody["field"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name": "Invalid model",
		"options": map[string]any{
			"pipeline": pipelineRequest("transcription", "large-v3"),
		},
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody = body["error"].(map[string]any)
	require.Equal(t, "options.pipeline", errBody["field"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name": "Invalid chunking",
		"options": map[string]any{
			"pipeline":          pipelineRequest("transcription", "whisper-base"),
			"chunking_strategy": "dynamic",
		},
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody = body["error"].(map[string]any)
	require.Equal(t, "options.chunking_strategy", errBody["field"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name": "Invalid threshold",
		"options": map[string]any{
			"pipeline":              pipelineRequest("transcription", "whisper-base"),
			"diarization_threshold": 1.5,
		},
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody = body["error"].(map[string]any)
	require.Equal(t, "options.diarization_threshold", errBody["field"])

	resp, _ = s.request(t, http.MethodGet, "/api/v1/profiles/profile_missing", nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestProfileRejectsLegacyDecodingOption(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name": "Whisper",
		"options": map[string]any{
			"pipeline":        pipelineRequest("transcription", "whisper-base"),
			"decoding_method": "modified_beam_search",
		},
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "options.decoding_method", errBody["field"])
}

func TestGetProfileDoesNotPublishUpdateEvent(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name": "Read only profile",
		"options": map[string]any{
			"pipeline": pipelineRequest("transcription", "whisper-base"),
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	profileID := body["id"].(string)

	sub, unsubscribe := s.handler.events.subscribe(currentTestUserID(t, "admin"), "")
	defer unsubscribe()

	resp, _ = s.request(t, http.MethodGet, "/api/v1/profiles/"+profileID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)

	select {
	case event := <-sub.ch:
		t.Fatalf("GET profile unexpectedly published %s", event.Name)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestSettingsPartialUpdateAndValidation(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodGet, "/api/v1/settings", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["auto_transcription_enabled"])
	require.Equal(t, true, body["local_only"])
	require.Greater(t, body["max_upload_size_mb"], float64(0))

	resp, body = s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name":       "Default",
		"is_default": true,
		"options": map[string]any{
			"pipeline": pipelineRequest("transcription", "whisper-base"),
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	profileID := body["id"].(string)

	resp, body = s.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"auto_transcription_enabled": true,
		"default_profile_id":         profileID,
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["auto_transcription_enabled"])
	require.Equal(t, profileID, body["default_profile_id"])
	require.Equal(t, true, body["local_only"])

	var settings models.UserSettings
	require.NoError(t, database.DB.First(&settings).Error)
	require.NotNil(t, settings.DefaultProfileID)
	require.Equal(t, strings.TrimPrefix(profileID, "profile_"), *settings.DefaultProfileID)
	var user models.User
	require.NoError(t, database.DB.First(&user).Error)
	require.NotContains(t, user.SettingsJSON, strings.TrimPrefix(profileID, "profile_"))

	resp, body = s.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"default_profile_id": "profile_missing",
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "default_profile_id", errBody["field"])
}

func TestSettingsAutomationEnablementRequiresDependencies(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodGet, "/api/v1/settings", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["auto_transcription_enabled"])
	require.Equal(t, true, body["auto_rename_enabled"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"auto_transcription_enabled": false,
		"auto_rename_enabled":        false,
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, false, body["auto_transcription_enabled"])
	require.Equal(t, false, body["auto_rename_enabled"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"auto_transcription_enabled": true,
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "default_profile_id", errBody["field"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"auto_rename_enabled": true,
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody = body["error"].(map[string]any)
	require.Equal(t, "auto_rename_enabled", errBody["field"])

	baseURL := "http://localhost:11434/v1"
	smallModel := "small-model"
	require.NoError(t, database.DB.Create(&models.LLMConfig{
		UserID:        1,
		Name:          "Default LLM",
		Provider:      "openai_compatible",
		BaseURL:       &baseURL,
		OpenAIBaseURL: &baseURL,
		SmallModel:    &smallModel,
		IsDefault:     true,
	}).Error)

	resp, body = s.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"auto_rename_enabled": true,
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["auto_rename_enabled"])
}

func TestCapabilitiesQueueAndEvents(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Queued",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)

	resp, body = s.request(t, http.MethodGet, "/api/v1/models/transcription", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.NotEmpty(t, items)
	model := items[0].(map[string]any)
	require.Equal(t, "local", model["provider"])
	require.Contains(t, model["capabilities"].([]any), "transcription")

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/queue", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, float64(1), body["queued"])
	require.Equal(t, float64(0), body["processing"])
	require.Equal(t, float64(0), body["failed"])

	resp, rawLogs := s.rawRequest(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/logs", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotContains(t, rawLogs, "/")
}
