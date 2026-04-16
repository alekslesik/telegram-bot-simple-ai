package learningflow

import (
	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

type Action string

const (
	ActionNext         Action = "next"
	ActionSubmitAnswer Action = "submit_answer"
	ActionSkipAnswer   Action = "skip_answer"
	ActionSkipReview   Action = "skip_review"
)

type Service struct {
}

func New() *Service {
	return &Service{}
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
		return current
	case learning.StepAnswer:
		// Any user action in the answer step is treated as "answer provided" and the flow moves to review.
		// (Skipping answer is handled from StepTask.)
		return learning.StepReview
	case learning.StepReview:
		if action == ActionSkipReview {
			return learning.StepSolution
		}
		return current
	case learning.StepSolution:
		if action == ActionNext {
			return learning.StepTask
		}
		return current
	default:
		return current
	}
}
