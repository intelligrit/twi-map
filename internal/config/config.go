package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds all user-facing configuration for twi-map.
type Config struct {
	Data    DataConfig    `toml:"data"`
	Server  ServerConfig  `toml:"server"`
	Extract ExtractConfig `toml:"extract"`
	Scrape  ScrapeConfig  `toml:"scrape"`
}

type DataConfig struct {
	Dir string `toml:"dir"`
}

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type ExtractConfig struct {
	Model     string `toml:"model"`
	MaxTokens int    `toml:"max_tokens"`
}

type ScrapeConfig struct {
	RateLimit float64 `toml:"rate_limit"`
}

// Defaults returns a Config populated with built-in default values.
func Defaults() *Config {
	return &Config{
		Data:    DataConfig{Dir: "data"},
		Server:  ServerConfig{Host: "localhost", Port: 8080},
		Extract: ExtractConfig{Model: "claude-sonnet-4-20250514", MaxTokens: 64000},
		Scrape:  ScrapeConfig{RateLimit: 1.0},
	}
}

// Load reads a TOML config file. If the file does not exist, built-in
// defaults are returned without error.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
