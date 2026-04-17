package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Token                 string
	Username              string
	DatabaseURL           string
	LLMProvider           string
	LLMBaseURL            string
	LLMAPIKey             string
	LLMModel              string
	LLMTimeout            time.Duration
	AIChatDailyLimit      int
	AIChatHistoryMessages int
	AIChatThinkingDelay   time.Duration
}

func FromEnv() Config {
	return Config{
		Token:                 trimmedEnv("TOKEN"),
		Username:              trimmedEnv("USERNAME"),
		DatabaseURL:           trimmedEnv("DATABASE_URL"),
		LLMProvider:           trimmedEnv("LLM_PROVIDER"),
		LLMBaseURL:            trimmedEnv("LLM_BASE_URL"),
		LLMAPIKey:             trimmedEnv("LLM_API_KEY"),
		LLMModel:              trimmedEnv("LLM_MODEL"),
		LLMTimeout:            envDurationSeconds("LLM_TIMEOUT_SEC"),
		AIChatDailyLimit:      envInt("AI_CHAT_DAILY_LIMIT"),
		AIChatHistoryMessages: envInt("AI_CHAT_HISTORY_MESSAGES"),
		AIChatThinkingDelay:   envDurationMilliseconds("AI_CHAT_THINKING_DELAY_MS"),
	}
}

func trimmedEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func envDurationSeconds(key string) time.Duration {
	value := trimmedEnv(key)
	if value == "" {
		return 0
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

func envInt(key string) int {
	value := trimmedEnv(key)
	if value == "" {
		return 0
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return parsed
}

func envDurationMilliseconds(key string) time.Duration {
	value := trimmedEnv(key)
	if value == "" {
		return 0
	}

	ms, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return time.Duration(ms) * time.Millisecond
}
