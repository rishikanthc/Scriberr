package api

import (
	"net/http"
	"testing"

	"scriberr/internal/auth"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/transcription/scheduler"

	"github.com/stretchr/testify/require"
)

func TestAdminSchedulerSettingsLifecycle(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodGet, "/api/v1/admin/queue/scheduler", nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "priority", body["policy"])

	resp, body = s.request(t, http.MethodPut, "/api/v1/admin/queue/scheduler", map[string]any{
		"policy": "fifo",
	}, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "fifo", body["policy"])

	var setting models.SystemSetting
	require.NoError(t, database.DB.First(&setting, "key = ?", scheduler.SettingKey).Error)
	require.JSONEq(t, `{"policy":"fifo"}`, setting.ValueJSON)

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/queue/scheduler", nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "fifo", body["policy"])

	resp, body = s.request(t, http.MethodPut, "/api/v1/admin/queue/scheduler", map[string]any{
		"policy": "random",
	}, adminToken, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errBody["code"])
	require.Equal(t, "policy", errBody["field"])

	require.NoError(t, database.DB.First(&setting, "key = ?", scheduler.SettingKey).Error)
	require.JSONEq(t, `{"policy":"fifo"}`, setting.ValueJSON)
}

func TestAdminSchedulerSettingsRejectNonAdminAndAPIKey(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)
	nonAdmin := models.User{Username: "scheduler-member", Password: "pw", Role: "user"}
	require.NoError(t, database.DB.Create(&nonAdmin).Error)
	nonAdminToken, err := auth.NewAuthService("test-secret").GenerateToken(&nonAdmin)
	require.NoError(t, err)

	resp, body := s.request(t, http.MethodPost, "/api/v1/api-keys", map[string]any{"name": "admin-cli"}, adminToken, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	apiKey := body["key"].(string)

	cases := []struct {
		name   string
		token  string
		apiKey string
		want   int
		code   string
	}{
		{name: "anonymous", want: http.StatusUnauthorized, code: "UNAUTHORIZED"},
		{name: "non-admin jwt", token: nonAdminToken, want: http.StatusForbidden, code: "FORBIDDEN"},
		{name: "api key", apiKey: apiKey, want: http.StatusForbidden, code: "FORBIDDEN"},
		{name: "admin jwt", token: adminToken, want: http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := s.request(t, http.MethodGet, "/api/v1/admin/queue/scheduler", nil, tc.token, tc.apiKey)
			require.Equal(t, tc.want, resp.Code)
			if tc.code != "" {
				errBody := body["error"].(map[string]any)
				require.Equal(t, tc.code, errBody["code"])
			}
		})
	}
}
