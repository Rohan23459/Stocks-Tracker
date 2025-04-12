package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global PostgreSQL database connection.
var DB *gorm.DB

// Rdb is the global Redis client.
var Rdb *redis.Client

// Ctx is the context for Redis operations.
var Ctx = context.Background()

func InitDB() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), // Use environment variable instead of "MyProjects"
		os.Getenv("DB_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
}

// InitRedis initializes the Redis connection.
func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	if err := Rdb.Ping(Ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
}
