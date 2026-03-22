package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OpenClawProfileRequest defines create/update payloads for OpenClaw profiles.
type OpenClawProfileRequest struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	SSHKey   string `json:"ssh_key"`
	HookKey  string `json:"hook_key"`
	HookName string `json:"hook_name"`
	Message  string `json:"message"`
}

type openClawProfileResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	IP         string    `json:"ip"`
	HookName   string    `json:"hook_name"`
	Message    string    `json:"message"`
	HasSSHKey  bool      `json:"has_ssh_key"`
	HasHookKey bool      `json:"has_hook_key"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SendToOpenClawRequest defines the payload for sending a transcription.
type SendToOpenClawRequest struct {
	ProfileID string `json:"profile_id" binding:"required"`
}

// ListOpenClawProfiles lists all saved OpenClaw profiles.
func (h *Handler) ListOpenClawProfiles(c *gin.Context) {
	profiles, _, err := h.openClawProfileRepo.List(c.Request.Context(), 0, 1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OpenClaw profiles"})
		return
	}

	response := make([]openClawProfileResponse, 0, len(profiles))
	for _, profile := range profiles {
		response = append(response, toOpenClawProfileResponse(profile))
	}
	c.JSON(http.StatusOK, response)
}

// CreateOpenClawProfile creates a new OpenClaw profile.
func (h *Handler) CreateOpenClawProfile(c *gin.Context) {
	req, ok := bindOpenClawProfileRequest(c)
	if !ok {
		return
	}

	profile := models.OpenClawProfile{
		Name:     strings.TrimSpace(req.Name),
		IP:       strings.TrimSpace(req.IP),
		SSHKey:   strings.TrimSpace(req.SSHKey),
		HookKey:  strings.TrimSpace(req.HookKey),
		HookName: strings.TrimSpace(req.HookName),
		Message:  strings.TrimSpace(req.Message),
	}
	if err := validateOpenClawProfile(profile, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.openClawProfileRepo.Create(c.Request.Context(), &profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OpenClaw profile"})
		return
	}

	c.JSON(http.StatusOK, toOpenClawProfileResponse(profile))
}

// GetOpenClawProfile fetches one OpenClaw profile by ID.
func (h *Handler) GetOpenClawProfile(c *gin.Context) {
	profile, err := h.openClawProfileRepo.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "OpenClaw profile not found"})
		return
	}
	c.JSON(http.StatusOK, toOpenClawProfileResponse(*profile))
}

// UpdateOpenClawProfile updates an existing OpenClaw profile.
func (h *Handler) UpdateOpenClawProfile(c *gin.Context) {
	profile, err := h.openClawProfileRepo.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "OpenClaw profile not found"})
		return
	}

	req, ok := bindOpenClawProfileRequest(c)
	if !ok {
		return
	}

	profile.Name = strings.TrimSpace(req.Name)
	profile.IP = strings.TrimSpace(req.IP)
	profile.HookName = strings.TrimSpace(req.HookName)
	profile.Message = strings.TrimSpace(req.Message)
	if sshKey := strings.TrimSpace(req.SSHKey); sshKey != "" {
		profile.SSHKey = sshKey
	}
	if hookKey := strings.TrimSpace(req.HookKey); hookKey != "" {
		profile.HookKey = hookKey
	}

	if err := validateOpenClawProfile(*profile, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.openClawProfileRepo.Update(c.Request.Context(), profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update OpenClaw profile"})
		return
	}

	c.JSON(http.StatusOK, toOpenClawProfileResponse(*profile))
}

// DeleteOpenClawProfile deletes a saved OpenClaw profile.
func (h *Handler) DeleteOpenClawProfile(c *gin.Context) {
	id := c.Param("id")
	if _, err := h.openClawProfileRepo.FindByID(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "OpenClaw profile not found"})
		return
	}

	if err := h.openClawProfileRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete OpenClaw profile"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "OpenClaw profile deleted"})
}

// SendTranscriptionToOpenClaw uploads SRT then triggers the OpenClaw hook.
func (h *Handler) SendTranscriptionToOpenClaw(c *gin.Context) {
	if h.openClawService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OpenClaw service is not initialized"})
		return
	}
	if authType, exists := c.Get("auth_type"); !exists || authType != "jwt" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT authentication required"})
		return
	}

	var req SendToOpenClawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	job, err := h.jobRepo.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transcription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load transcription"})
		return
	}
	if job.Status != models.StatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription is not completed yet"})
		return
	}
	if job.Transcript == nil || strings.TrimSpace(*job.Transcript) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcript is empty"})
		return
	}

	profile, err := h.openClawProfileRepo.FindByID(c.Request.Context(), req.ProfileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "OpenClaw profile not found"})
		return
	}

	speakerMap := map[string]string{}
	mappings, err := h.speakerMappingRepo.ListByJob(c.Request.Context(), job.ID)
	if err == nil {
		for _, mapping := range mappings {
			speakerMap[mapping.OriginalSpeaker] = mapping.CustomName
		}
	}

	srt, err := buildSRTFromRawTranscript(*job.Transcript, speakerMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build SRT: " + err.Error()})
		return
	}

	title := "Untitled Recording"
	if job.Title != nil && strings.TrimSpace(*job.Title) != "" {
		title = strings.TrimSpace(*job.Title)
	}

	result, err := h.openClawService.SendSRT(c.Request.Context(), profile, srt, title, job.ID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to send to OpenClaw: " + err.Error()})
		return
	}

	now := time.Now()
	job.OpenClawSentAt = &now
	job.OpenClawProfileName = &profile.Name
	if err := h.jobRepo.Update(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist OpenClaw send status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Sent to OpenClaw",
		"profile_name": result.ProfileName,
		"remote_path":  result.RemotePath,
		"hook_output":  result.HookOutput,
	})
}

type transcriptSegmentForSRT struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker,omitempty"`
}

