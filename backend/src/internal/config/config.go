package config

import (
	"errors"
	"log"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	Port        string `mapstructure:"PORT"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	JWTSecret   string `mapstructure:"JWT_SECRET"`
}

func LoadConfig() (*Config, error) {
	// Try several likely .env file locations relative to the process cwd so
	// the binary can be executed from different working directories.
	// TODO: Remove this shitshow
	tryPaths := []string{
		"src/.env",
		".env",
		"../.env",
		"../../.env",
		"../src/.env",
		"../../src/.env",
		"../../../.env",
	}

	var loaded string
	for _, p := range tryPaths {
		if err := gotenv.Load(p); err == nil {
			loaded = p
			break
		}
	}

	viper.AutomaticEnv()
	viper.AddConfigPath("../../")

	if viper.GetString("DATABASE_URL") == "" {
		log.Println("config: DATABASE_URL not set in environment or .env")
	} else {
		if loaded != "" {
			log.Printf("config: loaded env from %s", loaded)
		}
	}

	port := viper.GetString("PORT")
	if port == "" {
		port = ":8000"
	} else if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	dbURL := viper.GetString("DATABASE_URL")

	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required but not set in environment or .env")
	}

	jwtSecret := viper.GetString("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
		log.Println("config: JWT_SECRET not set, using default dev secret")
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}, nil
}
