package learning

import "testing"

func TestValidFlowSteps(t *testing.T) {
	steps := []FlowStep{StepTheory, StepTask, StepAnswer, StepReview, StepSolution}
	if len(steps) != 5 {
		t.Fatalf("expected 5 flow steps, got %d", len(steps))
	}
}
