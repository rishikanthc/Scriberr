package repository

import (
	"context"
	"testing"

	"scriberr/internal/models"
	"scriberr/internal/transcription/scheduler"

	"github.com/stretchr/testify/require"
)

func TestSystemSettingsRepositoryUpsertAndFind(t *testing.T) {
	db := openJobQueueTestDB(t)
	repo := NewSystemSettingsRepository(db)
	ctx := context.Background()

	raw, err := scheduler.Marshal(scheduler.Config{Policy: scheduler.PolicyFIFO})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, &models.SystemSetting{Key: scheduler.SettingKey, ValueJSON: raw}))

	setting, err := repo.FindByKey(ctx, scheduler.SettingKey)
	require.NoError(t, err)
	require.JSONEq(t, `{"policy":"fifo"}`, setting.ValueJSON)

	raw, err = scheduler.Marshal(scheduler.Config{Policy: scheduler.PolicyPriority})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, &models.SystemSetting{Key: scheduler.SettingKey, ValueJSON: raw}))

	setting, err = repo.FindByKey(ctx, scheduler.SettingKey)
	require.NoError(t, err)
	require.JSONEq(t, `{"policy":"priority"}`, setting.ValueJSON)
}
