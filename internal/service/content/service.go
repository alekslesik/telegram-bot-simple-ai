package content

import (
	"context"
	"fmt"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
	"github.com/alekslesik/telegram-bot-simple/internal/repository"
)

type Service struct {
	repo repository.ContentRepository
}

type BlockPayload struct {
	Title      string
	BlockType  learning.BlockType
	TheoryMD   string
	TaskMD     string
	SolutionMD string
	Difficulty string
	Tags       []string
}

func New(repo repository.ContentRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) BuildBlockPayload(ctx context.Context, blockID int64) (BlockPayload, error) {
	block, err := s.repo.GetBlock(ctx, blockID)
	if err != nil {
		return BlockPayload{}, fmt.Errorf("get block %d: %w", blockID, err)
	}

	content, err := s.repo.GetBlockContent(ctx, blockID)
	if err != nil {
		return BlockPayload{}, fmt.Errorf("get block content %d: %w", blockID, err)
	}

	payload := BlockPayload{
		Title:      block.Title,
		BlockType:  block.BlockType,
		TheoryMD:   content.TheoryMD,
		TaskMD:     content.TaskMD,
		SolutionMD: content.SolutionMD,
		Difficulty: content.Difficulty,
		Tags:       append([]string(nil), content.Tags...),
	}

	if block.BlockType != learning.BlockTask {
		return payload, nil
	}

	relations, err := s.repo.ListBlockRelations(ctx, blockID)
	if err != nil {
		return BlockPayload{}, fmt.Errorf("list block relations %d: %w", blockID, err)
	}

	relation := firstTaskSolutionRelation(relations)
	if relation == nil {
		return payload, nil
	}

	linkedContent, err := s.repo.GetBlockContent(ctx, relation.ToBlockID)
	if err != nil {
		return BlockPayload{}, fmt.Errorf("get linked solution content %d for block %d: %w", relation.ToBlockID, blockID, err)
	}

	if linkedContent.SolutionMD != "" {
		payload.SolutionMD = linkedContent.SolutionMD
	}

	return payload, nil
}

func firstTaskSolutionRelation(relations []learning.BlockRelation) *learning.BlockRelation {
	for i := range relations {
		if relations[i].RelationType == learning.RelationTaskSolution {
			return &relations[i]
		}
	}

	return nil
}
