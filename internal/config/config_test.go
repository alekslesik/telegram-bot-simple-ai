package config

import "testing"

func TestFromEnvReadsPostgresAndLLM(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://bot:bot@localhost:5432/bot?sslmode=disable")
	t.Setenv("LLM_PROVIDER", "openai_compatible")
	t.Setenv("LLM_BASE_URL", "https://api.deepseek.com")
	t.Setenv("LLM_MODEL", "deepseek-chat")

	cfg := FromEnv()

	if cfg.DatabaseURL == "" || cfg.LLMProvider == "" || cfg.LLMBaseURL == "" || cfg.LLMModel == "" {
		t.Fatal("expected config fields to be populated from env")
	}
}
