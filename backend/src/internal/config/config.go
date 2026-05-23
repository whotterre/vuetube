package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port string `mapstructure:"PORT"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
}

func LoadConfig() (*Config, error) {
	viper.AutomaticEnv()

	port := viper.GetString("PORT")
	if port == "" {
		port = ":8000"
	} else if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	dbURL := viper.GetString("DATABASE_URL")

	return &Config{
		Port: port,
		DatabaseURL: dbURL,
	}, nil
}
