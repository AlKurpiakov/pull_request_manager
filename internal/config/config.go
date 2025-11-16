package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBConn       string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func LoadFromEnv() *Config {
	readTimeout := getEnvAsInt("READ_TIMEOUT", 10)
	writeTimeout := getEnvAsInt("WRITE_TIMEOUT", 10)
	idleTimeout := getEnvAsInt("IDLE_TIMEOUT", 30)

	return &Config{
		DBConn:       getEnv("DB_CONN", "postgres://postgres:postgres@localhost:5432/pr_manager?sslmode=disable"),
		Port:         getEnv("PORT", "8080"),
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
		IdleTimeout:  time.Duration(idleTimeout) * time.Second,
	}
}

func getEnv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}

func getEnvAsInt(k string, d int) int {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return i
}
