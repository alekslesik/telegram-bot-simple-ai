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

func TestProgressAndSessionStates(t *testing.T) {
	if ProgressStatusInProgress != "in_progress" {
		t.Fatalf("expected in-progress status, got %q", ProgressStatusInProgress)
	}
	if ProgressStatusCompleted != "completed" {
		t.Fatalf("expected completed status, got %q", ProgressStatusCompleted)
	}
	if SessionStateActive != "active" {
		t.Fatalf("expected active session state, got %q", SessionStateActive)
	}
	if SessionStateClosed != "closed" {
		t.Fatalf("expected closed session state, got %q", SessionStateClosed)
	}
}
