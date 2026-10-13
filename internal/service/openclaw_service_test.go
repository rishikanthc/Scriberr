package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type commandCall struct {
	name  string
	args  []string
	stdin []byte
}

type fakeRunner struct {
	calls []commandCall
	errs  map[string]error
	outs  map[string][]byte
}

func (r *fakeRunner) Run(_ context.Context, name string, args []string, stdin []byte) ([]byte, error) {
	clonedArgs := append([]string(nil), args...)
	clonedStdin := append([]byte(nil), stdin...)
	r.calls = append(r.calls, commandCall{name: name, args: clonedArgs, stdin: clonedStdin})
	if err, ok := r.errs[name]; ok {
		return r.outs[name], err
	}
	return r.outs[name], nil
}

func TestOpenClawServiceSendSRTSuccess(t *testing.T) {
	runner := &fakeRunner{
		errs: map[string]error{},
		outs: map[string][]byte{
			"scp": []byte("uploaded"),
			"ssh": []byte("hook-ok"),
		},
	}
	svc := NewOpenClawServiceWithRunner(runner)

	profile := &models.OpenClawProfile{
		Name:     "prod",
		IP:       "user@example-host",
		SSHKey:   "test-private-key-content",
		HookKey:  "secret-hook-token",
		HookName: "Dashboard",
		Message:  "Summarize this meeting",
	}

	result, err := svc.SendSRT(context.Background(), profile, "1\n00:00:00,000 --> 00:00:01,000\nhello\n", "Weekly Sync", "job-123")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "prod", result.ProfileName)
	assert.Contains(t, result.RemotePath, "/tmp/")
	assert.Equal(t, "uploaded", result.SCPOutput)
	assert.Equal(t, "hook-ok", result.HookOutput)

	require.Len(t, runner.calls, 2)
	assert.Equal(t, "scp", runner.calls[0].name)
	assert.Equal(t, "ssh", runner.calls[1].name)

	sshStdin := string(runner.calls[1].stdin)
	assert.Contains(t, sshStdin, `"message":"Summarize this meeting`)
	assert.Contains(t, sshStdin, `"name":"Dashboard"`)
	assert.Contains(t, sshStdin, `"deliver":true`)
}

func TestOpenClawServiceSendSRTValidation(t *testing.T) {
	svc := NewOpenClawServiceWithRunner(&fakeRunner{errs: map[string]error{}, outs: map[string][]byte{}})

	_, err := svc.SendSRT(context.Background(), nil, "x", "title", "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile is required")

	profile := &models.OpenClawProfile{
		Name:     "p",
		IP:       "",
		SSHKey:   "key",
		HookKey:  "hk",
		HookName: "n",
		Message:  "m",
	}
	_, err = svc.SendSRT(context.Background(), profile, "x", "title", "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile ip")

	profile.IP = "example-host"
	profile.SSHKey = ""
	_, err = svc.SendSRT(context.Background(), profile, "x", "title", "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ssh key")

	profile.SSHKey = "key"
	profile.HookKey = ""
	_, err = svc.SendSRT(context.Background(), profile, "x", "title", "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook key")

	profile.HookKey = "hk"
	_, err = svc.SendSRT(context.Background(), profile, "", "title", "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "srt content")
}

func TestOpenClawServiceSendSRTCommandFailure(t *testing.T) {
	runner := &fakeRunner{
		errs: map[string]error{
			"scp": errors.New("scp-failed"),
		},
		outs: map[string][]byte{
			"scp": []byte("permission denied"),
		},
	}
	svc := NewOpenClawServiceWithRunner(runner)

	profile := &models.OpenClawProfile{
		Name:     "prod",
		IP:       "user@example-host",
		SSHKey:   "ssh-key",
		HookKey:  "hook-key",
		HookName: "Dashboard",
		Message:  "msg",
	}

	_, err := svc.SendSRT(context.Background(), profile, "valid srt", "title", "job-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scp upload failed")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestOpenClawServiceSendSRTHookErrorPayload(t *testing.T) {
	runner := &fakeRunner{
		errs: map[string]error{},
		outs: map[string][]byte{
			"scp": []byte("uploaded"),
			"ssh": []byte(`{"error":"invalid hook token"}`),
		},
	}
	svc := NewOpenClawServiceWithRunner(runner)

	profile := &models.OpenClawProfile{
		Name:     "prod",
		IP:       "user@example-host",
		SSHKey:   "ssh-key",
		HookKey:  "bad-token",
		HookName: "Dashboard",
		Message:  "msg",
	}

	_, err := svc.SendSRT(context.Background(), profile, "valid srt", "title", "job-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote hook returned error")
	assert.Contains(t, err.Error(), "invalid hook token")
}

func TestParseOpenClawHookError(t *testing.T) {
	assert.Equal(t, "bad request", parseOpenClawHookError([]byte(`{"error":"bad request"}`)))
	assert.Equal(t, "", parseOpenClawHookError([]byte(`{"ok":true}`)))
	assert.Equal(t, "", parseOpenClawHookError([]byte(`not-json`)))
}

func TestBuildRemoteBaseName(t *testing.T) {
	name := buildRemoteBaseName("Team / Weekly Sync", "abcdef123456789")
	assert.True(t, strings.HasPrefix(name, "team_weekly_sync_abcdef123456_"))
}
