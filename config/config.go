package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

const TXSLIMIT = 13

type Config struct {
	Node struct {
		Port           int    `yaml:"port"`
		PeersPath      string `yaml:"peers"`
		PrivateKeyPath string `yaml:"privateKey"`
		Database       string `yaml:"database"`
		Control        struct {
			Port int `yaml:"port"`
		} `yaml:"control"`
	} `yaml:"node"`
}

func New() (*Config, error) {
	file, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	cfg := new(Config)

	if err := yaml.Unmarshal(file, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}