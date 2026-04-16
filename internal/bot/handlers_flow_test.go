package bot

import (
	"context"
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
	blocks        map[int64]learning.Block
	chapterBlocks map[int64][]learning.Block
}

func (f fakeFlowContentRepository) GetSectionByCode(context.Context, string) (learning.Section, error) {
	return learning.Section{}, nil
}

func (f fakeFlowContentRepository) GetChapter(context.Context, int64) (learning.Chapter, error) {
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
