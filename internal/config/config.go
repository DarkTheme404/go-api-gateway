package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Backend struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

type Config struct {
	Server struct {
		ReadTimeout  int `yaml:"read_timeout"`
		WriteTimeout int `yaml:"write_timeout"`
	} `yaml:"server"`

	ListenAddr string `yaml:"listen_addr"`

	Backends []Backend `yaml:"backends"`

	JWT struct {
		Secret     string `yaml:"secret"`
		Issuer     string `yaml:"issuer"`
		AllowedAlg []string `yaml:"allowed_algorithms"`
	} `yaml:"jwt"`

	RateLimit struct {
		RPS   float64 `yaml:"rps"`
		Burst int     `yaml:"burst"`
	} `yaml:"rate_limit"`

	CircuitBreaker struct {
		Threshold int    `yaml:"threshold"`
		Timeout   int    `yaml:"timeout"`
	} `yaml:"circuit_breaker"`

	Metrics struct {
		Enabled bool   `yaml:"enabled"`
		Path    string `yaml:"path"`
	} `yaml:"metrics"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.ListenAddr = ":8080"
	cfg.Server.ReadTimeout = 30
	cfg.Server.WriteTimeout = 30
	cfg.RateLimit.RPS = 100
	cfg.RateLimit.Burst = 200
	cfg.CircuitBreaker.Threshold = 5
	cfg.CircuitBreaker.Timeout = 30
	cfg.Metrics.Enabled = true
	cfg.Metrics.Path = "/metrics"
	return cfg
}
