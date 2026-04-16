package content

import (
	"context"
	"errors"
	"testing"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

func TestBuildBlockPayloadReturnsTheoryBlockContent(t *testing.T) {
	repo := &fakeContentRepository{
		blocks: map[int64]learning.Block{
			10: {
				ID:        10,
				BlockType: learning.BlockTheory,
				Title:     "Hash Tables",
			},
		},
		contents: map[int64]learning.BlockContent{
			10: {
				BlockID:    10,
				TheoryMD:   "Theory text",
				Difficulty: "easy",
				Tags:       []string{"hash-map", "intro"},
			},
		},
	}

	svc := New(repo)

	payload, err := svc.BuildBlockPayload(context.Background(), 10)
	if err != nil {
		t.Fatalf("BuildBlockPayload() error = %v", err)
	}

	if payload.Title != "Hash Tables" {
		t.Fatalf("expected title %q, got %q", "Hash Tables", payload.Title)
	}
	if payload.BlockType != learning.BlockTheory {
		t.Fatalf("expected block type %q, got %q", learning.BlockTheory, payload.BlockType)
	}
	if payload.TheoryMD != "Theory text" {
		t.Fatalf("expected theory text, got %q", payload.TheoryMD)
	}
	if payload.TaskMD != "" {
		t.Fatalf("expected empty task md, got %q", payload.TaskMD)
	}
	if payload.SolutionMD != "" {
		t.Fatalf("expected empty solution md, got %q", payload.SolutionMD)
	}
	if payload.Difficulty != "easy" {
		t.Fatalf("expected difficulty %q, got %q", "easy", payload.Difficulty)
	}
	if len(payload.Tags) != 2 || payload.Tags[0] != "hash-map" || payload.Tags[1] != "intro" {
		t.Fatalf("unexpected tags: %#v", payload.Tags)
	}
}

func TestBuildBlockPayloadUsesLinkedSolutionForTask(t *testing.T) {
	repo := &fakeContentRepository{
		blocks: map[int64]learning.Block{
			20: {
				ID:        20,
				BlockType: learning.BlockTask,
				Title:     "Two Sum",
			},
			21: {
				ID:        21,
				BlockType: learning.BlockSolution,
				Title:     "Two Sum Solution",
			},
		},
		contents: map[int64]learning.BlockContent{
			20: {
				BlockID:    20,
				TaskMD:     "Find two numbers that sum to target.",
				Difficulty: "medium",
				Tags:       []string{"array", "hash-map"},
			},
			21: {
				BlockID:    21,
				SolutionMD: "Use a map from value to index.",
			},
		},
		relations: map[int64][]learning.BlockRelation{
			20: {
				{
					FromBlockID:  20,
					ToBlockID:    21,
					RelationType: learning.RelationTaskSolution,
				},
			},
		},
	}

	svc := New(repo)

	payload, err := svc.BuildBlockPayload(context.Background(), 20)
	if err != nil {
		t.Fatalf("BuildBlockPayload() error = %v", err)
	}

	if payload.TaskMD != "Find two numbers that sum to target." {
		t.Fatalf("expected task text, got %q", payload.TaskMD)
	}
	if payload.SolutionMD != "Use a map from value to index." {
		t.Fatalf("expected linked solution, got %q", payload.SolutionMD)
	}
	if payload.BlockType != learning.BlockTask {
		t.Fatalf("expected block type %q, got %q", learning.BlockTask, payload.BlockType)
	}
}

func TestBuildBlockPayloadDegradesGracefullyWhenLinkedSolutionContentMissing(t *testing.T) {
	repo := &fakeContentRepository{
		blocks: map[int64]learning.Block{
			30: {
				ID:        30,
				BlockType: learning.BlockTask,
				Title:     "Missing solution",
			},
			31: {
				ID:        31,
				BlockType: learning.BlockSolution,
				Title:     "Broken solution",
			},
		},
		contents: map[int64]learning.BlockContent{
			30: {
				BlockID: 30,
				TaskMD:  "Task without embedded solution",
			},
		},
		relations: map[int64][]learning.BlockRelation{
			30: {
				{
					FromBlockID:  30,
					ToBlockID:    31,
					RelationType: learning.RelationTaskSolution,
				},
			},
		},
		contentErrs: map[int64]error{
			31: errors.New("missing linked content"),
		},
	}

	svc := New(repo)

	payload, err := svc.BuildBlockPayload(context.Background(), 30)
	if err != nil {
		t.Fatalf("BuildBlockPayload() error = %v", err)
	}
	if payload.SolutionMD != "" {
		t.Fatalf("expected empty solution due to missing linked content, got %q", payload.SolutionMD)
	}
}

func TestBuildBlockPayloadEmbeddedSolutionPrecedenceOverBrokenRelation(t *testing.T) {
	repo := &fakeContentRepository{
		blocks: map[int64]learning.Block{
			40: {
				ID:        40,
				BlockType: learning.BlockTask,
				Title:     "Task with embedded solution",
			},
			41: {
				ID:        41,
				BlockType: learning.BlockSolution,
				Title:     "Broken linked solution",
			},
		},
		contents: map[int64]learning.BlockContent{
			40: {
				BlockID:    40,
				TaskMD:     "Task text",
				SolutionMD: "Embedded solution wins",
			},
			41: {
				BlockID: 41,
			},
		},
		relations: map[int64][]learning.BlockRelation{
			40: {
				{
					FromBlockID:  40,
					ToBlockID:    41,
					RelationType: learning.RelationTaskSolution,
				},
			},
		},
		contentErrs: map[int64]error{
			41: errors.New("broken linked content"),
		},
	}

	svc := New(repo)

	payload, err := svc.BuildBlockPayload(context.Background(), 40)
	if err != nil {
		t.Fatalf("BuildBlockPayload() error = %v", err)
	}
	if payload.SolutionMD != "Embedded solution wins" {
		t.Fatalf("expected embedded solution to win, got %q", payload.SolutionMD)
	}
}

type fakeContentRepository struct {
	blocks      map[int64]learning.Block
	contents    map[int64]learning.BlockContent
	relations   map[int64][]learning.BlockRelation
	blockErrs   map[int64]error
	contentErrs map[int64]error
}

func (f *fakeContentRepository) GetSectionByCode(context.Context, string) (learning.Section, error) {
	return learning.Section{}, errors.New("not implemented")
}

func (f *fakeContentRepository) GetChapter(context.Context, int64) (learning.Chapter, error) {
	return learning.Chapter{}, errors.New("not implemented")
}

func (f *fakeContentRepository) ListChaptersBySection(context.Context, int64) ([]learning.Chapter, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeContentRepository) ListChapterBlocks(context.Context, int64) ([]learning.Block, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeContentRepository) GetBlock(_ context.Context, blockID int64) (learning.Block, error) {
	if err := f.blockErrs[blockID]; err != nil {
		return learning.Block{}, err
	}

	block, ok := f.blocks[blockID]
	if !ok {
		return learning.Block{}, errors.New("block not found")
	}

	return block, nil
}

func (f *fakeContentRepository) GetBlockContent(_ context.Context, blockID int64) (learning.BlockContent, error) {
	if err := f.contentErrs[blockID]; err != nil {
		return learning.BlockContent{}, err
	}

	content, ok := f.contents[blockID]
	if !ok {
		return learning.BlockContent{}, errors.New("content not found")
	}

	return content, nil
}

func (f *fakeContentRepository) ListBlockRelations(_ context.Context, blockID int64) ([]learning.BlockRelation, error) {
	return f.relations[blockID], nil
}
