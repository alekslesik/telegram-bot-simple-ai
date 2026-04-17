package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/alekslesik/telegram-bot-simple/internal/bot"
	"github.com/alekslesik/telegram-bot-simple/internal/config"
	"github.com/alekslesik/telegram-bot-simple/internal/logging"
)

func TestFormatBuildDate_RFC3339(t *testing.T) {
	raw := time.Date(2024, 6, 10, 15, 30, 0, 0, time.UTC).Format(time.RFC3339)
	got := formatBuildDate(raw)
	// Europe/Moscow is UTC+3 in June (no DST).
	if want := "10/06/2024 18:30:00"; got != want {
		t.Fatalf("formatBuildDate(%q) = %q, want %q", raw, got, want)
	}
}

func TestFormatBuildDate_RFC3339Nano(t *testing.T) {
	raw := time.Date(2024, 1, 2, 3, 4, 5, 123456789, time.UTC).Format(time.RFC3339Nano)
	got := formatBuildDate(raw)
	if want := "02/01/2024 06:04:05"; got != want {
		t.Fatalf("formatBuildDate(nano) = %q, want %q", got, want)
	}
}

func TestFormatBuildDate_nonDatePassthrough(t *testing.T) {
	const s = "not-a-date"
	if formatBuildDate(s) != s {
		t.Fatalf("expected passthrough %q, got %q", s, formatBuildDate(s))
	}
}

func TestFormatBuildDate_loadLocationFails(t *testing.T) {
	orig := loadEuropeMoscow
	t.Cleanup(func() { loadEuropeMoscow = orig })
	loadEuropeMoscow = func() (*time.Location, error) {
		return nil, errors.New("no tz")
	}
	raw := time.Date(2024, 6, 10, 15, 30, 0, 0, time.UTC).Format(time.RFC3339)
	if got, want := formatBuildDate(raw), "10/06/2024 15:30:00"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSetMyCommandsConfig(t *testing.T) {
	cfg := setMyCommandsConfig()
	cmds := cfg.Commands
	if len(cmds) != 8 {
		t.Fatalf("expected 8 commands, got %d", len(cmds))
	}
	if cmds[0].Command != "start" {
		t.Fatalf("first command: %+v", cmds[0])
	}
	_ = cfg
}

func TestDeleteMyCommandsConfig(t *testing.T) {
	cfg := deleteMyCommandsConfig()
	if cfg.Scope != nil {
		t.Fatalf("expected nil scope for default delete config, got %#v", cfg.Scope)
	}
}

type stubTelegram struct {
	last tgbotapi.Chattable
}

func (s *stubTelegram) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	s.last = c
	return tgbotapi.Message{}, nil
}

func (s *stubTelegram) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	s.last = c
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func TestApplyTelegramUpdate_message(t *testing.T) {
	st := &stubTelegram{}
	h := bot.Handlers{
		Bot:    st,
		Logger: logging.NewWithWriter(&bytes.Buffer{}),
	}
	applyTelegramUpdate(&h, tgbotapi.Update{
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 1}, Text: "hi"},
	})
	if _, ok := st.last.(tgbotapi.MessageConfig); !ok {
		t.Fatalf("expected send, got %T", st.last)
	}
}

func TestApplyTelegramUpdate_callback(t *testing.T) {
	st := &stubTelegram{}
	h := bot.Handlers{
		Bot:    st,
		Logger: logging.NewWithWriter(&bytes.Buffer{}),
	}
	applyTelegramUpdate(&h, tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "x",
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 2}},
			Data:    "cmd:ping",
		},
	})
	if st.last == nil {
		t.Fatal("expected Request or Send")
	}
}

func TestApplyTelegramUpdate_empty(t *testing.T) {
	st := &stubTelegram{}
	h := bot.Handlers{Bot: st, Logger: logging.NewWithWriter(&bytes.Buffer{})}
	applyTelegramUpdate(&h, tgbotapi.Update{})
	if st.last != nil {
		t.Fatalf("expected no traffic, got %T", st.last)
	}
}

