package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type LeagueYAML struct {
	ID    int               `yaml:"id"`
	Min   int               `yaml:"min"`
	Max   int               `yaml:"max"`
	Names map[string]string `yaml:"names"`
}

type Config struct {
	RedisAddr string       `yaml:"redis_addr"`
	HTTPAddr  string       `yaml:"http_addr"`
	Leagues   []LeagueYAML `yaml:"leagues"`
}

func Load(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.RedisAddr == "" {
		cfg.RedisAddr = "localhost:6379"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	return &cfg, nil
}
