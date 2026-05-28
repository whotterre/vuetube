package config

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	Port               string `mapstructure:"PORT"`
	DatabaseURL        string `mapstructure:"DATABASE_URL"`
	JWTSecret          string `mapstructure:"JWT_SECRET"`
	AWSAccessKey       string `mapstructure:"AWS_ACCESS_KEY_ID"`
	AWSSecretAccessKey string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	AWSRegion          string `mapstructure:"AWS_REGION"`
	BucketName         string `mapstructure:"BUCKET_NAME"`
	RedisAddr          string `mapstructure:"REDIS_ADDR"`
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
		log.Println("config: JWT_SECRET not set, using default dev secret")
		return nil, errors.New("JWT_SECRET not set")
	}



	redisAddr := viper.GetString("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	return &Config{
		Port:               port,
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		AWSAccessKey:       viper.GetString("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: viper.GetString("AWS_SECRET_ACCESS_KEY"),
		BucketName:         viper.GetString("BUCKET_NAME"),
		AWSRegion:          viper.GetString("AWS_REGION"),
		RedisAddr:          redisAddr,
	}, nil
}

func LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	region := viper.GetString("AWS_REGION")
	if region != "" {
		return awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	}
	return awscfg.LoadDefaultConfig(ctx)
}
