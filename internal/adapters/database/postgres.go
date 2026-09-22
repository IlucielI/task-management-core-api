package database

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-management/internal/config"
)

// Postgres wraps GORM and standard SQL DB connection pool.
type Postgres struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

// NewPostgres initializes a new PostgreSQL connection with pooling and verification.
func NewPostgres(cfg config.Config) (*Postgres, error) {
	dsn := cfg.DSN()

	gormLogLevel := logger.Warn
	if cfg.AppEnv == "development" {
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
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
		_ = sqlDB.Close()
		return nil, err
	}

	return &Postgres{
		db:    db,
		sqlDB: sqlDB,
	}, nil
}

// DB returns the GORM database instance for repository operations.
func (p *Postgres) DB() *gorm.DB {
	return p.db
}

// SQLDB returns the underlying sql.DB instance.
func (p *Postgres) SQLDB() *sql.DB {
	return p.sqlDB
}

// WithContext returns a new GORM DB instance scoped to the provided context.
func (p *Postgres) WithContext(ctx context.Context) *gorm.DB {
	return p.db.WithContext(ctx)
}

// Transaction executes operations within a database transaction.
func (p *Postgres) Transaction(fc func(tx *gorm.DB) error) error {
	return p.db.Transaction(fc)
}

// Ping checks if the database is reachable.
func (p *Postgres) Ping(ctx context.Context) error {
	return p.sqlDB.PingContext(ctx)
}

// Close gracefully closes the database connections.
func (p *Postgres) Close() error {
	return p.sqlDB.Close()
}
