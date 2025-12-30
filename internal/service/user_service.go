package service

import (
	"context"
	"errors"
	"fmt"
	"scriberr/internal/auth"
	"scriberr/internal/models"
	"scriberr/internal/repository"
)

// UserService handles user business logic
type UserService interface {
	Register(ctx context.Context, username, password string) (*models.User, error)
	Login(ctx context.Context, username, password string) (string, *models.User, error)
	ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error
	ChangeUsername(ctx context.Context, userID uint, newUsername, password string) error
	GetUser(ctx context.Context, userID uint) (*models.User, error)
}

type userService struct {
	userRepo    repository.UserRepository
	authService *auth.AuthService
}

func NewUserService(userRepo repository.UserRepository, authService *auth.AuthService) UserService {
	return &userService{
		userRepo:    userRepo,
		authService: authService,
	}
}

func (s *userService) Register(ctx context.Context, username, password string) (*models.User, error) {
	// Check if user exists
	existing, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil && err.Error() != "record not found" { // GORM specific check, might need abstraction
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("username already exists")
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username: username,
		Password: hashedPassword,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Login(ctx context.Context, username, password string) (string, *models.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if !auth.CheckPassword(password, user.Password) {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := s.authService.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

func (s *userService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	if !auth.CheckPassword(currentPassword, user.Password) {
		return errors.New("incorrect current password")
	}

	hashedPassword, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.userRepo.Update(ctx, user)
}

func (s *userService) ChangeUsername(ctx context.Context, userID uint, newUsername, password string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	if !auth.CheckPassword(password, user.Password) {
		return errors.New("incorrect password")
	}

	// Check if new username is taken
	existing, err := s.userRepo.FindByUsername(ctx, newUsername)
	if err == nil && existing != nil {
		return errors.New("username already taken")
	}

	user.Username = newUsername
	return s.userRepo.Update(ctx, user)
}

func (s *userService) GetUser(ctx context.Context, userID uint) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}
