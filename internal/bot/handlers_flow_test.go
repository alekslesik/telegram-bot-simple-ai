package bot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
	"github.com/alekslesik/telegram-bot-simple/internal/repository"
	contentservice "github.com/alekslesik/telegram-bot-simple/internal/service/content"
	"github.com/alekslesik/telegram-bot-simple/internal/service/learningflow"
	llmservice "github.com/alekslesik/telegram-bot-simple/internal/service/llm"
)

type fakeFlowTelegram struct {
	sent     []tgbotapi.Chattable
	requests []tgbotapi.Chattable
}

func (f *fakeFlowTelegram) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	f.sent = append(f.sent, c)
	return tgbotapi.Message{}, nil
}

func (f *fakeFlowTelegram) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	f.requests = append(f.requests, c)
	return &tgbotapi.APIResponse{Ok: true}, nil
}

type fakeFlowContentService struct {
	payloads map[int64]contentservice.BlockPayload
}

func (f fakeFlowContentService) BuildBlockPayload(_ context.Context, blockID int64) (contentservice.BlockPayload, error) {
	return f.payloads[blockID], nil
}

type fakeFlowLearningService struct {
	next learning.FlowStep
}

func (f fakeFlowLearningService) NextStep(_ learning.BlockType, _ learning.FlowStep, _ learningflow.Action) learning.FlowStep {
	return f.next
}

type fakeFlowContentRepository struct {
	chapters      map[int64]learning.Chapter
	blocks        map[int64]learning.Block
	chapterBlocks map[int64][]learning.Block
}

func (f fakeFlowContentRepository) GetSectionByCode(context.Context, string) (learning.Section, error) {
	return learning.Section{}, nil
}

func (f fakeFlowContentRepository) GetChapter(_ context.Context, chapterID int64) (learning.Chapter, error) {
	if chapter, ok := f.chapters[chapterID]; ok {
		return chapter, nil
	}
	return learning.Chapter{}, nil
}

func (f fakeFlowContentRepository) ListChaptersBySection(context.Context, int64) ([]learning.Chapter, error) {
	return nil, nil
}

func (f fakeFlowContentRepository) ListChapterBlocks(_ context.Context, chapterID int64) ([]learning.Block, error) {
	return f.chapterBlocks[chapterID], nil
}

func (f fakeFlowContentRepository) GetBlock(_ context.Context, blockID int64) (learning.Block, error) {
	return f.blocks[blockID], nil
}

func (f fakeFlowContentRepository) GetBlockContent(context.Context, int64) (learning.BlockContent, error) {
	return learning.BlockContent{}, nil
}

func (f fakeFlowContentRepository) ListBlockRelations(context.Context, int64) ([]learning.BlockRelation, error) {
	return nil, nil
}

var _ repository.ContentRepository = fakeFlowContentRepository{}

func newFlowTestHandlers(
	bot TelegramClient,
	repo repository.ContentRepository,
	contentSvc flowContentBuilder,
	learningSvc flowNavigator,
	aiSvc aiChatProvider,
) *Handlers {
	return &Handlers{
		Bot:      bot,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})),
		Content:  contentSvc,
		Learning: learningSvc,
		AI:       aiSvc,
		Repo:     repo,
		State:    newInMemoryFlowStateStore(),
	}
}

type fakeAIProvider struct {
	reply     string
	err       error
	lastInput llmservice.ChatInput
	delay     time.Duration
}

func (f *fakeAIProvider) Chat(ctx context.Context, input llmservice.ChatInput) (string, error) {
	f.lastInput = input
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	if f.err != nil {
		return "", f.err
	}
	return f.reply, nil
}

