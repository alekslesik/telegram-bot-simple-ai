package learning

import "testing"

func TestValidFlowSteps(t *testing.T) {
	steps := []FlowStep{StepTheory, StepTask, StepAnswer, StepReview, StepSolution}
	if len(steps) != 5 {
		t.Fatalf("expected 5 flow steps, got %d", len(steps))
	}

	want := []FlowStep{"theory", "task", "answer", "review", "solution"}
	for i := range steps {
		if steps[i] != want[i] {
			t.Fatalf("expected step %d to be %q, got %q", i, want[i], steps[i])
		}
	}
}
