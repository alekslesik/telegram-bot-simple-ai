package learningflow

import (
	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
	"github.com/alekslesik/telegram-bot-simple/internal/repository"
)

type Action string

const (
	ActionNext         Action = "next"
	ActionSubmitAnswer Action = "submit_answer"
	ActionSkipAnswer   Action = "skip_answer"
	ActionSkipReview   Action = "skip_review"
)

type Service struct {
	contentRepo  repository.ContentRepository
	progressRepo repository.ProgressRepository
}

func New(contentRepo repository.ContentRepository, progressRepo repository.ProgressRepository) *Service {
	return &Service{
		contentRepo:  contentRepo,
		progressRepo: progressRepo,
	}
}

func (s *Service) NextStep(blockType learning.BlockType, current learning.FlowStep, action Action) learning.FlowStep {
	if blockType == learning.BlockTheory || current == learning.StepTheory {
		return learning.StepTask
	}

	switch current {
	case learning.StepTask:
		if action == ActionSkipAnswer {
			return learning.StepSolution
		}
		if action == ActionSubmitAnswer {
			return learning.StepReview
		}
		return learning.StepSolution
	case learning.StepAnswer:
		return learning.StepReview
	case learning.StepReview:
		return learning.StepSolution
	default:
		return learning.StepSolution
	}
}
