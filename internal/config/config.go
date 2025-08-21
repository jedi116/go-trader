package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Broker   BrokerConfig   `mapstructure:"broker"`
	Brave    BraveConfig    `mapstructure:"brave"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

type BrokerConfig struct {
	OANDA struct {
		APIKey    string `mapstructure:"api_key"`
		AccountID string `mapstructure:"account_id"`
		BaseURL   string `mapstructure:"base_url"`
	} `mapstructure:"oanda"`
}

type BraveConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

func Load() (*Config, error) {
	// Load .env if it exists (silent fail)
	_ = godotenv.Load()
	configData, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	// Expand environment variables (${VAR} syntax)
	expandedData := os.ExpandEnv(string(configData))

	// Parse the expanded YAML with Viper
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(expandedData)); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
