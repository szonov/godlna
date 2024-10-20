package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log/slog"
	"os"
	"time"
)

type Network struct {
	IFace string `toml:"iface"`
	IP    string `toml:"ip"`
}

type Server struct {
	Header string `toml:"header"`
	Port   int    `toml:"port"`
}

type Ssdp struct {
	Disable bool `toml:"disable"`
}

type Logger struct {
	Level slog.Level `toml:"level"`
}

type Device struct {
	FriendlyName string `toml:"friendlyName"`
	UUID         string `toml:"uuid"`
}

type Store struct {
	MediaDir      string        `toml:"media_dir"`
	CacheDir      string        `toml:"cache_dir"`
	CacheLifeTime time.Duration `toml:"cache_life_time"`
}

type Config struct {
	Network Network `toml:"network"`
	Server  Server  `toml:"server"`
	Ssdp    Ssdp    `toml:"ssdp"`
	Logger  Logger  `toml:"logger"`
	Device  Device  `toml:"device"`
	Store   Store   `toml:"store"`
}

func Read(configFile string) (*Config, error) {
	_, err := os.Stat(configFile)
	if err != nil {
		return nil, fmt.Errorf("config file is missing: %w", err)
	}
	var config *Config
	if _, err = toml.DecodeFile(configFile, &config); err != nil {
		return nil, fmt.Errorf("can not parse config: %w", err)
	}
	return config, nil
}
