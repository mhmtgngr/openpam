package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	ServiceName string
	Port        string
	LogLevel    string
	Environment string

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
	JWT               JWTConfig

	// Nested config structures for services
	Database DatabaseConfig
	Redis    RedisConfig
	ServiceNow ServiceNowConfig
	Jira     JiraConfig
	ITSM     ITSMConfig
	RateLimit RateLimitConfig

	// Service discovery
	AnalyticsServiceHost string
	AnalyticsServicePort string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// ServiceNowConfig holds ServiceNow configuration
type ServiceNowConfig struct {
	Enabled     bool
	InstanceURL string
	Username    string
	Password    string
}

// JiraConfig holds Jira configuration
type JiraConfig struct {
	Enabled   bool
	BaseURL   string
	Username  string
	APIToken  string
	ProjectKey string
}

// ITSMConfig holds ITSM sync configuration
type ITSMConfig struct {
	SyncInterval string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret string
}

// RateLimitConfig holds rate limit configuration
type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
}

// Load loads configuration from environment variables with defaults
func Load(serviceName string) Config {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnvInt("DB_PORT", 5432)
	dbUser := getEnv("DB_USER", "openpam")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "openpam")
	dbSSLMode := getEnv("DB_SSLMODE", "require")
	dbSSLRootCert := getEnv("DB_SSLROOTCERT", "/etc/ssl/certs/postgresql-ca.crt")

	// Build database URL
	databaseURL := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	if dbSSLRootCert != "" {
		databaseURL += fmt.Sprintf(" sslrootcert=%s", dbSSLRootCert)
	}

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnvInt("REDIS_PORT", 6379)
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)

	cfg := Config{
		ServiceName: serviceName,
		Port:        getEnv("PORT", defaultPort(serviceName)),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),

		DBHost:       dbHost,
		DBPort:       dbPort,
		DBUser:       dbUser,
		DBPassword:   dbPassword,
		DBName:       dbName,
		DBSSLMode:    dbSSLMode,
		DBSSLRootCert: dbSSLRootCert,

		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,

		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "/etc/openpam/jwt/private.pem"),
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "/etc/openpam/jwt/public.pem"),
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
		},

		Database: DatabaseConfig{
			URL: databaseURL,
		},
		Redis: RedisConfig{
			Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
			Password: redisPassword,
			DB:       redisDB,
		},
		ServiceNow: ServiceNowConfig{
			Enabled:     getEnvBool("SERVICENOW_ENABLED", false),
			InstanceURL: getEnv("SERVICENOW_INSTANCE_URL", ""),
			Username:    getEnv("SERVICENOW_USERNAME", ""),
			Password:    getEnv("SERVICENOW_PASSWORD", ""),
		},
		Jira: JiraConfig{
			Enabled:    getEnvBool("JIRA_ENABLED", false),
			BaseURL:    getEnv("JIRA_BASE_URL", ""),
			Username:   getEnv("JIRA_USERNAME", ""),
			APIToken:   getEnv("JIRA_API_TOKEN", ""),
			ProjectKey: getEnv("JIRA_PROJECT_KEY", ""),
		},
		ITSM: ITSMConfig{
			SyncInterval: getEnv("ITSM_SYNC_INTERVAL", "5m"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: getEnvInt("RATE_LIMIT_RPS", 100),
			BurstSize:         getEnvInt("RATE_LIMIT_BURST", 200),
		},

		AnalyticsServiceHost: getEnv("ANALYTICS_SERVICE_HOST", "localhost"),
		AnalyticsServicePort: getEnv("ANALYTICS_SERVICE_PORT", "8507"),
	}
	return cfg
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

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}
