package admin

import (
	"context"
	"testing"

	"scriberr/internal/models"
	"scriberr/internal/transcription/scheduler"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSchedulerConfigDefaultsAndPersists(t *testing.T) {
	ctx := context.Background()
	users := &fakeAdminUsers{users: map[uint]*models.User{
		1: {ID: 1, Username: "admin", Role: "admin", Status: models.UserStatusActive},
	}}
	settings := &fakeSystemSettings{settings: map[string]*models.SystemSetting{}}
	service := NewService(users, nil, nil, settings)

	config, err := service.GetSchedulerConfig(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, scheduler.PolicyPriority, config.Policy)

	config, err = service.UpdateSchedulerConfig(ctx, 1, scheduler.Config{Policy: scheduler.PolicyFIFO})
	require.NoError(t, err)
	require.Equal(t, scheduler.PolicyFIFO, config.Policy)
	require.JSONEq(t, `{"policy":"fifo"}`, settings.settings[scheduler.SettingKey].ValueJSON)

	config, err = service.GetSchedulerConfig(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, scheduler.PolicyFIFO, config.Policy)
}

func TestSchedulerConfigRejectsInvalidAndNonAdmin(t *testing.T) {
	ctx := context.Background()
	users := &fakeAdminUsers{users: map[uint]*models.User{
		1: {ID: 1, Username: "admin", Role: "admin", Status: models.UserStatusActive},
		2: {ID: 2, Username: "member", Role: "user", Status: models.UserStatusActive},
	}}
	settings := &fakeSystemSettings{settings: map[string]*models.SystemSetting{}}
	service := NewService(users, nil, nil, settings)

	_, err := service.UpdateSchedulerConfig(ctx, 1, scheduler.Config{Policy: "random"})
	require.ErrorIs(t, err, ErrInvalidScheduler)
	require.Empty(t, settings.settings)

	_, err = service.GetSchedulerConfig(ctx, 2)
	require.ErrorIs(t, err, ErrForbidden)

	_, err = service.UpdateSchedulerConfig(ctx, 2, scheduler.Config{Policy: scheduler.PolicyPriority})
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, settings.settings)
}

type fakeAdminUsers struct {
	users map[uint]*models.User
}

func (f *fakeAdminUsers) Create(context.Context, *models.User) error { return nil }
func (f *fakeAdminUsers) Update(context.Context, *models.User) error { return nil }
func (f *fakeAdminUsers) ListUsersForAdmin(context.Context, int, int) ([]models.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeAdminUsers) CountActiveAdmins(context.Context) (int64, error) { return 1, nil }
func (f *fakeAdminUsers) FindUserByIDForAdmin(_ context.Context, userID uint) (*models.User, error) {
	user, ok := f.users[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *user
	return &copy, nil
}

type fakeSystemSettings struct {
	settings map[string]*models.SystemSetting
}

func (f *fakeSystemSettings) FindByKey(_ context.Context, key string) (*models.SystemSetting, error) {
	setting, ok := f.settings[key]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *setting
	return &copy, nil
}

func (f *fakeSystemSettings) Upsert(_ context.Context, setting *models.SystemSetting) error {
	copy := *setting
	f.settings[setting.Key] = &copy
	return nil
}
