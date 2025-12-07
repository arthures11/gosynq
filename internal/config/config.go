package config

import (
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Worker   WorkerConfig
	Retries  RetryConfig
}

type ServerConfig struct {
	Port        int
	Host        string
	MetricsPort int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type WorkerConfig struct {
	PoolSize          int
	VisibilityTimeout time.Duration
	Concurrency       int
}

type RetryConfig struct {
	DefaultStrategy string
	DefaultInterval int
	MaxAttempts     int
	ExponentialBase float64
}

func NewDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        8080,
			Host:        "localhost",
			MetricsPort: 9090,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			DBName:   "gosynq_db",
			SSLMode:  "disable",
		},
		Worker: WorkerConfig{
			PoolSize:          10,
			VisibilityTimeout: 30 * time.Second,
			Concurrency:       5,
		},
		Retries: RetryConfig{
			DefaultStrategy: "exponential",
			DefaultInterval: 5,
			MaxAttempts:     5,
			ExponentialBase: 2.0,
		},
	}
}