type transcriptPayloadForSRT struct {
	Text     string                    `json:"text"`
	Segments []transcriptSegmentForSRT `json:"segments"`
}

func bindOpenClawProfileRequest(c *gin.Context) (*OpenClawProfileRequest, bool) {
	var req OpenClawProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return nil, false
	}
	return &req, true
}

func validateOpenClawProfile(profile models.OpenClawProfile, requireSecrets bool) error {
	if strings.TrimSpace(profile.Name) == "" ||
		strings.TrimSpace(profile.IP) == "" ||
		strings.TrimSpace(profile.HookName) == "" ||
		strings.TrimSpace(profile.Message) == "" {
		return fmt.Errorf("name, host, hook name, and message are required")
	}
	if requireSecrets && (strings.TrimSpace(profile.SSHKey) == "" || strings.TrimSpace(profile.HookKey) == "") {
		return fmt.Errorf("ssh key and hook key are required")
	}
	if !requireSecrets && (strings.TrimSpace(profile.SSHKey) == "" || strings.TrimSpace(profile.HookKey) == "") {
		return fmt.Errorf("stored ssh key and hook key are required")
	}
	return nil
}

func toOpenClawProfileResponse(profile models.OpenClawProfile) openClawProfileResponse {
	return openClawProfileResponse{
		ID:         profile.ID,
		Name:       profile.Name,
		IP:         profile.IP,
		HookName:   profile.HookName,
		Message:    profile.Message,
		HasSSHKey:  strings.TrimSpace(profile.SSHKey) != "",
		HasHookKey: strings.TrimSpace(profile.HookKey) != "",
		CreatedAt:  profile.CreatedAt,
		UpdatedAt:  profile.UpdatedAt,
	}
}

func buildSRTFromRawTranscript(raw string, speakerMap map[string]string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty transcript")
	}

	var payload transcriptPayloadForSRT
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		var plain string
		if errString := json.Unmarshal([]byte(trimmed), &plain); errString == nil && strings.TrimSpace(plain) != "" {
			return buildSingleLineSRT(plain), nil
		}
		return "", fmt.Errorf("invalid transcript format")
	}

	if len(payload.Segments) > 0 {
		var out strings.Builder
		index := 1
		for _, segment := range payload.Segments {
			text := strings.TrimSpace(segment.Text)
			if text == "" {
				continue
			}

			start := math.Max(0, segment.Start)
			end := math.Max(start+0.001, segment.End)
			if strings.TrimSpace(segment.Speaker) != "" {
				displaySpeaker := segment.Speaker
				if custom, ok := speakerMap[segment.Speaker]; ok && strings.TrimSpace(custom) != "" {
					displaySpeaker = custom
				}
				text = fmt.Sprintf("%s: %s", displaySpeaker, text)
			}

			out.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s\n\n", index, formatSRTTime(start), formatSRTTime(end), text))
			index++
		}

		if out.Len() == 0 {
			return "", fmt.Errorf("no usable transcript segments")
		}
		return out.String(), nil
	}

	if strings.TrimSpace(payload.Text) != "" {
		return buildSingleLineSRT(payload.Text), nil
	}

	return "", fmt.Errorf("no transcript segments found")
}

func buildSingleLineSRT(text string) string {
	return fmt.Sprintf("1\n00:00:00,000 --> 00:00:05,000\n%s\n\n", strings.TrimSpace(text))
}

func formatSRTTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := int(seconds / 3600)
	minutes := int(math.Mod(seconds, 3600) / 60)
	secs := int(math.Mod(seconds, 60))
	milliseconds := int(math.Mod(seconds, 1.0) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, milliseconds)
}
