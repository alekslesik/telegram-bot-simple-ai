package bot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
	"github.com/alekslesik/telegram-bot-simple/internal/repository"
	contentservice "github.com/alekslesik/telegram-bot-simple/internal/service/content"
	"github.com/alekslesik/telegram-bot-simple/internal/service/learningflow"
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

func newFlowTestHandlers(bot TelegramClient, repo repository.ContentRepository, contentSvc flowContentBuilder, learningSvc flowNavigator) *Handlers {
	return &Handlers{
		Bot:      bot,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})),
		Content:  contentSvc,
		Learning: learningSvc,
		Repo:     repo,
		State:    newInMemoryFlowStateStore(),
	}
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

	h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepSolution})

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

	h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepTask})

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

	h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepTask})
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

	required := []string{"Введение", "Алгоритмы по порядку", "Рандом задача", "Мой прогресс", "Настройки"}
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

		h := newFlowTestHandlers(bot, repo, contentSvc, fakeFlowLearningService{next: learning.StepTask})

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

	h := newFlowTestHandlers(bot, repo, fakeFlowContentService{}, fakeFlowLearningService{next: learning.StepSolution})
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
