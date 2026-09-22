package repositories

import (
	"gorm.io/gorm"

	"task-management/internal/adapters/redis"
)

// Repositories is the unified data access container.
type Repositories struct {
	db  *gorm.DB
	rdb *redis.Redis
}

// New initializes the central repositories container.
func New(db *gorm.DB, rdb ...*redis.Redis) *Repositories {
	var redisAdapter *redis.Redis
	if len(rdb) > 0 {
		redisAdapter = rdb[0]
	}
	return &Repositories{
		db:  db,
		rdb: redisAdapter,
	}
}

// DB returns the underlying GORM database instance.
func (r *Repositories) DB() *gorm.DB {
	if r == nil {
		return nil
	}
	return r.db
}

// Redis returns the underlying Redis adapter instance.
func (r *Repositories) Redis() *redis.Redis {
	if r == nil {
		return nil
	}
	return r.rdb
}

