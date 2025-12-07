package config

import "time"

var initialisedConfig *Config

type Config struct {
	NewListingsPollingInterval       time.Duration
	ReanalyseListingsPollingInterval time.Duration
}

func InitialiseConfig() {
	initialisedConfig = &Config{
		NewListingsPollingInterval:       time.Minute * 5,
		ReanalyseListingsPollingInterval: time.Hour * 60,
	}
}

func GetConfig() *Config {
	return initialisedConfig
}
