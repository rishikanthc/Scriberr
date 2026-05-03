package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrRegistrationClosed     = errors.New("registration is already complete")
	ErrUsernameInUse          = errors.New("username is already in use")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
	ErrInvalidCurrentPassword = errors.New("current password is invalid")
	ErrInvalidDefaultProfile  = errors.New("default profile is invalid")
	ErrDefaultProfileRequired = errors.New("default profile is required")
	ErrSmallLLMRequired       = errors.New("small LLM model is required")
	ErrAPIKeyNotFound         = errors.New("api key not found")
)

type Service struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	apiKeys       repository.APIKeyRepository
	profiles      repository.ProfileRepository
	llmConfigs    repository.LLMConfigRepository
	auth          *auth.AuthService
	now           func() time.Time
}

type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	User         *models.User
}

type SettingsUpdate struct {
	DefaultProfileIDSet      bool
	DefaultProfileID         *string
	AutoTranscriptionEnabled *bool
	AutoRenameEnabled        *bool
}

func NewService(users repository.UserRepository, refreshTokens repository.RefreshTokenRepository, apiKeys repository.APIKeyRepository, profiles repository.ProfileRepository, llmConfigs repository.LLMConfigRepository, authService *auth.AuthService) *Service {
	return &Service{
		users:         users,
		refreshTokens: refreshTokens,
		apiKeys:       apiKeys,
		profiles:      profiles,
		llmConfigs:    llmConfigs,
		auth:          authService,
		now:           time.Now,
	}
}

func (s *Service) RegistrationEnabled(ctx context.Context) (bool, error) {
	count, err := s.users.Count(ctx)
	return count == 0, err
}

func (s *Service) Register(ctx context.Context, username, password string) (*TokenResponse, error) {
	enabled, err := s.RegistrationEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrRegistrationClosed
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &models.User{Username: username, Password: passwordHash}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, ErrUsernameInUse
	}
	return s.issueTokenResponse(ctx, user)
}

func (s *Service) Login(ctx context.Context, username, password string) (*TokenResponse, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !auth.CheckPassword(password, user.Password) {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokenResponse(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (*TokenResponse, error) {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil, ErrInvalidRefreshToken
	}
	refreshToken, err := s.refreshTokens.FindByHash(ctx, hashToken(rawRefreshToken))
	if err != nil || refreshToken.RevokedAt != nil || !refreshToken.ExpiresAt.After(s.now()) {
		return nil, ErrInvalidRefreshToken
	}
	if err := s.refreshTokens.Revoke(ctx, refreshToken.ID); err != nil {
		return nil, err
	}
	user, err := s.users.FindByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	return s.issueTokenResponse(ctx, user)
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil
	}
	return s.refreshTokens.RevokeByHash(ctx, hashToken(rawRefreshToken))
}

func (s *Service) GetUser(ctx context.Context, userID uint) (*models.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *Service) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(currentPassword, user.Password) {
		return ErrInvalidCurrentPassword
	}
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.Password = passwordHash
	return s.users.Update(ctx, user)
}

func (s *Service) ChangeUsername(ctx context.Context, userID uint, newUsername, password string) (*models.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !auth.CheckPassword(password, user.Password) {
		return nil, ErrInvalidCurrentPassword
	}
	user.Username = newUsername
	if err := s.users.Update(ctx, user); err != nil {
		return nil, ErrUsernameInUse
	}
	return user, nil
}

func (s *Service) UpdateSettings(ctx context.Context, userID uint, update SettingsUpdate) (*models.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	defaultProfileID := user.DefaultProfileID
	if update.DefaultProfileIDSet {
		defaultProfileID = update.DefaultProfileID
	}
	if defaultProfileID != nil && strings.TrimSpace(*defaultProfileID) != "" {
		if _, err := s.profiles.FindByIDForUser(ctx, strings.TrimSpace(*defaultProfileID), userID); err != nil {
			return nil, ErrInvalidDefaultProfile
		}
	}
	autoTranscription := user.AutoTranscriptionEnabled
	if update.AutoTranscriptionEnabled != nil {
		autoTranscription = *update.AutoTranscriptionEnabled
	}
	if autoTranscription && (defaultProfileID == nil || strings.TrimSpace(*defaultProfileID) == "") {
		return nil, ErrDefaultProfileRequired
	}
	autoRename := user.AutoRenameEnabled
	if update.AutoRenameEnabled != nil {
		autoRename = *update.AutoRenameEnabled
	}
	if autoRename && !s.smallLLMReady(ctx, userID) {
		return nil, ErrSmallLLMRequired
	}
	user.DefaultProfileID = defaultProfileID
	user.AutoTranscriptionEnabled = autoTranscription
	user.AutoRenameEnabled = autoRename
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, userID uint) ([]models.APIKey, error) {
	return s.apiKeys.ListActiveByUser(ctx, userID)
}

func (s *Service) CreateAPIKey(ctx context.Context, userID uint, name, description string) (*models.APIKey, string, error) {
	rawKey := "sk_" + randomHex(32)
	key := &models.APIKey{
		UserID:      userID,
		Name:        strings.TrimSpace(name),
		Key:         rawKey,
		KeyPrefix:   rawKey[:8],
		KeyHash:     hashToken(rawKey),
		Description: &description,
	}
	if err := s.apiKeys.Create(ctx, key); err != nil {
		return nil, "", err
	}
	return key, rawKey, nil
}

func (s *Service) DeleteAPIKey(ctx context.Context, userID uint, id uint) error {
	key, err := s.apiKeys.FindByIDForUser(ctx, id, userID)
	if err != nil || key.RevokedAt != nil {
		return ErrAPIKeyNotFound
	}
	return s.apiKeys.RevokeForUser(ctx, id, userID)
}

func (s *Service) AuthenticateAPIKey(ctx context.Context, rawKey string) (*models.APIKey, error) {
	if strings.TrimSpace(rawKey) == "" {
		return nil, ErrInvalidCredentials
	}
	key, err := s.apiKeys.FindByKey(ctx, rawKey)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	now := s.now()
	key.LastUsed = &now
	_ = s.apiKeys.Update(ctx, key)
	return key, nil
}

func (s *Service) issueTokenResponse(ctx context.Context, user *models.User) (*TokenResponse, error) {
	accessToken, err := s.auth.GenerateToken(user)
	if err != nil {
		return nil, err
	}
	refreshToken := "rt_" + randomHex(32)
	stored := &models.RefreshToken{
		UserID:    user.ID,
		Hashed:    hashToken(refreshToken),
		ExpiresAt: s.now().Add(30 * 24 * time.Hour),
	}
	if err := s.refreshTokens.Create(ctx, stored); err != nil {
		return nil, err
	}
	return &TokenResponse{AccessToken: accessToken, RefreshToken: refreshToken, User: user}, nil
}

func (s *Service) smallLLMReady(ctx context.Context, userID uint) bool {
	config, err := s.llmConfigs.GetActiveByUser(ctx, userID)
	if err != nil {
		return false
	}
	return config != nil &&
		strings.TrimSpace(config.Provider) != "" &&
		strings.TrimSpace(llmBaseURL(config)) != "" &&
		config.SmallModel != nil &&
		strings.TrimSpace(*config.SmallModel) != ""
}

func llmBaseURL(config *models.LLMConfig) string {
	if config.BaseURL != nil && strings.TrimSpace(*config.BaseURL) != "" {
		return strings.TrimSpace(*config.BaseURL)
	}
	if config.OpenAIBaseURL != nil {
		return strings.TrimSpace(*config.OpenAIBaseURL)
	}
	return ""
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
