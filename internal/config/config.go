// Package config loads application configuration from environment variables and .env files.
package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration values.
type Config struct {
	// HTTPPort is the port the HTTP server listens on.
	HTTPPort string
	// DB_DSN is the PostgreSQL connection string.
	DB_DSN string
	// RabbitURL is the RabbitMQ connection URL.
	RabbitURL string
}

// getEnv reads an environment variable by key.
func getEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return val, nil
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found")
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	dbUser, err := getEnv("DB_USER")
	if err != nil {
		return nil, err
	}
	dbPass, err := getEnv("DB_PASSWORD")
	if err != nil {
		return nil, err
	}
	dbHost, err := getEnv("DB_HOST")
	if err != nil {
		return nil, err
	}
	dbPort, err := getEnv("DB_PORT")
	if err != nil {
		return nil, err
	}
	dbName, err := getEnv("DB_NAME")
	if err != nil {
		return nil, err
	}

	rbUser, err := getEnv("RABBIT_USER")
	if err != nil {
		return nil, err
	}
	rbPass, err := getEnv("RABBIT_PASSWORD")
	if err != nil {
		return nil, err
	}
	rbHost, err := getEnv("RABBIT_HOST")
	if err != nil {
		return nil, err
	}
	rbPort, err := getEnv("RABBIT_PORT")
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)

	rurl := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		rbUser, rbPass, rbHost, rbPort,
	)

	return &Config{
		HTTPPort:  port,
		DB_DSN:    dsn,
		RabbitURL: rurl,
	}, nil
}
