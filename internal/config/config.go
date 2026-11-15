package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type AppMode string

const (
	ModeCloud AppMode = "CLOUD"
	ModeEdge  AppMode = "EDGE"
)

type Config struct {
	AppEnv         string
	AppMode        AppMode
	Port           int
	DBDriver       string
	DBDSN          string
	Ed25519PrivKey string
	Ed25519PubKey  string
	CloudSyncURL   string
	SyncInterval   time.Duration
	TenantID       string
	OutletID       string
	SyncBatchSize  int
	EdgeAuthToken  string
	APIAuthToken   string
	AllowedOrigins []string
}

func Load() (*Config, error) {
	appModeStr := strings.ToUpper(getEnv("APP_MODE", "EDGE"))
	var appMode AppMode
	switch appModeStr {
	case "CLOUD":
		appMode = ModeCloud
	case "EDGE":
		appMode = ModeEdge
	default:
		return nil, fmt.Errorf("invalid APP_MODE: %s (must be CLOUD or EDGE)", appModeStr)
	}

	port, err := strconv.Atoi(getEnv("PORT", defaultPort(appMode)))
	if err != nil {
		port = 8080
	}

	dbDriver := getEnv("DB_DRIVER", "")
	dbDSN := getEnv("DB_DSN", "")

	if appMode == ModeCloud {
		if dbDriver == "" {
			dbDriver = "postgres"
		}
		if dbDSN == "" {
			dbDSN = getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pos_cloud?sslmode=disable")
		}
	} else {
		if dbDriver == "" {
			dbDriver = "sqlite"
		}
		if dbDSN == "" {
			dbDSN = getEnv("SQLITE_PATH", "edge.db")
		}
	}

	privKey := getEnv("ED25519_PRIVATE_KEY", "")
	pubKey := getEnv("ED25519_PUBLIC_KEY", "")

	syncIntervalSec, err := strconv.Atoi(getEnv("SYNC_INTERVAL_SEC", "5"))
	if err != nil || syncIntervalSec <= 0 {
		syncIntervalSec = 5
	}

	batchSize, err := strconv.Atoi(getEnv("SYNC_BATCH_SIZE", "50"))
	if err != nil || batchSize <= 0 {
		batchSize = 50
	}

	appEnv := strings.ToLower(getEnv("APP_ENV", "development"))

	var allowedOrigins []string
	if rawOrigins := getEnv("ALLOWED_ORIGINS", ""); rawOrigins != "" {
		for _, o := range strings.Split(rawOrigins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	cfg := &Config{
		AppEnv:         appEnv,
		AppMode:        appMode,
		Port:           port,
		DBDriver:       dbDriver,
		DBDSN:          dbDSN,
		Ed25519PrivKey: privKey,
		Ed25519PubKey:  pubKey,
		CloudSyncURL:   strings.TrimRight(getEnv("CLOUD_SYNC_URL", "http://localhost:8080"), "/"),
		SyncInterval:   time.Duration(syncIntervalSec) * time.Second,
		TenantID:       getEnv("TENANT_ID", "tenant-default"),
		OutletID:       getEnv("OUTLET_ID", "outlet-default"),
		SyncBatchSize:  batchSize,
		EdgeAuthToken:  getEnv("EDGE_AUTH_TOKEN", ""),
		APIAuthToken:   getEnv("API_AUTH_TOKEN", ""),
		AllowedOrigins: allowedOrigins,
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.AppMode == ModeCloud {
		if c.DBDriver != "postgres" {
			return errors.New("CLOUD mode requires 'postgres' DBDriver")
		}
		if c.AppEnv == "production" {
			if c.Ed25519PrivKey == "" {
				return errors.New("CLOUD mode in production requires ED25519_PRIVATE_KEY")
			}
			if c.EdgeAuthToken == "" {
				return errors.New("CLOUD mode in production requires EDGE_AUTH_TOKEN")
			}
		}
	} else if c.AppMode == ModeEdge {
		if c.DBDriver != "sqlite" {
			return errors.New("EDGE mode requires 'sqlite' DBDriver")
		}
	}
	return nil
}

func defaultPort(mode AppMode) string {
	if mode == ModeCloud {
		return "8080"
	}
	return "8081"
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}
