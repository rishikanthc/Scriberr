package profile

import (
	"context"
	"errors"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("profile not found")

type Service struct {
	profiles repository.ProfileRepository
}

func NewService(profiles repository.ProfileRepository) *Service {
	return &Service{profiles: profiles}
}

func (s *Service) List(ctx context.Context, userID uint) ([]models.TranscriptionProfile, error) {
	items, _, err := s.profiles.ListByUser(ctx, userID, 0, 1000)
	return items, err
}

func (s *Service) Get(ctx context.Context, userID uint, id string) (*models.TranscriptionProfile, error) {
	profile, err := s.profiles.FindByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return profile, nil
}

func (s *Service) Create(ctx context.Context, profile *models.TranscriptionProfile) error {
	return s.profiles.CreateForUser(ctx, profile)
}

func (s *Service) Update(ctx context.Context, profile *models.TranscriptionProfile, defaultChanged bool) error {
	return s.profiles.UpdateForUser(ctx, profile, defaultChanged)
}

func (s *Service) Delete(ctx context.Context, userID uint, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	return s.profiles.DeleteForUser(ctx, id, userID)
}

func (s *Service) SetDefault(ctx context.Context, userID uint, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	return s.profiles.SetDefaultForUser(ctx, id, userID)
}

func (s *Service) Exists(ctx context.Context, userID uint, id string) (bool, error) {
	_, err := s.profiles.FindByIDForUser(ctx, id, userID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
