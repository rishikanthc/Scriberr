package api

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"scriberr/internal/auth"
	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func TestAdminUserManagementLifecycle(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"username":     "member",
		"password":     "member-password",
		"role":         "user",
		"email":        "member@example.com",
		"display_name": "Member User",
	}, adminToken, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	require.Equal(t, "member", body["username"])
	require.Equal(t, "user", body["role"])
	require.Equal(t, models.UserStatusActive, body["status"])
	userID := body["id"].(string)

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/users", nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.GreaterOrEqual(t, len(body["items"].([]any)), 2)

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/users/"+userID, nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "member", body["username"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "member",
		"password": "member-password",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	memberToken := body["access_token"].(string)
	memberRefresh := body["refresh_token"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/api-keys", map[string]any{
		"name": "member-cli",
	}, memberToken, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	memberAPIKey := body["key"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/admin/users/"+userID+":disable", nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, models.UserStatusDisabled, body["status"])

	resp, _ = s.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "member",
		"password": "member-password",
	}, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, _ = s.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": memberRefresh,
	}, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/files", nil, "", memberAPIKey)
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, body = s.request(t, http.MethodPost, "/api/v1/admin/users/"+userID+":enable", nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, models.UserStatusActive, body["status"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/admin/users/"+userID+":reset-password", map[string]any{
		"password": "reset-password",
	}, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["ok"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "member",
		"password": "reset-password",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotEmpty(t, body["access_token"])
}

func TestAdminUserManagementRejectsNonAdminAndAPIKey(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)
	nonAdmin := models.User{Username: "member", Password: "pw", Role: "user"}
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
			resp, body := s.request(t, http.MethodGet, "/api/v1/admin/users", nil, tc.token, tc.apiKey)
			require.Equal(t, tc.want, resp.Code)
			if tc.code != "" {
				errBody := body["error"].(map[string]any)
				require.Equal(t, tc.code, errBody["code"])
			}
		})
	}
}

func TestAdminCannotDisableOrDemoteLastActiveAdmin(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)
	adminID := "user_" + strconvUserID(t, "admin")

	resp, body := s.request(t, http.MethodPost, "/api/v1/admin/users/"+adminID+":disable", nil, adminToken, "")
	require.Equal(t, http.StatusConflict, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "CONFLICT", errBody["code"])
	require.True(t, strings.Contains(errBody["message"].(string), "last active admin"))

	resp, body = s.request(t, http.MethodPatch, "/api/v1/admin/users/"+adminID, map[string]any{
		"role": "user",
	}, adminToken, "")
	require.Equal(t, http.StatusConflict, resp.Code)
	errBody = body["error"].(map[string]any)
	require.Equal(t, "CONFLICT", errBody["code"])
}

func strconvUserID(t *testing.T, username string) string {
	t.Helper()
	return strconv.FormatUint(uint64(currentTestUserID(t, username)), 10)
}
