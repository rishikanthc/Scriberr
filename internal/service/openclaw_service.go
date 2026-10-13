package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"scriberr/internal/models"
)

const (
	openClawRemoteHookURL = "http://127.0.0.1:18789/hooks/meeting"
	openClawRemoteDir     = "/tmp"
)

// OpenClawCommandRunner executes shell commands.
type OpenClawCommandRunner interface {
	Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, error)
}

type osOpenClawCommandRunner struct{}

func (r *osOpenClawCommandRunner) Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	return cmd.CombinedOutput()
}

// OpenClawService handles secure delivery to remote OpenClaw hosts.
type OpenClawService struct {
	runner OpenClawCommandRunner
}

// NewOpenClawService creates a service with OS-backed command execution.
func NewOpenClawService() *OpenClawService {
	return &OpenClawService{runner: &osOpenClawCommandRunner{}}
}

// NewOpenClawServiceWithRunner creates a service with a custom command runner for tests.
func NewOpenClawServiceWithRunner(runner OpenClawCommandRunner) *OpenClawService {
	return &OpenClawService{runner: runner}
}

type openClawHookPayload struct {
	Message string `json:"message"`
	Name    string `json:"name"`
	Deliver bool   `json:"deliver"`
}

// OpenClawSendResult captures remote delivery results.
type OpenClawSendResult struct {
	RemotePath  string `json:"remote_path"`
	SCPOutput   string `json:"scp_output,omitempty"`
	HookOutput  string `json:"hook_output,omitempty"`
	ProfileName string `json:"profile_name"`
}

// SendSRT uploads an SRT file to a remote host via SCP, then triggers the OpenClaw hook over SSH.
func (s *OpenClawService) SendSRT(ctx context.Context, profile *models.OpenClawProfile, srtContent string, title string, transcriptionID string) (*OpenClawSendResult, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is required")
	}
	if strings.TrimSpace(profile.IP) == "" {
		return nil, fmt.Errorf("profile ip is required")
	}
	if strings.TrimSpace(profile.SSHKey) == "" {
		return nil, fmt.Errorf("profile ssh key is required")
	}
	if strings.TrimSpace(profile.HookKey) == "" {
		return nil, fmt.Errorf("profile hook key is required")
	}
	if strings.TrimSpace(srtContent) == "" {
		return nil, fmt.Errorf("srt content is empty")
	}

	keyPath, cleanupKey, err := writeTempPrivateKey(profile.SSHKey)
	if err != nil {
		return nil, err
	}
	defer cleanupKey()

	fileBase := buildRemoteBaseName(title, transcriptionID)
	localPath, cleanupSRT, err := writeTempSRT(srtContent)
	if err != nil {
		return nil, err
	}
	defer cleanupSRT()

	remotePath := filepath.ToSlash(filepath.Join(openClawRemoteDir, fileBase+".srt"))

	scpArgs := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=3",
		localPath,
		fmt.Sprintf("%s:%s", profile.IP, remotePath),
	}
	scpOutput, err := s.runner.Run(ctx, "scp", scpArgs, nil)
	if err != nil {
		return nil, fmt.Errorf("scp upload failed: %w: %s", err, strings.TrimSpace(string(scpOutput)))
	}

	message := strings.TrimSpace(profile.Message)
	if message == "" {
		message = "Please summarize this meeting transcript."
	}
	message = fmt.Sprintf("%s\n\nTitle: %s\nTranscription ID: %s\nSRT Path: %s",
		message, titleOrFallback(title), transcriptionID, remotePath)

	payload := openClawHookPayload{
		Message: message,
		Name:    strings.TrimSpace(profile.HookName),
		Deliver: true,
	}
	if payload.Name == "" {
		payload.Name = "Scriberr"
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hook payload: %w", err)
	}

	remoteCmd := fmt.Sprintf(
		"curl -fsS -X POST %s -H %s -H %s --data-binary @-",
		shellQuote(openClawRemoteHookURL),
		shellQuote("Authorization: Bearer "+profile.HookKey),
		shellQuote("Content-Type: application/json"),
	)

	sshArgs := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=3",
		profile.IP,
		remoteCmd,
	}
	hookOutput, err := s.runner.Run(ctx, "ssh", sshArgs, payloadJSON)
	if err != nil {
		return nil, fmt.Errorf("remote hook trigger failed: %w: %s", err, strings.TrimSpace(string(hookOutput)))
	}
	if hookErr := parseOpenClawHookError(hookOutput); hookErr != "" {
		return nil, fmt.Errorf("remote hook returned error: %s", hookErr)
	}

	return &OpenClawSendResult{
		RemotePath:  remotePath,
		SCPOutput:   strings.TrimSpace(string(scpOutput)),
		HookOutput:  strings.TrimSpace(string(hookOutput)),
		ProfileName: profile.Name,
	}, nil
}

func writeTempPrivateKey(keyContent string) (string, func(), error) {
	file, err := os.CreateTemp("", "scriberr-openclaw-key-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp ssh key: %w", err)
	}
	cleanup := func() { _ = os.Remove(file.Name()) }

	cleaned := strings.TrimSpace(keyContent) + "\n"
	if _, err := file.WriteString(cleaned); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to write temp ssh key: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to close temp ssh key: %w", err)
	}
	if err := os.Chmod(file.Name(), 0600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to set ssh key permissions: %w", err)
	}
	return file.Name(), cleanup, nil
}

func writeTempSRT(content string) (string, func(), error) {
	file, err := os.CreateTemp("", "scriberr-openclaw-*.srt")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp srt: %w", err)
	}
	cleanup := func() { _ = os.Remove(file.Name()) }

	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to write temp srt: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to close temp srt: %w", err)
	}

	return file.Name(), cleanup, nil
}

var invalidFileChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
var multiUnderscore = regexp.MustCompile(`_+`)

func buildRemoteBaseName(title, transcriptionID string) string {
	base := titleOrFallback(title)
	base = strings.ToLower(strings.TrimSpace(base))
	base = strings.ReplaceAll(base, " ", "_")
	base = invalidFileChars.ReplaceAllString(base, "_")
	base = multiUnderscore.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._-")
	if base == "" {
		base = "meeting"
	}
	if len(base) > 48 {
		base = base[:48]
	}

	shortID := strings.TrimSpace(transcriptionID)
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	shortID = invalidFileChars.ReplaceAllString(shortID, "_")

	tstamp := time.Now().UTC().Format("20060102_150405")
	if shortID == "" {
		return fmt.Sprintf("%s_%s", base, tstamp)
	}
	return fmt.Sprintf("%s_%s_%s", base, shortID, tstamp)
}

func titleOrFallback(title string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "Untitled Recording"
	}
	return trimmed
}

func shellQuote(input string) string {
	return "'" + strings.ReplaceAll(input, "'", `'"'"'`) + "'"
}

func parseOpenClawHookError(output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return ""
	}

	rawErr, ok := payload["error"]
	if !ok {
		return ""
	}

	errMsg, ok := rawErr.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(errMsg)
}
