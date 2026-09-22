package repositories

import "gorm.io/gorm"

// Repositories is the unified data access container.
type Repositories struct {
	db *gorm.DB
}

// New initializes the central repositories container.
func New(db *gorm.DB) *Repositories {
	return &Repositories{db: db}
}

// DB returns the underlying GORM database instance.
func (r *Repositories) DB() *gorm.DB {
	if r == nil {
		return nil
	}
	return r.db
}
