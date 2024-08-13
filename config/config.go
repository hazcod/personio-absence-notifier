package config

import (
	"fmt"
	validator "github.com/asaskevich/govalidator"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	defaultLogLevel = "DEBUG"
)

type Config struct {
	Log struct {
		Level string `yaml:"level" env:"LOG_LEVEL" minLength:"1"`
	} `yaml:"log"`

	Personio struct {
		ClientID string `yaml:"client_id" env:"PERSONIO_CLIENT_ID" minLength:"1"`
		Secret   string `yaml:"secret" env:"PERSONIO_SECRET" minLength:"1"`
	} `yaml:"personio"`

	Slack struct {
		WebhookURL string `yaml:"webhook_url" env:"SLACK_WEBHOOK_URL" minLength:"1"`
	} `yaml:"slack"`
}

func (c *Config) Validate() error {
	if c.Log.Level == "" {
		c.Log.Level = defaultLogLevel
	}

	if valid, err := validator.ValidateStruct(c); !valid || err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	return nil
}

func (c *Config) Load(path string) error {
	if path != "" {
		configBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to load configuration file at '%s': %v", path, err)
		}

		if err = yaml.Unmarshal(configBytes, c); err != nil {
			return fmt.Errorf("failed to parse configuration: %v", err)
		}
	}

	if err := envconfig.Process("", c); err != nil {
		return fmt.Errorf("could not load environment: %v", err)
	}

	return nil
}
