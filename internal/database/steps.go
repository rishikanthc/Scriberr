package database

import (
	"fmt"

	"scriberr/internal/models"

	"gorm.io/gorm"
)

type migrationStep func(*gorm.DB) error

var schemaSteps = map[int]migrationStep{
	2: migrateStepV1ToV2,
}

func runSchemaSteps(tx *gorm.DB, currentVersion int) error {
	for version := currentVersion + 1; version <= latestSchemaVersion; version++ {
		step := schemaSteps[version]
		if step != nil {
			if err := step(tx); err != nil {
				return fmt.Errorf("apply schema migration v%d: %w", version, err)
			}
		}
		if err := recordSchemaVersion(tx, version); err != nil {
			return fmt.Errorf("record schema version %d: %w", version, err)
		}
	}
	return nil
}

func migrateStepV1ToV2(tx *gorm.DB) error {
	if err := ensureSingleDefaultPerUser(tx); err != nil {
		return err
	}
	if err := backfillCompatibilityColumns(tx); err != nil {
		return err
	}
	return nil
}

func backfillCompatibilityColumns(tx *gorm.DB) error {
	if err := resaveAll(tx, &[]models.User{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.APIKey{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.TranscriptionProfile{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.TranscriptionJob{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.TranscriptionJobExecution{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.MultiTrackFile{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.SummaryTemplate{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.Note{}); err != nil {
		return err
	}
	if err := resaveAll(tx, &[]models.LLMConfig{}); err != nil {
		return err
	}
	return nil
}

func resaveAll[T any](tx *gorm.DB, rows *[]T) error {
	if err := tx.Find(rows).Error; err != nil {
		return err
	}
	for _, row := range *rows {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
