package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/alekslesik/telegram-bot-simple/internal/bot"
	"github.com/alekslesik/telegram-bot-simple/internal/logging"
	"github.com/alekslesik/telegram-bot-simple/internal/telegram"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// loadEuropeMoscow is swappable in tests to cover the LoadLocation error path.
var loadEuropeMoscow = func() (*time.Location, error) {
	return time.LoadLocation("Europe/Moscow")
}

// formatBuildDate turns an RFC3339 / RFC3339Nano build timestamp into log display format (Europe/Moscow).
func formatBuildDate(raw string) string {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		if loc, locErr := loadEuropeMoscow(); locErr == nil {
			t = t.In(loc)
		}
		return t.Format("02/01/2006 15:04:05")
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		if loc, locErr := loadEuropeMoscow(); locErr == nil {
			t = t.In(loc)
		}
		return t.Format("02/01/2006 15:04:05")
	}
	return raw
}

func applyTelegramUpdate(h *bot.Handlers, u tgbotapi.Update) {
	if u.CallbackQuery != nil {
		h.HandleCallback(u.CallbackQuery)
		return
	}
	if u.Message == nil {
		return
	}
	h.HandleMessage(u.Message)
}

func updateKind(u tgbotapi.Update) string {
	switch {
	case u.CallbackQuery != nil:
		return "callback_query"
	case u.Message != nil && u.Message.IsCommand():
		return "command_message"
	case u.Message != nil:
		return "message"
	default:
		return "other"
	}
}

func probeTelegramAPI(tg *tgbotapi.BotAPI, logger slogLogger, reason string) {
	me, err := tg.GetMe()
	if err != nil {
		logger.Error("telegram api probe failed", "reason", reason, "err", err)
		return
	}
	logger.Info("telegram api probe ok", "reason", reason, "bot_id", me.ID, "username", me.UserName)
}

func telegramAPIAddr() string {
	return "api.telegram.org:443"
}

func startupRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(attempt) * 3 * time.Second
}

func probeTelegramNetwork(logger slogLogger, reason string) {
	host, _, err := net.SplitHostPort(telegramAPIAddr())
	if err != nil {
		logger.Error("invalid telegram api addr", "addr", telegramAPIAddr(), "err", err)
		return
	}
	ips, lookupErr := net.LookupHost(host)
	if lookupErr != nil {
		logger.Error("telegram dns probe failed", "reason", reason, "host", host, "err", lookupErr)
		return
	}
	conn, dialErr := net.DialTimeout("tcp", telegramAPIAddr(), 5*time.Second)
	if dialErr != nil {
		logger.Error("telegram tcp probe failed",
			"reason", reason,
			"addr", telegramAPIAddr(),
			"resolved_ips", strings.Join(ips, ","),
			"err", dialErr,
		)
		return
	}
	_ = conn.Close()
	logger.Info("telegram network probe ok",
		"reason", reason,
		"addr", telegramAPIAddr(),
		"resolved_ips", strings.Join(ips, ","),
	)
}

func createBotWithRetry(token string, logger slogLogger) (*tgbotapi.BotAPI, error) {
	const maxAttempts = 5
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		probeTelegramNetwork(logger, fmt.Sprintf("create_bot_attempt_%d", attempt))
		tg, err := telegram.New(token)
		if err == nil {
			if attempt > 1 {
				logger.Info("telegram client created after retry", "attempt", attempt)
			}
			return tg, nil
		}
		lastErr = err
		logger.Error("failed to create telegram client", "attempt", attempt, "max_attempts", maxAttempts, "err", err)
		if attempt < maxAttempts {
			time.Sleep(startupRetryDelay(attempt))
		}
	}
	return nil, fmt.Errorf("failed to create bot after %d attempts: %w", maxAttempts, lastErr)
}

func logAuthorized(logger slogLogger, username, botUsername string) {
	if username != "" {
		logger.Info("authorized",
			"username", botUsername,
			"expected_username", username,
		)
	} else {
		logger.Info("authorized",
			"username", botUsername,
		)
	}
}

// slogLogger is the subset of *slog.Logger used by main (tests pass a concrete *slog.Logger).
type slogLogger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
}

type commandRegistrar interface {
	Request(tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

func registerBotCommands(reg commandRegistrar, logger slogLogger) {
	if _, err := reg.Request(setMyCommandsConfig()); err != nil {
		logger.Error("failed to register bot commands", "err", err)
	}
}

func tokenFromEnv() string {
	return strings.TrimSpace(os.Getenv("TOKEN"))
}

func longPollTimeoutSeconds() int {
	return 60
}

func setMyCommandsConfig() tgbotapi.SetMyCommandsConfig {
	return tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "🚀 Старт"},
		tgbotapi.BotCommand{Command: "menu", Description: "📋 Демо-меню"},
		tgbotapi.BotCommand{Command: "help", Description: "📋 Меню команд"},
		tgbotapi.BotCommand{Command: "about", Description: "ℹ️ О боте"},
		tgbotapi.BotCommand{Command: "usecases", Description: "💼 Примеры задач"},
		tgbotapi.BotCommand{Command: "features", Description: "🧩 Возможности"},
		tgbotapi.BotCommand{Command: "ping", Description: "✅ Проверка статуса"},
		tgbotapi.BotCommand{Command: "echo", Description: "🗣️ Повторить текст"},
	)
}

func main() {
	logger := logging.NewFromEnv()

	buildDate := formatBuildDate(BuildDate)

	logger.Info("starting",
		"version", Version,
		"commit", Commit,
		"build_date", buildDate,
	)

	token := tokenFromEnv()
	if token == "" {
		log.Fatal("env var TOKEN is not set (see .env)")
	}

	username := os.Getenv("USERNAME")

	tg, err := createBotWithRetry(token, logger)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	logAuthorized(logger, username, tg.Self.UserName)
	probeTelegramAPI(tg, logger, "startup")

	registerBotCommands(tg, logger)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = longPollTimeoutSeconds()

	updates := tg.GetUpdatesChan(u)
	probeTicker := time.NewTicker(2 * time.Minute)
	defer probeTicker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	h := bot.Handlers{
		Bot:    tg,
		Logger: logger,
	}

	logger.Info("bot started with long polling, press Ctrl+C to stop")

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				logger.Error("updates channel closed; stopping bot loop")
				return
			}
			logger.Debug("received telegram update",
				"update_id", update.UpdateID,
				"kind", updateKind(update),
				"has_message", update.Message != nil,
				"has_callback", update.CallbackQuery != nil,
			)
			applyTelegramUpdate(&h, update)

		case <-probeTicker.C:
			probeTelegramAPI(tg, logger, "periodic")

		case sig := <-stop:
			logger.Info("received signal, shutting down", "signal", sig.String())
			return
		}
	}
}
