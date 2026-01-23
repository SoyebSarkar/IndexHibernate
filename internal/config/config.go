package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type ReloadMode string

const (
	ReloadAsync    ReloadMode = "async"
	ReloadBlocking ReloadMode = "blocking" // future
)

type Config struct {
	TypesenseURL         string
	TypesenseAPIKey      string
	Port                 string
	OffloadAfter         time.Duration
	DrainGracePeriod     time.Duration
	SchedulerInterval    time.Duration
	ReloadMode           ReloadMode
	MaxConcurrentReloads int
	SnapshotDir          string
	StateDBPath          string
	ListenAddr           string
}

func Load() *Config {
	cfg := &Config{
		TypesenseURL:         getEnv("TYPESENSE_URL", "http://localhost:8108"),
		Port:                 getEnv("PORT", "8080"),
		TypesenseAPIKey:      getEnv("TYPESENSE_API_KEY", "xyz"),
		OffloadAfter:         getDuration("OFFLOAD_AFTER", 6*time.Hour),
		DrainGracePeriod:     getDuration("DRAIN_GRACE_PERIOD", 30*time.Second),
		SchedulerInterval:    getDuration("SCHEDULER_INTERVAL", 10*time.Minute),
		ReloadMode:           ReloadAsync,
		MaxConcurrentReloads: getInt("MAX_CONCURRENT_RELOADS", 2),
		SnapshotDir:          getEnv("SNAPSHOT_DIR", "./snapshots"),
		StateDBPath:          getEnv("STATE_DB_PATH", "./state.db"),
		ListenAddr:           getEnv("LISTEN_ADDR", "localhost"),
	}
	if v := os.Getenv("RELOAD_MODE"); v != "" {
		switch ReloadMode(v) {
		case ReloadAsync, ReloadBlocking:
			cfg.ReloadMode = ReloadMode(v)
		default:
			log.Fatalf("invalid RELOAD_MODE: %s", v)
		}
	}

	logConfig(cfg)
	return cfg
}

// Helper functions to read env vars with defaults
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("invalid duration for %s", key)
		}
		return d
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid int for %s", key)
		}
		return i
	}
	return def
}
func logConfig(cfg *Config) {
	log.Printf(
		"config offload_after=%s drain_grace=%s scheduler_interval=%s reload_mode=%s max_concurrent_reloads=%d",
		cfg.OffloadAfter,
		cfg.DrainGracePeriod,
		cfg.SchedulerInterval,
		cfg.ReloadMode,
		cfg.MaxConcurrentReloads,
	)

}
