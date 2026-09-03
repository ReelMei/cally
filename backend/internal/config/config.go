package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                  string
	Env                   string
	AllowedOrigins        []string
	MaxRoomParticipants   int
	RoomIdleTimeoutMin    int
	ReadTimeoutSec        int
	WriteTimeoutSec       int
	IdleTimeoutSec        int

	// PostgreSQL Configuration
	DatabaseURL           string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	DBSSLMode             string
	DBMaxOpenConns        int
	DBMaxIdleConns        int
	DBEnable              bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")

	originsRaw := getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173")
	var origins []string
	for _, o := range strings.Split(originsRaw, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	maxParticipants, err := strconv.Atoi(getEnv("MAX_ROOM_PARTICIPANTS", "6"))
	if err != nil || maxParticipants <= 0 {
		maxParticipants = 6
	}

	roomIdleMin, err := strconv.Atoi(getEnv("ROOM_IDLE_TIMEOUT_MINUTES", "30"))
	if err != nil || roomIdleMin <= 0 {
		roomIdleMin = 30
	}

	readTimeout, err := strconv.Atoi(getEnv("READ_TIMEOUT_SECONDS", "15"))
	if err != nil || readTimeout <= 0 {
		readTimeout = 15
	}

	writeTimeout, err := strconv.Atoi(getEnv("WRITE_TIMEOUT_SECONDS", "15"))
	if err != nil || writeTimeout <= 0 {
		writeTimeout = 15
	}

	idleTimeout, err := strconv.Atoi(getEnv("IDLE_TIMEOUT_SECONDS", "60"))
	if err != nil || idleTimeout <= 0 {
		idleTimeout = 60
	}

	// Database config
	databaseURL := getEnv("DATABASE_URL", "")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "cally_db")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	maxOpenConns, err := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	if err != nil || maxOpenConns <= 0 {
		maxOpenConns = 25
	}

	maxIdleConns, err := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	if err != nil || maxIdleConns <= 0 {
		maxIdleConns = 10
	}

	dbEnableStr := getEnv("DB_ENABLE", "true")
	dbEnable := dbEnableStr == "true" || dbEnableStr == "1"

	return &Config{
		Port:                port,
		Env:                 env,
		AllowedOrigins:      origins,
		MaxRoomParticipants: maxParticipants,
		RoomIdleTimeoutMin:  roomIdleMin,
		ReadTimeoutSec:      readTimeout,
		WriteTimeoutSec:     writeTimeout,
		IdleTimeoutSec:      idleTimeout,
		DatabaseURL:         databaseURL,
		DBHost:              dbHost,
		DBPort:              dbPort,
		DBUser:              dbUser,
		DBPassword:          dbPassword,
		DBName:              dbName,
		DBSSLMode:           dbSSLMode,
		DBMaxOpenConns:      maxOpenConns,
		DBMaxIdleConns:      maxIdleConns,
		DBEnable:            dbEnable,
	}, nil
}

func (c *Config) GetDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) IsOriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

func (c *Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}
