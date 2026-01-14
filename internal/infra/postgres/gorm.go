package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/sifan077/PowerURL/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewGorm returns a gorm.DB configured for the application's Postgres instance.
func NewGorm(cfg config.PostgresConfig) (*gorm.DB, error) {
	dsn := ConnString(cfg)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: open gorm connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres: retrieve sql db: %w", err)
	}

	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// AutoMigrate uses GORM to perform schema migrations for the provided models.
func AutoMigrate(ctx context.Context, db *gorm.DB, models ...interface{}) error {
	if db == nil || len(models) == 0 {
		return nil
	}

	if err := db.WithContext(ctx).AutoMigrate(models...); err != nil {
		return fmt.Errorf("postgres: auto migrate: %w", err)
	}

	return nil
}