func TestHandlers_HandleCallback_FlowSkipAnswerShowsSolution(t *testing.T) {
	bot := &fakeFlowTelegram{}
	repo := fakeFlowContentRepository{
		blocks: map[int64]learning.Block{
			42: {ID: 42, ChapterID: 7, BlockType: learning.BlockTask, Title: "Task"},
			43: {ID: 43, ChapterID: 7, BlockType: learning.BlockSolution, Title: "Solution"},
		},
		chapterBlocks: map[int64][]learning.Block{
			7: {
				{ID: 42, ChapterID: 7, BlockType: learning.BlockTask, Title: "Task"},
				{ID: 43, ChapterID: 7, BlockType: learning.BlockSolution, Title: "Solution"},
			},
		},
	}
	contentSvc := fakeFlowContentService{
		payloads: map[int64]contentservice.BlockPayload{
			43: {
				Title:      "Solution",
				BlockType:  learning.BlockSolution,
				SolutionMD: "SolutionMD: use two pointers",
			},
		},
	}

	h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepSolution}, nil)

	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID:   "skip-answer",
		From: &tgbotapi.User{ID: 1001},
		Data: "flow:skip_answer:42",
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 10},
		},
	})

	if len(bot.sent) == 0 {
		t.Fatal("expected a reply message")
	}

	cfg, ok := bot.sent[len(bot.sent)-1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[len(bot.sent)-1])
	}
	if !strings.Contains(cfg.Text, "SolutionMD: use two pointers") {
		t.Fatalf("expected solution payload text, got %q", cfg.Text)
	}
}

func TestHandlers_HandleCallback_FlowNextFromTheoryShowsTask(t *testing.T) {
	bot := &fakeFlowTelegram{}
	repo := fakeFlowContentRepository{
		blocks: map[int64]learning.Block{
			10: {ID: 10, ChapterID: 3, BlockType: learning.BlockTheory, Title: "Theory"},
			11: {ID: 11, ChapterID: 3, BlockType: learning.BlockTask, Title: "Task"},
		},
		chapterBlocks: map[int64][]learning.Block{
			3: {
				{ID: 10, ChapterID: 3, BlockType: learning.BlockTheory, Title: "Theory"},
				{ID: 11, ChapterID: 3, BlockType: learning.BlockTask, Title: "Task"},
			},
		},
	}
	contentSvc := fakeFlowContentService{
		payloads: map[int64]contentservice.BlockPayload{
			11: {
				Title:     "Task",
				BlockType: learning.BlockTask,
				TaskMD:    "TaskMD: solve the example",
			},
		},
	}

	h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepTask}, nil)

	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID:   "next-theory",
		From: &tgbotapi.User{ID: 2002},
		Data: "flow:next:10",
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 11},
		},
	})

	if len(bot.sent) == 0 {
		t.Fatal("expected a reply message")
	}

	cfg, ok := bot.sent[len(bot.sent)-1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[len(bot.sent)-1])
	}
	if !strings.Contains(cfg.Text, "TaskMD: solve the example") {
		t.Fatalf("expected task payload text, got %q", cfg.Text)
	}
}

