package config

import (
	"errors"
	"fmt"
	"github.com/joeshaw/envdecode"
	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"io/fs"
	"time"
)

type Config struct {
	Server   ServerConfig
	Accrual  AccrualConfig
	Database DatabaseConfig

	SecretKey  string `env:"APP_SECRET_KEY,default=ChangeMe"`
	LogVerbose bool   `env:"APP_VERBOSE,default=0"`
	LogPretty  bool   `env:"APP_PRETTY,default=0"`
}

type ServerConfig struct {
	Listen       string        `env:"RUN_ADDRESS,default=localhost:8088"`
	TimeoutRead  time.Duration `env:"SERVER_TIMEOUT_READ,default=5s"`
	TimeoutWrite time.Duration `env:"SERVER_TIMEOUT_WRITE,default=10s"`
	TimeoutIdle  time.Duration `env:"SERVER_TIMEOUT_IDLE,default=1m"`
}

type DatabaseConfig struct {
	DSN string `env:"DATABASE_URI,required"`
}

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR,default=localhost:6379"`
	Password string `env:"REDIS_PASSWORD,default="`
	DB       int    `env:"REDIS_DB,default=0"`
}

type AccrualConfig struct {
	RemoteURL string `env:"ACCRUAL_SYSTEM_ADDRESS,required"`
}

// New config constructor
func New() Config {
	return Config{}
}

// Load config from environment and from .env file (if exists) and from flags
func (cfg *Config) Load() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf(".env load: %w", err)
	}

	if err := envdecode.StrictDecode(cfg); err != nil {
		return fmt.Errorf("env decode: %w", err)
	}

	pflag.StringVarP(&cfg.Server.Listen, "listen-addr", "a", cfg.Server.Listen, "Server address to listen on")
	pflag.StringVarP(&cfg.Database.DSN, "database-uri", "d", cfg.Database.DSN, "Database URI")
	pflag.StringVarP(&cfg.Accrual.RemoteURL, "accrual-url", "r", cfg.Accrual.RemoteURL, "Accrual base URL")
	pflag.BoolVarP(&cfg.LogVerbose, "verbose", "v", cfg.LogVerbose, "Verbose output")
	pflag.BoolVarP(&cfg.LogPretty, "pretty", "p", cfg.LogPretty, "Pretty output")
	pflag.Parse()

	return nil
}
