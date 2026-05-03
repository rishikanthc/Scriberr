package repository

import "gorm.io/gorm"

// ErrRecordNotFound is the repository-layer not-found sentinel.
var ErrRecordNotFound = gorm.ErrRecordNotFound

// ErrInvalidData is returned when GORM rejects an invalid model shape.
var ErrInvalidData = gorm.ErrInvalidData
