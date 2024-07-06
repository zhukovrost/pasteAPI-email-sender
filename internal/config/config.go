package config

import (
	"flag"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	RabbitMQ struct {
		URL          string `yaml:"url" envconfig:"RABBITMQ_URL"`
		Timeout      string `yaml:"timeout" envconfig:"RABBITMQ_TIMEOUT"`
		Exchange     string `yaml:"exchange" envconfig:"RABBITMQ_EXCHANGE"`
		ExchangeType string `yaml:"exchange_type" envconfig:"RABBITMQ_EXCHANGE_TYPE"`
		Queue        string `yaml:"queue" envconfig:"RABBITMQ_QUEUE"`
		ConsumerTag  string `yaml:"consumer_tag" envconfig:"RABBITMQ_CONSUMER_TAG"`
	} `yaml:"rabbitmq"`
	Logger struct {
		NeedDebug bool `yaml:"need_debug" envconfig:"API_LOGGER_DEBUG"`
	} `yaml:"logger"`
	SMTP struct {
		Host     string `yaml:"host" envconfig:"PASTE_SMTP_HOST"`
		Port     int    `yaml:"port" envconfig:"PASTE_SMTP_PORT"`
		Username string `yaml:"user" envconfig:"PASTE_SMTP_USER"`
		Password string `yaml:"password" envconfig:"PASTE_SMTP_PASSWORD"`
		Sender   string `yaml:"sender" envconfig:"PASTE_SMTP_SENDER"`
		Timeout  string `yaml:"timeout" envconfig:"PASTE_SMTP_TIMEOUT"`
	} `yaml:"smtp"`
}

func New() (*Config, error) {
	var cfg Config

	if err := loadConfig("configs/config.yml", &cfg); err != nil {
		return nil, err
	}
	if err := processEnvironment(&cfg); err != nil {
		return nil, err
	}

	flag.BoolVar(&cfg.Logger.NeedDebug, "debug", cfg.Logger.NeedDebug, "turns on debug level (log)")
	flag.Parse()

	return &cfg, nil
}

func loadConfig(filename string, cfg *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config file %s: %w", filename, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to decode YAML from config file %s: %w", filename, err)
	}

	return nil
}

func processEnvironment(cfg *Config) error {
	if err := envconfig.Process("", cfg); err != nil {
		return fmt.Errorf("failed to process environment variables: %w", err)
	}
	return nil
}
