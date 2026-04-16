package config

import (
	"os"
	"strings"
)

type Config struct {
	Token       string
	Username    string
	DatabaseURL string
	LLMProvider string
	LLMBaseURL  string
	LLMAPIKey   string
	LLMModel    string
}

func FromEnv() Config {
	return Config{
		Token:       trimmedEnv("TOKEN"),
		Username:    trimmedEnv("USERNAME"),
		DatabaseURL: trimmedEnv("DATABASE_URL"),
		LLMProvider: trimmedEnv("LLM_PROVIDER"),
		LLMBaseURL:  trimmedEnv("LLM_BASE_URL"),
		LLMAPIKey:   trimmedEnv("LLM_API_KEY"),
		LLMModel:    trimmedEnv("LLM_MODEL"),
	}
}

func trimmedEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
