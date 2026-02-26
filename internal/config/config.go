package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	ServiceName string
	Port        string
	LogLevel    string

	DBHost       string
	DBPort       int
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	DBSSLRootCert string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	JWTPrivateKeyPath string
	JWTPublicKeyPath  string

	// Service discovery
	AnalyticsServiceHost string
	AnalyticsServicePort string
}

// Load loads configuration from environment variables with defaults
func Load(serviceName string) Config {
	return Config{
		ServiceName: serviceName,
		Port:        getEnv("PORT", defaultPort(serviceName)),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnvInt("DB_PORT", 5432),
		DBUser:       getEnv("DB_USER", "openpam"),
		DBPassword:   getEnv("DB_PASSWORD", ""),
		DBName:       getEnv("DB_NAME", "openpam"),
		DBSSLMode:    getEnv("DB_SSLMODE", "require"),
		DBSSLRootCert: getEnv("DB_SSLROOTCERT", "/etc/ssl/certs/postgresql-ca.crt"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "/etc/openpam/jwt/private.pem"),
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "/etc/openpam/jwt/public.pem"),

		AnalyticsServiceHost: getEnv("ANALYTICS_SERVICE_HOST", "localhost"),
		AnalyticsServicePort: getEnv("ANALYTICS_SERVICE_PORT", "8507"),
	}
}

func defaultPort(serviceName string) string {
	switch serviceName {
	case "gateway":
		return "8500"
	case "session-service":
		return "8501"
	case "vault-service":
		return "8502"
	case "credential-service":
		return "8503"
	case "audit-service":
		return "8504"
	case "admin-service":
		return "8505"
	case "discovery-service":
		return "8506"
	case "identity-service":
		return "8507"
	case "analytics-service":
		return "8508"
	default:
		return "8080"
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
