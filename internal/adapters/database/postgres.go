package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-management/internal/config"
)

// Postgres wraps GORM and standard SQL DB connection pool.
type Postgres struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

// NewPostgres initializes a new PostgreSQL connection with pooling, verification, and migrations.
func NewPostgres(cfg config.Config) (*Postgres, error) {
	dsn := cfg.DSN()

	gormLogLevel := logger.Warn
	if cfg.AppEnv == "development" {
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(gormPostgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Apply connection pool settings
	sqlDB.SetMaxOpenConns(cfg.DBPoolMaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.DBPoolMaxIdleConn)
	sqlDB.SetConnMaxLifetime(cfg.DBPoolMaxConnLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.DBPoolMaxConnIdleTime)

	// Verify connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		if closeErr := sqlDB.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping failed (%w) and failed to close db: %v", err, closeErr)
		}
		return nil, err
	}

	// Run migrations using golang-migrate
	if err := RunMigrations(sqlDB); err != nil {
		if closeErr := sqlDB.Close(); closeErr != nil {
			return nil, fmt.Errorf("migration failed (%w) and failed to close db: %v", err, closeErr)
		}
		return nil, fmt.Errorf("failed to execute migrations: %w", err)
	}

	return &Postgres{
		db:    db,
		sqlDB: sqlDB,
	}, nil
}

// RunMigrations executes database migrations from the migrations directory using golang-migrate.
func RunMigrations(sqlDB *sql.DB) error {
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres migrate driver: %w", err)
	}

	migrationsDir := findMigrationsDir()
	sourceUri := "file://" + migrationsDir

	m, err := migrate.NewWithDatabaseInstance(sourceUri, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil || dbErr != nil {
			// Resource cleanup completed with potential warnings
			_ = srcErr
			_ = dbErr
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// findMigrationsDir locates the migrations directory searching upward from current working directory.
func findMigrationsDir() string {
	if envDir := os.Getenv("MIGRATIONS_DIR"); envDir != "" {
		cleaned := filepath.Clean(envDir)
		if fi, err := os.Stat(cleaned); err == nil && fi.IsDir() {
			if abs, err := filepath.Abs(cleaned); err == nil {
				return abs
			}
			return cleaned
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		curr := filepath.Clean(cwd)
		for i := 0; i < 6; i++ {
			candidate := filepath.Join(curr, "migrations")
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				return candidate
			}
			parent := filepath.Dir(curr)
			if parent == curr {
				break
			}
			curr = parent
		}
	}

	return "migrations"
}

// DB returns the GORM database instance for repository operations.
func (p *Postgres) DB() *gorm.DB {
	if p == nil {
		return nil
	}
	return p.db
}

// SQLDB returns the underlying sql.DB instance.
func (p *Postgres) SQLDB() *sql.DB {
	if p == nil {
		return nil
	}
	return p.sqlDB
}

// WithContext returns a new GORM DB instance scoped to the provided context.
func (p *Postgres) WithContext(ctx context.Context) *gorm.DB {
	if p == nil || p.db == nil {
		return nil
	}
	return p.db.WithContext(ctx)
}

// Transaction executes operations within a database transaction.
func (p *Postgres) Transaction(fc func(tx *gorm.DB) error) error {
	if p == nil || p.db == nil {
		return errors.New("postgres connection is nil")
	}
	return p.db.Transaction(fc)
}

// Ping checks if the database is reachable.
func (p *Postgres) Ping(ctx context.Context) error {
	if p == nil || p.sqlDB == nil {
		return errors.New("database connection is nil")
	}
	return p.sqlDB.PingContext(ctx)
}

// Close gracefully closes the database connections.
func (p *Postgres) Close() error {
	if p == nil || p.sqlDB == nil {
		return nil
	}
	return p.sqlDB.Close()
}
