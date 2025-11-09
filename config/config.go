package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port     int    `json:"port"`
	DBPath   string `json:"db_path"`
	DataDir  string `json:"data_dir"`
	LogLevel string `json:"log_level"` // debug, info, warn, error
	MaxPeers int    `json:"max_peers"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:     9999,
		DBPath:   "~/.whisper/whisper.db",
		DataDir:  "~/.whisper",
		LogLevel: "info",
		MaxPeers: 100,
	}

	// Override with environment variables
	if port := os.Getenv("WHISPER_PORT"); port != "" {
		p, _ := strconv.Atoi(port)
		cfg.Port = p
	}

	if db := os.Getenv("WHISPER_DB"); db != "" {
		cfg.DBPath = db
	}

	// Create data directory if not exists
	os.MkdirAll(expandPath(cfg.DataDir), 0700)

	return cfg, nil
}

func expandPath(path string) string {
	// Expand ~ to home directory
	if path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
