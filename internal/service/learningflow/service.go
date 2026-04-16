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
	if current == learning.StepTheory {
		// Theory-only card should advance only on an explicit "next" action.
		// For robustness, unknown actions keep the user on the same step.
		if action == ActionNext {
			return learning.StepTask
		}
		return current
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
		switch action {
		case ActionSubmitAnswer:
			return learning.StepReview
		case ActionSkipAnswer:
			return learning.StepSolution
		default:
			return current
		}
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
