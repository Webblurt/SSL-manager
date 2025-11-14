package utils

import (
	"errors"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	CertDir     string   `yaml:"cert_dir"`
	ReloadNginx bool     `yaml:"reload_nginx"`
	ReloadCmd   string   `yaml:"reload_cmd"`
	Domains     []string `yaml:"domains"`
	Email       string   `yaml:"email"`
	Database    struct {
		Name          string `yaml:"name"`
		Host          string `yaml:"host"`
		Port          int    `yaml:"port"`
		User          string `yaml:"user"`
		Password      string `yaml:"password"`
		Database      string `yaml:"database"`
		MigrationPath string `yaml:"migration_path"`
	} `yaml:"database"`
	Auth struct {
		AccessSecKey  string `yaml:"access_sec_key"`
		RefreshSecKey string `yaml:"refresh_sec_key"`
	} `yaml:"auth"`
	Certs struct {
		StorageDir      string        `yaml:"storage_dir"`
		Email           string        `yaml:"email"`
		RenewalDuration time.Duration `yaml:"renuwal_duration"` // in hours
	} `yaml:"certs"`
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Logger struct {
		LogLevel string `yaml:"log_level"`
	} `yaml:"logger"`
}

func LoadConfig(confPath string) (*Config, error) {
	if confPath == "" {
		return nil, errors.New("config path is empty")
	}

	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		return nil, errors.New("config file does not exist")
	}

	var cfg Config

	if err := cleanenv.ReadConfig(confPath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
