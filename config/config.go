package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	// Config -.
	Config struct {
		App    `yaml:"app"`
		HTTP   `yaml:"http"`
		Log    `yaml:"logger"`
		PG     `yaml:"postgres"`
		Minio  `yaml:"minio"`
		ApiKey `yaml:"api_key"`
		JWT    `yaml:"jwt"`
	}

	// App -.
	App struct {
		Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
		Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
	}

	// HTTP -.
	HTTP struct {
		Port string `env-required:"true" yaml:"port" env:"HTTP_PORT"`
	}

	// Log -.
	Log struct {
		Level string `env-required:"true" yaml:"log_level"   env:"LOG_LEVEL"`
	}

	// PG -.
	PG struct {
		PoolMax int    `env-required:"true" yaml:"pool_max" env:"PG_POOL_MAX"`
		URL     string `env-required:"true"                 env:"PG_URL"`
	}

	// Minio -.
	Minio struct {
		MINIO_ENDPOINT    string `env-required:"true" yaml:"MINIO_ENDPOINT" env:"MINIO_ENDPOINT"`
		MINIO_ACCESS_KEY  string `env-required:"true" yaml:"MINIO_ACCESS_KEY" env:"MINIO_ACCESS_KEY"`
		MINIO_SECRET_KEY  string `env-required:"true" yaml:"MINIO_SECRET_KEY" env:"MINIO_SECRET_KEY"`
		MINIO_BUCKET_NAME string `env-required:"true" yaml:"MINIO_BUCKET_NAME" env:"MINIO_BUCKET_NAME"`
	}

	// ApiKey -.
	ApiKey struct {
		Key string `env-required:"true" yaml:"key" env:"API_KEY"`
	}

	// JWT -.
	JWT struct {
		Secret string `env-required:"true" yaml:"secret" env:"JWT_SECRET"`
	}

)

// NewConfig returns app config.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig("./config/config.yml", cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
