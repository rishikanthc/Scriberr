package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/models"
)

var (
	ErrForbidden       = errors.New("admin access is required")
	ErrUserNotFound    = errors.New("user not found")
	ErrUsernameInUse   = errors.New("username is already in use")
	ErrInvalidUser     = errors.New("user fields are invalid")
	ErrLastActiveAdmin = errors.New("last active admin cannot be disabled or demoted")
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	FindUserByIDForAdmin(ctx context.Context, userID uint) (*models.User, error)
	ListUsersForAdmin(ctx context.Context, offset, limit int) ([]models.User, int64, error)
	CountActiveAdmins(ctx context.Context) (int64, error)
}

type RefreshTokenRepository interface {
	RevokeByUser(ctx context.Context, userID uint) error
}

type APIKeyRepository interface {
	RevokeByUser(ctx context.Context, userID uint) error
}

type Service struct {
	users         UserRepository
	refreshTokens RefreshTokenRepository
	apiKeys       APIKeyRepository
	now           func() time.Time
}

type CreateUserCommand struct {
	Username    string
	Email       *string
	DisplayName *string
	Role        string
	Password    string
}

type UpdateUserCommand struct {
	Email            *string
	DisplayName      *string
	Role             *string
	Status           *string
	ClearEmail       bool
	ClearDisplayName bool
}

func NewService(users UserRepository, refreshTokens RefreshTokenRepository, apiKeys APIKeyRepository) *Service {
	return &Service{
		users:         users,
		refreshTokens: refreshTokens,
		apiKeys:       apiKeys,
		now:           time.Now,
	}
}

func (s *Service) ListUsers(ctx context.Context, actorID uint, offset, limit int) ([]models.User, int64, error) {
	if _, err := s.requireActiveAdmin(ctx, actorID); err != nil {
		return nil, 0, err
	}
	return s.users.ListUsersForAdmin(ctx, offset, limit)
}

func (s *Service) CreateUser(ctx context.Context, actorID uint, cmd CreateUserCommand) (*models.User, error) {
	if _, err := s.requireActiveAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	username := strings.TrimSpace(cmd.Username)
	password := strings.TrimSpace(cmd.Password)
	role := normalizeRole(cmd.Role)
	if username == "" || len(username) < 3 || len(password) < 8 || !validRole(role) {
		return nil, ErrInvalidUser
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Username:    username,
		Password:    hash,
		Email:       cleanOptional(cmd.Email),
		DisplayName: cleanOptional(cmd.DisplayName),
		Role:        role,
		Status:      models.UserStatusActive,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, ErrUsernameInUse
	}
	return user, nil
}

func (s *Service) GetUser(ctx context.Context, actorID uint, targetID uint) (*models.User, error) {
	if _, err := s.requireActiveAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	return s.findTarget(ctx, targetID)
}

func (s *Service) UpdateUser(ctx context.Context, actorID uint, targetID uint, cmd UpdateUserCommand) (*models.User, error) {
	if _, err := s.requireActiveAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	user, err := s.findTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}
	nextRole := user.Role
	if cmd.Role != nil {
		nextRole = normalizeRole(*cmd.Role)
		if !validRole(nextRole) {
			return nil, ErrInvalidUser
		}
	}
	nextStatus := user.Status
	if cmd.Status != nil {
		nextStatus = strings.TrimSpace(*cmd.Status)
		if !validStatus(nextStatus) {
			return nil, ErrInvalidUser
		}
	}
	if err := s.ensureLastAdminNotRemoved(ctx, user, nextRole, nextStatus); err != nil {
		return nil, err
	}
	user.Role = nextRole
	user.Status = nextStatus
	if cmd.ClearEmail {
		user.Email = nil
	} else if cmd.Email != nil {
		user.Email = cleanOptional(cmd.Email)
	}
	if cmd.ClearDisplayName {
		user.DisplayName = nil
	} else if cmd.DisplayName != nil {
		user.DisplayName = cleanOptional(cmd.DisplayName)
	}
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	if nextStatus == models.UserStatusDisabled {
		if err := s.revokeCredentials(ctx, user.ID, true); err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (s *Service) ResetPassword(ctx context.Context, actorID uint, targetID uint, password string) error {
	if _, err := s.requireActiveAdmin(ctx, actorID); err != nil {
		return err
	}
	user, err := s.findTarget(ctx, targetID)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(password)) < 8 {
		return ErrInvalidUser
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	now := s.now()
	user.Password = hash
	user.PasswordChangedAt = &now
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}
	return s.revokeCredentials(ctx, user.ID, false)
}

func (s *Service) DisableUser(ctx context.Context, actorID uint, targetID uint) (*models.User, error) {
	return s.UpdateUser(ctx, actorID, targetID, UpdateUserCommand{Status: stringPtr(models.UserStatusDisabled)})
}

func (s *Service) EnableUser(ctx context.Context, actorID uint, targetID uint) (*models.User, error) {
	return s.UpdateUser(ctx, actorID, targetID, UpdateUserCommand{Status: stringPtr(models.UserStatusActive)})
}

func (s *Service) requireActiveAdmin(ctx context.Context, actorID uint) (*models.User, error) {
	if s == nil || s.users == nil {
		return nil, ErrForbidden
	}
	user, err := s.users.FindUserByIDForAdmin(ctx, actorID)
	if err != nil {
		return nil, ErrForbidden
	}
	if user.Role != "admin" || user.Status != models.UserStatusActive {
		return nil, ErrForbidden
	}
	return user, nil
}

func (s *Service) findTarget(ctx context.Context, targetID uint) (*models.User, error) {
	user, err := s.users.FindUserByIDForAdmin(ctx, targetID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *Service) ensureLastAdminNotRemoved(ctx context.Context, user *models.User, nextRole string, nextStatus string) error {
	if user.Role != "admin" || user.Status != models.UserStatusActive {
		return nil
	}
	if nextRole == "admin" && nextStatus == models.UserStatusActive {
		return nil
	}
	count, err := s.users.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastActiveAdmin
	}
	return nil
}

func (s *Service) revokeCredentials(ctx context.Context, userID uint, includeAPIKeys bool) error {
	if s.refreshTokens != nil {
		if err := s.refreshTokens.RevokeByUser(ctx, userID); err != nil {
			return err
		}
	}
	if includeAPIKeys && s.apiKeys != nil {
		if err := s.apiKeys.RevokeByUser(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

func normalizeRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return "user"
	}
	return role
}

func validRole(role string) bool {
	return role == "admin" || role == "user"
}

func validStatus(status string) bool {
	return status == models.UserStatusActive || status == models.UserStatusDisabled
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	clean := strings.TrimSpace(*value)
	if clean == "" {
		return nil
	}
	return &clean
}

func stringPtr(value string) *string {
	return &value
}
