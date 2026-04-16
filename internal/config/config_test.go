package config

import (
	"testing"
	"time"
)

func TestFromEnvReadsPostgresAndLLM(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://bot:bot@localhost:5432/bot?sslmode=disable")
	t.Setenv("LLM_PROVIDER", "openai_compatible")
	t.Setenv("LLM_BASE_URL", "https://api.deepseek.com")
	t.Setenv("LLM_MODEL", "deepseek-chat")
	t.Setenv("LLM_TIMEOUT_SEC", "45")

	cfg := FromEnv()

	if cfg.DatabaseURL == "" || cfg.LLMProvider == "" || cfg.LLMBaseURL == "" || cfg.LLMModel == "" {
		t.Fatal("expected config fields to be populated from env")
	}
	if cfg.LLMTimeout != 45*time.Second {
		t.Fatalf("expected llm timeout to be parsed, got %s", cfg.LLMTimeout)
	}
}