func TestUpdateKind(t *testing.T) {
	commandText := "/start"
	tests := []struct {
		name string
		u    tgbotapi.Update
		want string
	}{
		{
			name: "callback",
			u: tgbotapi.Update{
				CallbackQuery: &tgbotapi.CallbackQuery{ID: "cb"},
			},
			want: "callback_query",
		},
		{
			name: "command message",
			u: tgbotapi.Update{
				Message: &tgbotapi.Message{
					Text: commandText,
					Entities: []tgbotapi.MessageEntity{
						{Type: "bot_command", Offset: 0, Length: len(commandText)},
					},
				},
			},
			want: "command_message",
		},
		{
			name: "plain message",
			u: tgbotapi.Update{
				Message: &tgbotapi.Message{Text: "hello"},
			},
			want: "message",
		},
		{
			name: "other",
			u:    tgbotapi.Update{},
			want: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := updateKind(tt.u); got != tt.want {
				t.Fatalf("updateKind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLogAuthorized_withExpectedUsername(t *testing.T) {
	var buf bytes.Buffer
	logAuthorized(logging.NewWithWriter(&buf), "want", "got")
	if !bytes.Contains(buf.Bytes(), []byte("expected_username")) {
		t.Fatalf("log: %s", buf.String())
	}
}

func TestLogAuthorized_withoutExpectedUsername(t *testing.T) {
	var buf bytes.Buffer
	logAuthorized(logging.NewWithWriter(&buf), "", "only")
	if bytes.Contains(buf.Bytes(), []byte("expected_username")) {
		t.Fatalf("unexpected field: %s", buf.String())
	}
}

type errRegistrar struct{}

func (errRegistrar) Request(tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return nil, errors.New("boom")
}

func TestRegisterBotCommands_errorLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWithWriter(&buf)
	registerBotCommands(errRegistrar{}, logger)
	if !bytes.Contains(buf.Bytes(), []byte("failed to hide bot command menu")) {
		t.Fatalf("log: %s", buf.String())
	}
}

func TestRegisterBotCommands_ok(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWithWriter(&buf)
	st := &stubTelegram{}
	registerBotCommands(st, logger)
	if buf.Len() != 0 {
		t.Fatalf("expected no error log, got %s", buf.String())
	}
	if _, ok := st.last.(tgbotapi.DeleteMyCommandsConfig); !ok {
		t.Fatalf("expected delete-my-commands request, got %T", st.last)
	}
}

func TestTokenFromEnv(t *testing.T) {
	t.Setenv("TOKEN", "  abc  ")
	if got := tokenFromEnv(); got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildAIProvider_usesDefaultModelWhenMissing(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWithWriter(&buf)
	cfg := config.Config{
		LLMProvider: "openai_compatible",
		LLMBaseURL:  "https://api.openai.com/v1",
		LLMAPIKey:   "key",
		LLMModel:    "  ",
	}

	provider := buildAIProvider(cfg, logger)
	if provider == nil {
		t.Fatal("expected provider to be created with default model")
	}
	if !bytes.Contains(buf.Bytes(), []byte("llm model is not set; using default model")) {
		t.Fatalf("expected default-model log, got: %s", buf.String())
	}
}

func TestBuildAIProvider_requiresBaseURLAndAPIKey(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWithWriter(&buf)
	cfg := config.Config{
		LLMProvider: "openai_compatible",
		LLMModel:    "gpt-4o-mini",
	}

	provider := buildAIProvider(cfg, logger)
	if provider != nil {
		t.Fatal("expected nil provider when base url/api key are missing")
	}
	if !bytes.Contains(buf.Bytes(), []byte("llm is not fully configured; ai chat mode will be unavailable")) {
		t.Fatalf("expected missing-config log, got: %s", buf.String())
	}
}

func TestValidateRuntimeConfig_requiresDatabaseURL(t *testing.T) {
	cfg := config.Config{
		Token:       "token",
		DatabaseURL: "   ",
	}

	err := validateRuntimeConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got, want := err.Error(), "env var DATABASE_URL is not set (see .env)"; got != want {
		t.Fatalf("validateRuntimeConfig() error = %q, want %q", got, want)
	}
}

func TestOpenPostgresWithTimeout_appliesStartupDeadline(t *testing.T) {
	orig := openPostgres
	t.Cleanup(func() { openPostgres = orig })

	var (
		gotURL      string
		gotDeadline time.Time
		gotHasLimit bool
	)
	openPostgres = func(ctx context.Context, databaseURL string) (*sql.DB, error) {
		gotURL = databaseURL
		gotDeadline, gotHasLimit = ctx.Deadline()
		return &sql.DB{}, nil
	}

	start := time.Now()
	db, err := openPostgresWithTimeout(context.Background(), "postgres://bot:bot@localhost:5432/bot?sslmode=disable")
	if err != nil {
		t.Fatalf("openPostgresWithTimeout() error = %v", err)
	}
	if db == nil {
		t.Fatal("expected db")
	}

	if gotURL != "postgres://bot:bot@localhost:5432/bot?sslmode=disable" {
		t.Fatalf("openPostgresWithTimeout() url = %q", gotURL)
	}
	if !gotHasLimit {
		t.Fatal("expected startup timeout deadline")
	}
	if got := gotDeadline.Sub(start); got < startupPostgresConnectTimeout()-time.Second || got > startupPostgresConnectTimeout()+time.Second {
		t.Fatalf("deadline offset = %s, want about %s", got, startupPostgresConnectTimeout())
	}
}

func TestLongPollTimeoutSeconds(t *testing.T) {
	if longPollTimeoutSeconds() != 60 {
		t.Fatal("unexpected long poll timeout")
	}
}

func TestStartupRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 0, want: 3 * time.Second},
		{attempt: 1, want: 3 * time.Second},
		{attempt: 2, want: 6 * time.Second},
		{attempt: 5, want: 15 * time.Second},
	}
	for _, tt := range tests {
		if got := startupRetryDelay(tt.attempt); got != tt.want {
			t.Fatalf("startupRetryDelay(%d)=%s, want %s", tt.attempt, got, tt.want)
		}
	}
}

func TestTelegramAPIAddr(t *testing.T) {
	if got, want := telegramAPIAddr(), "api.telegram.org:443"; got != want {
		t.Fatalf("telegramAPIAddr()=%q, want %q", got, want)
	}
}