func TestHandlers_RenderBlockStep_LogsLearningFlowStep(t *testing.T) {
	var logs bytes.Buffer

	bot := &fakeFlowTelegram{}
	repo := fakeFlowContentRepository{
		chapters: map[int64]learning.Chapter{
			3: {ID: 3, Code: "two-pointers", Title: "Two Pointers"},
		},
	}
	contentSvc := fakeFlowContentService{
		payloads: map[int64]contentservice.BlockPayload{
			11: {
				Title:     "Task",
				BlockType: learning.BlockTask,
				TaskMD:    "TaskMD: solve the example",
			},
		},
	}

	h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepTask}, nil)
	h.Logger = slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{}))

	err := h.renderBlockStep(context.Background(), 11, 2002, learning.Block{
		ID:        11,
		ChapterID: 3,
		BlockType: learning.BlockTask,
		Title:     "Task",
	}, learning.StepTask)
	if err != nil {
		t.Fatalf("renderBlockStep returned error: %v", err)
	}

	got := logs.String()
	for _, want := range []string{
		`"msg":"learning flow step"`,
		`"user_id":2002`,
		`"chapter":"two-pointers"`,
		`"block_id":11`,
		`"step":"task"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected logs to contain %s, got %q", want, got)
		}
	}
}

func TestCommandKeyboard_LearningMenuLabelsResolveThroughMap(t *testing.T) {
	kb := commandKeyboard()
	seen := make(map[string]struct{})
	for _, row := range kb.Keyboard {
		for _, btn := range row {
			seen[btn.Text] = struct{}{}
		}
	}

	required := []string{"Введение", "Алгоритмы по порядку", "Рандом задача", "Мой прогресс", "Настройки", "🤖 ИИ-чат (тест)"}
	for _, label := range required {
		if _, ok := seen[label]; !ok {
			t.Fatalf("expected reply keyboard to contain %q", label)
		}
		if _, ok := learningMenuButtons[label]; !ok {
			t.Fatalf("expected %q to exist in learningMenuButtons", label)
		}
	}
}

type captureSectionRepo struct {
	lastSectionCode string
}

func (c *captureSectionRepo) GetSectionByCode(_ context.Context, code string) (learning.Section, error) {
	c.lastSectionCode = code
	return learning.Section{ID: 1}, nil
}

func (c *captureSectionRepo) GetChapter(context.Context, int64) (learning.Chapter, error) {
	return learning.Chapter{}, errors.New("not implemented")
}

func (c *captureSectionRepo) ListChaptersBySection(context.Context, int64) ([]learning.Chapter, error) {
	return []learning.Chapter{{ID: 10, SectionID: 1, Code: "ch", Title: "ch", SortOrder: 1}}, nil
}

func (c *captureSectionRepo) ListChapterBlocks(context.Context, int64) ([]learning.Block, error) {
	return []learning.Block{{ID: 100, ChapterID: 10, BlockType: learning.BlockTheory, Title: "t", Code: "b"}}, nil
}

func (c *captureSectionRepo) GetBlock(context.Context, int64) (learning.Block, error) {
	return learning.Block{}, errors.New("not implemented")
}

func (c *captureSectionRepo) GetBlockContent(context.Context, int64) (learning.BlockContent, error) {
	return learning.BlockContent{}, errors.New("not implemented")
}

func (c *captureSectionRepo) ListBlockRelations(context.Context, int64) ([]learning.BlockRelation, error) {
	return nil, nil
}

var _ repository.ContentRepository = (*captureSectionRepo)(nil)

func TestLearningMenuButtonsRouteToExpectedSectionCodes(t *testing.T) {
	tests := []struct {
		label           string
		expectedSection string
	}{
		{label: "Введение", expectedSection: "introduction"},
		{label: "Алгоритмы по порядку", expectedSection: "algorithms"},
	}

	for _, tt := range tests {
		bot := &fakeFlowTelegram{}
		repo := &captureSectionRepo{}
		contentSvc := fakeFlowContentService{
			payloads: map[int64]contentservice.BlockPayload{
				100: {
					Title:     "t",
					BlockType: learning.BlockTheory,
					TheoryMD:  "TheoryMD",
				},
			},
		}

		h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepTask}, nil)

		msg := &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 10},
			From: &tgbotapi.User{ID: 999},
		}
		h.handleLearningMenuSelection(msg, learningMenuButtons[tt.label])

		if repo.lastSectionCode != tt.expectedSection {
			t.Fatalf("for label %q expected section %q, got %q", tt.label, tt.expectedSection, repo.lastSectionCode)
		}
	}
}

func TestHandlers_HandleCallback_UnknownFlowActionKeepsStateAndSendsInstruction(t *testing.T) {
	bot := &fakeFlowTelegram{}
	repo := fakeFlowContentRepository{
		blocks: map[int64]learning.Block{
			55: {ID: 55, ChapterID: 8, BlockType: learning.BlockTask, Title: "Task"},
		},
		chapterBlocks: map[int64][]learning.Block{
			8: {
				{ID: 55, ChapterID: 8, BlockType: learning.BlockTask, Title: "Task"},
			},
		},
	}

	h := newFlowTestHandlers(bot, repo, fakeFlowContentService{}, fakeFlowLearningService{next: learning.StepSolution}, nil)
	userID := int64(3003)
	h.State.Save(userID, flowState{
		BlockID:    55,
		ChapterID:  8,
		Step:       learning.StepReview,
		LastAnswer: "my previous answer",
	})

	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID:   "unknown-action",
		From: &tgbotapi.User{ID: userID},
		Data: "flow:unknown:55",
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 12},
		},
	})

	if len(bot.sent) == 0 {
		t.Fatal("expected instructional reply")
	}

	cfg, ok := bot.sent[len(bot.sent)-1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[len(bot.sent)-1])
	}
	if !strings.Contains(strings.ToLower(cfg.Text), "кноп") {
		t.Fatalf("expected instructional text, got %q", cfg.Text)
	}

	got, ok := h.State.Get(userID)
	if !ok {
		t.Fatal("expected flow state to remain present")
	}
	if got.BlockID != 55 || got.Step != learning.StepReview || got.LastAnswer != "my previous answer" {
		t.Fatalf("state changed unexpectedly: %#v", got)
	}
}

func TestHandlers_HandleLearningMenuSelection_TogglesAIChatMode(t *testing.T) {
	bot := &fakeFlowTelegram{}
	h := newFlowTestHandlers(bot, fakeFlowContentRepository{}, fakeFlowContentService{}, fakeFlowLearningService{}, nil)
	msg := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 42},
		From: &tgbotapi.User{ID: 100},
	}

	h.handleLearningMenuSelection(msg, "ai_chat")
	if !h.aiChatModeEnabled(100) {
		t.Fatal("expected ai chat mode to be enabled")
	}

	h.handleLearningMenuSelection(msg, "ai_chat")
	if h.aiChatModeEnabled(100) {
		t.Fatal("expected ai chat mode to be disabled")
	}
}

func TestHandlers_HandleMessage_AIChatModeRepliesWithAI(t *testing.T) {
	bot := &fakeFlowTelegram{}
	ai := &fakeAIProvider{reply: "AI ответ"}
	h := newFlowTestHandlers(
		bot,
		fakeFlowContentRepository{},
		fakeFlowContentService{},
		fakeFlowLearningService{},
		ai,
	)
	h.setAIChatMode(777, true)

	h.HandleMessage(&tgbotapi.Message{
		Text: "как работает map в go?",
		Chat: &tgbotapi.Chat{ID: 51},
		From: &tgbotapi.User{ID: 777},
	})

	if len(bot.sent) == 0 {
		t.Fatal("expected ai reply message")
	}
	cfg, ok := bot.sent[len(bot.sent)-1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[len(bot.sent)-1])
	}
	if strings.TrimSpace(cfg.Text) != "AI ответ" {
		t.Fatalf("expected ai response text, got %q", cfg.Text)
	}
	if ai.lastInput.Message != "как работает map в go?" {
		t.Fatalf("expected chat message to equal user text, got %q", ai.lastInput.Message)
	}
	if cfg.ParseMode != "" {
		t.Fatalf("expected plain text parse mode, got %q", cfg.ParseMode)
	}
}

func TestHandlers_HandleMessage_AIChatModeSlowReplySendsThinkingFirst(t *testing.T) {
	bot := &fakeFlowTelegram{}
	ai := &fakeAIProvider{
		reply: "AI ответ после задержки",
		delay: 40 * time.Millisecond,
	}
	h := newFlowTestHandlers(
		bot,
		fakeFlowContentRepository{},
		fakeFlowContentService{},
		fakeFlowLearningService{},
		ai,
	)
	h.AIThinkingDelay = 10 * time.Millisecond
	h.setAIChatMode(777, true)

	h.HandleMessage(&tgbotapi.Message{
		Text: "расскажи про map в go",
		Chat: &tgbotapi.Chat{ID: 51},
		From: &tgbotapi.User{ID: 777},
	})

	if len(bot.sent) < 2 {
		t.Fatalf("expected at least 2 messages (thinking + answer), got %d", len(bot.sent))
	}

	first, ok := bot.sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected first sent item to be MessageConfig, got %T", bot.sent[0])
	}
	if strings.TrimSpace(first.Text) != "Думаю..." {
		t.Fatalf("expected thinking message first, got %q", first.Text)
	}

	last, ok := bot.sent[len(bot.sent)-1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected last sent item to be MessageConfig, got %T", bot.sent[len(bot.sent)-1])
	}
	if strings.TrimSpace(last.Text) != "AI ответ после задержки" {
		t.Fatalf("expected final ai answer, got %q", last.Text)
	}
}

func TestHandlers_HandleMessage_AIChatModeRejectsOffTopic(t *testing.T) {
	bot := &fakeFlowTelegram{}
	ai := &fakeAIProvider{reply: "unused"}
	h := newFlowTestHandlers(
		bot,
		fakeFlowContentRepository{},
		fakeFlowContentService{},
		fakeFlowLearningService{},
		ai,
	)
	h.setAIChatMode(777, true)

	h.HandleMessage(&tgbotapi.Message{
		Text: "посоветуй фильм на вечер",
		Chat: &tgbotapi.Chat{ID: 51},
		From: &tgbotapi.User{ID: 777},
	})

	if len(bot.sent) == 0 {
		t.Fatal("expected instructional off-topic reply")
	}
	cfg, ok := bot.sent[len(bot.sent)-1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[len(bot.sent)-1])
	}
	if !strings.Contains(strings.ToLower(cfg.Text), "я отвечаю на it-темы") {
		t.Fatalf("expected off-topic restriction text, got %q", cfg.Text)
	}
	if ai.lastInput.Message != "" {
		t.Fatalf("expected provider not to be called for off-topic request, got %q", ai.lastInput.Message)
	}
}

func TestHandlers_HandleMessage_AIChatModeEnforcesDailyLimit(t *testing.T) {
	bot := &fakeFlowTelegram{}
	ai := &fakeAIProvider{reply: "ok"}
	h := newFlowTestHandlers(
		bot,
		fakeFlowContentRepository{},
		fakeFlowContentService{},
		fakeFlowLearningService{},
		ai,
	)
	h.AIChatDailyLimit = 1
	h.setAIChatMode(777, true)

	first := &tgbotapi.Message{
		Text: "что такое map в go?",
		Chat: &tgbotapi.Chat{ID: 51},
		From: &tgbotapi.User{ID: 777},
	}
	second := &tgbotapi.Message{
		Text: "а что такое slice в go?",
		Chat: &tgbotapi.Chat{ID: 51},
		From: &tgbotapi.User{ID: 777},
	}
	h.HandleMessage(first)
	h.HandleMessage(second)

	if len(bot.sent) < 2 {
		t.Fatalf("expected at least 2 responses, got %d", len(bot.sent))
	}
	cfg, ok := bot.sent[len(bot.sent)-1].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[len(bot.sent)-1])
	}
	if !strings.Contains(strings.ToLower(cfg.Text), "дневного лимита") {
		t.Fatalf("expected daily limit text, got %q", cfg.Text)
	}
}

func TestIsAllowedAIChatTopic(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{in: "привет", want: true},
		{in: "что такое interface в go", want: true},
		{in: "какие возможности у этого бота?", want: true},
		{in: "погода завтра", want: false},
	}
	for _, tt := range tests {
		if got := isAllowedAIChatTopic(tt.in); got != tt.want {
			t.Fatalf("isAllowedAIChatTopic(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseFlowCallbackData_MalformedInputsReturnOkFalse(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "wrong prefix", in: "xxx:next:10"},
		{name: "missing parts", in: "flow:next"},
		{name: "too many parts", in: "flow:next:10:extra"},
		{name: "non numeric id", in: "flow:next:notnum"},
		{name: "zero id", in: "flow:next:0"},
		{name: "negative id", in: "flow:next:-1"},
		{name: "unknown action", in: "flow:unknown:10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := parseFlowCallbackData(tt.in)
			if ok {
				t.Fatalf("expected ok=false for input %q", tt.in)
			}
		})
	}
}
