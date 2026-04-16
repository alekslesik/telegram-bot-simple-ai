package learningflow

import (
	"testing"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
)

func TestNextFromTaskSkipAnswerMovesToSolution(t *testing.T) {
	svc := New()

	next := svc.NextStep(learning.BlockTask, learning.StepTask, ActionSkipAnswer)

	if next != learning.StepSolution {
		t.Fatalf("expected %q, got %q", learning.StepSolution, next)
	}
}

func TestNextStepTransitions(t *testing.T) {
	svc := New()

	tests := []struct {
		name      string
		blockType learning.BlockType
		current   learning.FlowStep
		action    Action
		want      learning.FlowStep
	}{
		{
			name:      "theory always moves to task",
			blockType: learning.BlockTheory,
			current:   learning.StepTheory,
			action:    ActionSkipAnswer,
			want:      learning.StepTask,
		},
		{
			name:      "task submit answer moves to review",
			blockType: learning.BlockTask,
			current:   learning.StepTask,
			action:    ActionSubmitAnswer,
			want:      learning.StepReview,
		},
		{
			name:      "review skip moves to solution",
			blockType: learning.BlockTask,
			current:   learning.StepReview,
			action:    ActionSkipReview,
			want:      learning.StepSolution,
		},
		{
			name:      "solution next moves to task",
			blockType: learning.BlockTask,
			current:   learning.StepSolution,
			action:    ActionNext,
			want:      learning.StepTask,
		},
		{
			name:      "unknown action on task keeps step",
			blockType: learning.BlockTask,
			current:   learning.StepTask,
			action:    Action("unknown"),
			want:      learning.StepTask,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.NextStep(tt.blockType, tt.current, tt.action)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
